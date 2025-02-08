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
	error         error
	isDoneReading sync.WaitGroup
	isClosed      sync.WaitGroup
}

func NewTestHandlerFactory() *TestHandlerFactory {
	hf := &TestHandlerFactory{
		isDoneReading: sync.WaitGroup{},
		isClosed:      sync.WaitGroup{},
	}
	hf.isDoneReading.Add(1)
	hf.isClosed.Add(1)
	return hf
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func (t *TestHandlerFactory) New(c net.Conn, done <-chan struct{}) {
	reader := bufio.NewReader(c)
	t.text, t.error = reader.ReadString('\n')
	t.isDoneReading.Done()

	<-done
	must(c.Close())
	t.isClosed.Done()
}

type ListenerSuite struct {
	suite.Suite
}

func TestListenerSuite(t *testing.T) {
	suite.Run(t, new(ListenerSuite))
}

func (s *ListenerSuite) TestConnectionOpenAndClose() {
	h := NewTestHandlerFactory()
	l := NewListener(0, h)
	err := l.Start(context.Background())
	s.Require().Nil(err)

	conn, err := net.Dial("tcp", l.Address().String())
	s.Require().Nil(err)

	_, err = conn.Write([]byte("hello\n"))
	s.Require().Nil(err)

	h.isDoneReading.Wait()
	s.Assert().Nil(h.error)
	s.Assert().Equal("hello\n", h.text)

	s.Require().Nil(conn.Close())
	s.Require().Nil(l.Close())
	h.isClosed.Wait()
}
