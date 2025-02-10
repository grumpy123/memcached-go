package gonet

import (
	"bufio"
	"fmt"
	"github.com/stretchr/testify/suite"
	"net"
	"strconv"
	"sync"
	"testing"
	"time"
)

type ServerSuite struct {
	BaseSuite
}

func TestServerSuite(t *testing.T) {
	suite.Run(t, new(ServerSuite))
}

func (t *TestRequest) Handle() {
	t.ts = time.Now()
	// todo: inject things here
}

func (t *TestRequest) WriteResponse(writer *bufio.Writer) error {
	_, err := writer.WriteString(fmt.Sprintf("%d\n%s", t.ts.UnixNano(), t.input))
	if err != nil {
		return err
	}
	return nil
}

type TestRequestHandler struct{}

type TestRequest struct {
	input string
	ts    time.Time
}

func (t *TestRequestHandler) ReadRequest(reader *bufio.Reader) (Request, error) {
	input, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}
	return &TestRequest{input: input}, nil
}

func (s *ServerSuite) TestServer() {
	h := &TestRequestHandler{}
	th := WithTracking(NewServerFactory(h))

	l := s.setupListener(th)

	conn, err := net.Dial("tcp", l.Address().String())
	s.Require().Nil(err)

	_, err = conn.Write([]byte("hello\n"))
	s.Require().Nil(err)

	reader := bufio.NewReader(conn)

	resTS, err := reader.ReadString('\n')
	s.Require().Nil(err)
	s.Assert().NotEmpty(resTS)
	ts, err := strconv.ParseInt(resTS, 10, 64)
	s.Assert().GreaterOrEqual(time.Now().Add(-time.Minute).UnixNano(), ts)
	s.Assert().GreaterOrEqual(time.Now().UnixNano(), ts)

	resText, err := reader.ReadString('\n')
	s.Require().Nil(err)
	s.Assert().Equal("hello\n", resText)

	s.Require().Nil(l.Close())
	s.Require().Nil(conn.Close())
	<-th.Done()
}

func (s *ServerSuite) TestServerConcurrency() {
	h := &TestRequestHandler{}
	th := WithTracking(NewServerFactory(h))
	l := s.setupListener(th)

	workers := s.intEnv("TEST_CONCURRENT_WORKERS", 5)
	iterations := s.intEnv("TEST_CONCURRENT_ITERATIONS", 10)
	wg := &sync.WaitGroup{}
	wg.Add(workers)

	clientTest := func(worker int) {
		conn, err := net.Dial("tcp", l.Address().String())
		s.Require().Nil(err)
		reader := bufio.NewReader(conn)

		for i := 1; i <= iterations; i++ {
			text := fmt.Sprintf("hello %d from worker %d\n", i, worker)
			_, err = conn.Write([]byte(text))
			s.Require().Nil(err)

			resTS, err := reader.ReadString('\n')
			s.Require().Nil(err)
			s.Assert().NotEmpty(resTS)
			ts, err := strconv.ParseInt(resTS, 10, 64)
			s.Assert().GreaterOrEqual(time.Now().Add(-time.Minute).UnixNano(), ts)
			s.Assert().GreaterOrEqual(time.Now().UnixNano(), ts)

			resText, err := reader.ReadString('\n')
			s.Require().Nil(err)
			s.Assert().Equal(text, resText)
		}

		s.Require().Nil(conn.Close())
		wg.Done()
	}
	for i := 1; i <= workers; i++ {
		go clientTest(i)
	}

	wg.Wait()
	s.Require().Nil(l.Close())
	<-th.Done()
}
