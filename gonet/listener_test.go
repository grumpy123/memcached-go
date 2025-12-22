package gonet

import (
	"bufio"
	"fmt"
	"net"
	"sync"
	"testing"

	"github.com/stretchr/testify/suite"
)

type ListenerSuite struct {
	BaseSuite
}

func TestListenerSuite(t *testing.T) {
	suite.Run(t, new(ListenerSuite))
}

type SingleConnectionTestHandler struct {
	text          string
	readErr       error
	writeErr      error
	isDoneReading sync.WaitGroup
	isDoneWriting sync.WaitGroup
}

func NewSingleConnectionTestHandler() *SingleConnectionTestHandler {
	hf := &SingleConnectionTestHandler{
		isDoneReading: sync.WaitGroup{},
		isDoneWriting: sync.WaitGroup{},
	}
	hf.isDoneReading.Add(1)
	hf.isDoneWriting.Add(1)
	return hf
}

func (sc *SingleConnectionTestHandler) New(c net.Conn, done <-chan struct{}) {
	defer must(c.Close)

	reader := bufio.NewReader(c)
	writer := bufio.NewWriter(c)
	sc.text, sc.readErr = reader.ReadString('\n')
	sc.isDoneReading.Done()
	_, sc.writeErr = writer.WriteString("world\n")
	if sc.writeErr == nil {
		sc.writeErr = writer.Flush()
	}
	sc.isDoneWriting.Done()

	<-done
}

func (s *ListenerSuite) TestSingleConnection() {
	h := NewSingleConnectionTestHandler()
	th := WithTracking(h)
	l := s.SetupListener(th)

	conn, err := net.Dial("tcp", l.Address().String())
	s.Require().NoError(err)

	_, err = conn.Write([]byte("hello\n"))
	s.Require().NoError(err)

	h.isDoneReading.Wait()
	s.Assert().NoError(h.readErr)
	s.Assert().Equal("hello\n", h.text)

	h.isDoneWriting.Wait()
	reader := bufio.NewReader(conn)
	resText, err := reader.ReadString('\n')
	s.Require().NoError(err)
	s.Assert().Equal("world\n", resText)
	s.Assert().NoError(h.writeErr)

	s.Require().NoError(conn.Close())
	s.Require().NoError(l.Close())

	<-th.Done()
}

type ConcurrentConnectionsTestHandler struct{}

func (cc *ConcurrentConnectionsTestHandler) New(conn net.Conn, done <-chan struct{}) {
	defer must(conn.Close)

	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)
	for {
		readText, err := reader.ReadString('\n')
		if err != nil {
			return
		}
		_, err = writer.WriteString(fmt.Sprintf("same to you: %s", readText))
		if err != nil {
			return
		}
		err = writer.Flush()
		if err != nil {
			return
		}
	}
}

func (s *ListenerSuite) TestConcurrentConnections() {
	h := &ConcurrentConnectionsTestHandler{}
	th := WithTracking(h)
	l := s.SetupListener(th)

	workers := s.IntEnv("TEST_CONCURRENT_WORKERS", 5)
	iterations := s.IntEnv("TEST_CONCURRENT_ITERATIONS", 10)
	wg := &sync.WaitGroup{}
	wg.Add(workers)

	clientTest := func(worker int) {
		conn, err := net.Dial("tcp", l.Address().String())
		s.Require().NoError(err)

		for i := 1; i <= iterations; i++ {
			text := fmt.Sprintf("hello %d from worker %d\n", i, worker)
			_, err = conn.Write([]byte(text))
			s.Require().NoError(err)
			reader := bufio.NewReader(conn)
			resText, err := reader.ReadString('\n')
			s.Require().NoError(err)
			s.Assert().Equal("same to you: "+text, resText)
		}

		s.Require().NoError(conn.Close())
		wg.Done()
	}
	for i := 1; i <= workers; i++ {
		go clientTest(i)
	}

	wg.Wait()
	s.Require().NoError(l.Close())
	<-th.Done()
}

// todo: test reading from socket doesn't block accepting (parallel connections, may need to wait with N semaphore)
// todo: implement and test closing listener first, to confirm closing done channel wraps stuff up
//       need to consider draining pending responses, so can't just close connections?
