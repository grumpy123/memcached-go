package memcached_go

import (
	"context"
	"errors"
	"fmt"
	"memcached-go/testutil"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

type ClientSuite struct {
	testutil.BaseSuite
}

func TestClientSuite(t *testing.T) {
	suite.Run(t, new(ClientSuite))
}

func (s *ClientSuite) TestClientConcurrency() {
	mmcAddr := s.StrEnv("MEMCACHED_ADDR", "")
	if mmcAddr == "" {
		s.T().Skip("MEMCACHED_ADDR not set, skipping test")
		return
	}
	conns := s.IntEnv("TEST_CONNECTIONS", 5)
	workers := s.IntEnv("TEST_CONCURRENT_WORKERS", 10)
	iterations := s.IntEnv("TEST_CONCURRENT_ITERATIONS", 50)

	cli, err := NewClient(mmcAddr, 0, conns)
	s.Require().NoError(err)

	for i := 1; i <= iterations; i++ {
		key := fmt.Sprintf("test-%d", i)
		val := []byte(fmt.Sprintf("value-%d-blahblahblah", i))
		err = cli.Set(context.Background(), key, val)
		s.Require().NoError(err)
	}

	wg := &sync.WaitGroup{}
	wg.Add(workers)

	clientTest := func(worker int) {
		for i := 1; i <= iterations; i++ {
			s.testIteration(cli, worker, i)
			s.testTimeout(cli, i)
		}
		wg.Done()
	}
	for i := 1; i <= workers; i++ {
		go clientTest(i)
	}
	wg.Wait()
	cli.Close()
}

func (s *ClientSuite) testIteration(cli *Client, worker, i int) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	key := fmt.Sprintf("test-%d", i)
	val := []byte(fmt.Sprintf("value-%d-blahblahblah", i))

	got, err := cli.Get(ctx, key)
	s.Assert().NoError(err)
	s.Assert().Equal(val, got)

	keyMiss := fmt.Sprintf("miss-%d", i)
	got, err = cli.Get(ctx, keyMiss)
	s.Assert().NoError(err)
	s.Assert().Nil(got)

	key = fmt.Sprintf("test-%d-worker-%d", i, worker)
	val = []byte(fmt.Sprintf("value-%d-blahblahblah-worker-%d", i, worker))

	err = cli.Set(ctx, key, val)
	s.Assert().NoError(err)

	got, err = cli.Get(ctx, key)
	s.Assert().NoError(err)
	s.Assert().Equal(val, got)
}

func (s *ClientSuite) testTimeout(cli *Client, i int) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Millisecond)
	defer cancel()

	key := fmt.Sprintf("test-%d", i)
	val := []byte(fmt.Sprintf("value-%d-blahblahblah", i))
	for {
		got, err := cli.Get(ctx, key)
		if err != nil && errors.Is(err, context.DeadlineExceeded) {
			// Timeout is sometimes allowed
			return
		}

		s.Assert().NoError(err)
		s.Assert().Equal(val, got)
	}
}
