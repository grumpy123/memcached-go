package gonet

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"os"
	"strconv"
	"sync"
	"testing"

	"github.com/stretchr/testify/suite"
)

func must(f func() error) {
	if err := f(); err != nil {
		panic(err)
	}
}

type ListenerSuite struct {
	suite.Suite
}

func TestListenerSuite(t *testing.T) {
	suite.Run(t, new(ListenerSuite))
}

func (s *ListenerSuite) setupListener(hf ConnectionHandler) *Listener {
	l := NewListener(0, hf)
	err := l.Start(context.Background())
	s.Require().Nil(err)

	return l
}

func (s *ListenerSuite) intEnv(env string, defaultValue int) int {
	strValue := os.Getenv(env)
	if strValue == "" {
		return defaultValue
	}

	i, err := strconv.Atoi(strValue)
	s.Require().Nil(err)
	return i
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
	hf := NewSingleConnectionTestHandler()
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
	hf := &ConcurrentConnectionsTestHandler{}
	thf := WithTracking(hf)
	l := s.setupListener(thf)

	workers := s.intEnv("TEST_CONCURRENT_WORKERS", 5)
	iterations := s.intEnv("TEST_CONCURRENT_ITERATIONS", 10)
	wg := &sync.WaitGroup{}
	wg.Add(workers)

	clientTest := func(worker int) {
		conn, err := net.Dial("tcp", l.Address().String())
		s.Require().Nil(err)

		for i := 1; i <= iterations; i++ {
			text := fmt.Sprintf("hello %d from worker %d\n", i, worker)
			_, err = conn.Write([]byte(text))
			s.Require().Nil(err)
			reader := bufio.NewReader(conn)
			resText, err := reader.ReadString('\n')
			s.Require().Nil(err)
			s.Assert().Equal("same to you: "+text, resText)
		}

		s.Require().Nil(conn.Close())
		wg.Done()
	}
	for i := 1; i <= workers; i++ {
		go clientTest(1)
	}

	wg.Wait()
	s.Require().Nil(l.Close())
	<-thf.Done()
}

// todo: test reading from socket doesn't block accepting (parallel connections, may need to wait with N semaphore)
// todo: implement and test closing listener first, to confirm closing done channel wraps stuff up
//       need to consider draining pending responses, so can't just close connections?
