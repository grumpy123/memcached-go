package gonet

import (
	"bufio"
	"context"
	"net"
	"sync"
	"testing"

	"github.com/stretchr/testify/suite"
)

type TestHandlerFactory struct {
	text          string
	readErr       error
	writeErr      error
	isDoneReading sync.WaitGroup
	isDoneWriting sync.WaitGroup
}

func NewTestHandlerFactory(expectedClients int) *TestHandlerFactory {
	hf := &TestHandlerFactory{
		isDoneReading: sync.WaitGroup{},
		isDoneWriting: sync.WaitGroup{},
	}
	hf.isDoneReading.Add(expectedClients)
	hf.isDoneWriting.Add(expectedClients)
	return hf
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func (hf *TestHandlerFactory) New(c net.Conn, done <-chan struct{}) {
	reader := bufio.NewReader(c)
	writer := bufio.NewWriter(c)
	hf.text, hf.readErr = reader.ReadString('\n')
	hf.isDoneReading.Done()
	_, hf.writeErr = writer.WriteString("world\n")
	if hf.writeErr == nil {
		hf.writeErr = writer.Flush()
	}
	hf.isDoneWriting.Done()

	<-done
	must(c.Close())
}

type ListenerSuite struct {
	suite.Suite
}

func TestListenerSuite(t *testing.T) {
	suite.Run(t, new(ListenerSuite))
}

func (s *ListenerSuite) setupListener(hf HandlerFactory) *Listener {
	l := NewListener(0, hf)
	err := l.Start(context.Background())
	s.Require().Nil(err)

	return l
}

func (s *ListenerSuite) TestConnectionOpenAndClose() {
	hf := NewTestHandlerFactory(1)
	thf := WithTracking(hf)
	l := s.setupListener(thf)

	conn, err := net.Dial("tcp", l.Address().String())
	s.Require().Nil(err)

	_, err = conn.Write([]byte("hello\n"))
	s.Require().Nil(err)

	hf.isDoneReading.Wait()
	s.Assert().Nil(hf.readErr)
	s.Assert().Equal("hello\n", hf.text)

	hf.isDoneWriting.Wait()
	reader := bufio.NewReader(conn)
	resText, err := reader.ReadString('\n')
	s.Require().Nil(err)
	s.Assert().Equal("world\n", resText)
	s.Assert().Nil(hf.writeErr)

	s.Require().Nil(conn.Close())
	s.Require().Nil(l.Close())

	<-thf.Done()
}

// todo: stress test connection creation and destruction
// todo: test reading from socket doesn't block accepting (parallel connections, may need to wait with N semaphore)
