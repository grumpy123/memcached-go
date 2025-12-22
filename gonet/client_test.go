package gonet

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

type ClientSuite struct {
	BaseSuite
}

func TestClientSuite(t *testing.T) {
	suite.Run(t, new(ClientSuite))
}

func (s *ClientSuite) TestClient() {
	h := &TestRequestHandler{}
	th := WithTracking(NewServerFactory(h))

	l := s.setupListener(th)

	cli, err := NewClient(l.Address().String(), 1, 1)
	s.Require().NoError(err)

	sendTime := s.nowUnixMicro()
	msg := &TestMessage{inText: "hello world"}
	err = cli.Call(context.Background(), msg)
	s.Require().NoError(err)

	s.Assert().LessOrEqual(sendTime, msg.outTS)
	s.Assert().GreaterOrEqual(time.Now(), msg.outTS)
	s.Assert().Equal(msg.outText, msg.inText)

	s.Require().NoError(l.Close())
	<-th.Done()
}

func (s *ClientSuite) TestClientConcurrency() {
	h := &TestRequestHandler{}
	th := WithTracking(NewServerFactory(h))

	l := s.setupListener(th)

	cli, err := NewClient(l.Address().String(), 3, 10)
	s.Require().NoError(err)

	workers := s.intEnv("TEST_CONCURRENT_WORKERS", 10)
	iterations := s.intEnv("TEST_CONCURRENT_ITERATIONS", 5)
	s.testClients(cli, workers, iterations)

	cli.Close()

	<-th.Done()
	s.Require().NoError(l.Close())
}

func (s *ClientSuite) TestClientWithErrors() {
	h := &TestRequestHandler{}
	th := WithTracking(NewServerFactory(h))

	l := s.setupListener(th)

	maxConns := s.intEnv("TEST_MAX_CONCURRENT_CONNECTIONS", 10)
	cli, err := NewClient(l.Address().String(), 3, maxConns)
	s.Require().NoError(err)

	s.Equal(3, numConnections(cli))

	workers := s.intEnv("TEST_CONCURRENT_WORKERS", 10)
	iterations := s.intEnv("TEST_CONCURRENT_ITERATIONS", 5)

	s.testClients(cli, workers, iterations)

	s.testErrors(cli, workers, iterations)
	s.LessOrEqual(numConnections(cli), 3) // Usually just 0 or 1, but it's possible to have more

	s.testClients(cli, workers, iterations)

	cli.Close()

	<-th.Done()
	s.Require().NoError(l.Close())
}

func (s *ClientSuite) TestClientMaxConnections() {
	h := &TestRequestHandler{}
	th := WithTracking(NewServerFactory(h))

	l := s.setupListener(th)

	cli, err := NewClient(l.Address().String(), 0, 3)
	s.Require().NoError(err)

	workers := 20
	iterations := 50

	s.Equal(0, numConnections(cli))

	s.testClients(cli, workers, iterations)

	s.Equal(3, numConnections(cli))

	cli.Close()

	<-th.Done()
	s.Require().NoError(l.Close())
}

// todo: Need to define the expected behavior on empty pool first, fast failure or timeout, and then test it here
//func (s *ClientSuite) TestClientClose() {
//	h := &TestRequestHandler{}
//	th := WithTracking(NewServerFactory(h))
//
//	l := s.setupListener(th)
//
//	cli, err := NewClient(l.Address().String(), 1, 5)
//	s.Require().NoError(err)
//
//	workers := s.intEnv("TEST_CONCURRENT_WORKERS", 5)
//	iterations := s.intEnv("TEST_CONCURRENT_ITERATIONS", 2)
//	s.testClients(cli, workers, iterations)
//
//	s.Require().NoError(l.Close())
//	// Wait until the server loops completes and closes the connection
//	<-th.Done()
//
//	// validation here
//
//	cli.Close()
//}

func (s *ClientSuite) testClients(cli *Client, workers int, iterations int) {
	wg := &sync.WaitGroup{}
	wg.Add(workers)

	clientTest := func(worker int) {
		for i := 1; i <= iterations; i++ {
			text := fmt.Sprintf("hello %d from worker %d", i, worker)
			msg := &TestMessage{inText: text}
			sendTime := s.nowUnixMicro()
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			err := cli.Call(ctx, msg)
			s.Assert().NoError(err)

			s.Assert().LessOrEqual(sendTime, msg.outTS)
			s.Assert().GreaterOrEqual(time.Now(), msg.outTS)
			s.Assert().Equal(text, msg.outText)
		}
		wg.Done()
	}
	for i := 1; i <= workers; i++ {
		go clientTest(i)
	}
	wg.Wait()

	// Invariant, we can check it all the time
	s.LessOrEqual(numConnections(cli), cli.maxCons)
}

func (s *ClientSuite) testErrors(cli *Client, workers int, iterations int) {
	wg := &sync.WaitGroup{}
	wg.Add(workers)

	clientTest := func(worker int) {
		for i := 1; i <= iterations; i++ {
			text := fmt.Sprintf("err:error %d from worker %d", i, worker)
			msg := &TestMessage{inText: text}
			err := cli.Call(context.Background(), msg)
			s.Error(err)
		}
		wg.Done()
	}
	for i := 1; i <= workers; i++ {
		go clientTest(i)
	}
	wg.Wait()
}

func numConnections(cli *Client) int {
	// Avoid triggering data race in test.
	// Note: this is only safe in test scenarios, when we know there are no in-flight requests which might get stuck when
	// grabbing this lock prevents pool replenishing.
	cli.connLock.Lock()
	defer cli.connLock.Unlock()

	return len(cli.conns)
}
