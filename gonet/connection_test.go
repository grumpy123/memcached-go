package gonet

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

type ConnectionSuite struct {
	BaseSuite
}

func TestConnectionSuite(t *testing.T) {
	suite.Run(t, new(ConnectionSuite))
}

type TestMessage struct {
	inText  string
	outText string
	outTS   time.Time
}

func (t *TestMessage) WriteRequest(w *bufio.Writer) error {
	realInText := strings.Replace(t.inText, "\n", "-", -1)
	_, err := w.WriteString(realInText + "\n")
	return err
}

func (t *TestMessage) ReadResponse(r *bufio.Reader) error {
	strTS, err := r.ReadString('\n')
	if err != nil {
		return err
	}
	outTS, err := strconv.ParseInt(strings.TrimSpace(strTS), 10, 64)
	if err != nil {
		return err
	}
	t.outTS = time.UnixMicro(outTS)

	t.outText, err = r.ReadString('\n')
	if err != nil {
		return err
	}

	t.outText = t.outText[:len(t.outText)-1]
	return nil
}

func (s *ConnectionSuite) TestConnection() {
	h := &TestRequestHandler{}
	th := WithTracking(NewServerFactory(h))

	l := s.setupListener(th)

	conn, err := NewConnection(l.Address().String())
	s.Require().Nil(err)

	sendTime := time.Now()
	msg := &TestMessage{inText: "hello world"}
	err = conn.Call(context.Background(), msg)
	s.Require().Nil(err)

	s.Assert().LessOrEqual(sendTime, msg.outTS)
	s.Assert().GreaterOrEqual(time.Now(), msg.outTS)
	s.Assert().Equal(msg.outText, msg.inText)

	s.Require().Nil(l.Close())
	<-th.Done()
}

func (s *ConnectionSuite) TestConnectionConcurrency() {
	h := &TestRequestHandler{}
	th := WithTracking(NewServerFactory(h))

	l := s.setupListener(th)

	conn, err := NewConnection(l.Address().String())
	s.Require().Nil(err)

	workers := s.intEnv("TEST_CONCURRENT_WORKERS", 10)
	iterations := s.intEnv("TEST_CONCURRENT_ITERATIONS", 5)
	s.testConnections(conn, workers, iterations)

	conn.Close()

	<-th.Done()
	s.Require().Nil(l.Close())
}

func (s *ConnectionSuite) TestConnectionClose() {
	h := &TestRequestHandler{}
	th := WithTracking(NewServerFactory(h))

	l := s.setupListener(th)

	conn, err := NewConnection(l.Address().String())
	s.Require().Nil(err)

	workers := s.intEnv("TEST_CONCURRENT_WORKERS", 5)
	iterations := s.intEnv("TEST_CONCURRENT_ITERATIONS", 2)
	s.testConnections(conn, workers, iterations)

	s.Require().Nil(l.Close())
	// Wait until the server loops completes and closes the connection
	<-th.Done()

	// todo: switch to Send when it's implemented
	err = conn.Call(context.Background(), &TestMessage{inText: "Expected to fail gracefully"})
	s.Require().Error(err)
	s.Assert().ErrorContains(err, "receiving error")
	err = conn.Call(context.Background(), &TestMessage{inText: "Expected to fail gracefully too"})
	s.Require().Error(err)
	if !errors.Is(err, ErrConnClosed) {
		s.Assert().ErrorContains(err, "sending error")
		err = conn.Call(context.Background(), &TestMessage{inText: "Expected to fail gracefully as well"})
	}
	s.Require().Error(err)
	s.Assert().ErrorIs(err, ErrConnClosed)
	for i := 0; i < 10; i++ {
		err = conn.Call(context.Background(), &TestMessage{inText: "Expected to fail gracefully, forever and always"})
		s.Require().Error(err)
		s.Assert().ErrorIs(err, ErrConnClosed)
	}
	conn.Close()
}

func (s *ConnectionSuite) testConnections(conn *Connection, workers int, iterations int) {
	wg := &sync.WaitGroup{}
	wg.Add(workers)

	clientTest := func(worker int) {
		for i := 1; i <= iterations; i++ {
			text := fmt.Sprintf("hello %d from worker %d", i, worker)
			msg := &TestMessage{inText: text}
			startTime := time.Now()
			err := conn.Call(context.Background(), msg)
			s.Assert().Nil(err)

			s.Assert().LessOrEqual(startTime, msg.outTS)
			s.Assert().GreaterOrEqual(time.Now(), msg.outTS)
			s.Assert().Equal(text, msg.outText)
		}
		wg.Done()
	}
	for i := 1; i <= workers; i++ {
		go clientTest(i)
	}
	wg.Wait()
}
