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
	s.Require().Nil(err)

	sendTime := time.Now()
	msg := &TestMessage{inText: "hello world"}
	err = cli.Call(context.Background(), msg)
	s.Require().Nil(err)

	s.Assert().LessOrEqual(sendTime, msg.outTS)
	s.Assert().GreaterOrEqual(time.Now(), msg.outTS)
	s.Assert().Equal(msg.outText, msg.inText)

	s.Require().Nil(l.Close())
	<-th.Done()
}

func (s *ClientSuite) TestConnectionConcurrency() {
	h := &TestRequestHandler{}
	th := WithTracking(NewServerFactory(h))

	l := s.setupListener(th)

	cli, err := NewClient(l.Address().String(), 5, 5)
	s.Require().Nil(err)

	workers := s.intEnv("TEST_CONCURRENT_WORKERS", 10)
	iterations := s.intEnv("TEST_CONCURRENT_ITERATIONS", 5)
	s.testClients(cli, workers, iterations)

	cli.Close()

	<-th.Done()
	s.Require().Nil(l.Close())
}

func (s *ClientSuite) testClients(cli *Client, workers int, iterations int) {
	wg := &sync.WaitGroup{}
	wg.Add(workers)

	clientTest := func(worker int) {
		for i := 1; i <= iterations; i++ {
			text := fmt.Sprintf("hello %d from worker %d", i, worker)
			msg := &TestMessage{inText: text}
			startTime := time.Now()
			err := cli.Call(context.Background(), msg)
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
