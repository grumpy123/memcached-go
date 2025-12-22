package gonet

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

type ServerSuite struct {
	BaseSuite
}

func TestServerSuite(t *testing.T) {
	suite.Run(t, new(ServerSuite))
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
	if strings.HasPrefix(input, "err:") {
		return nil, errors.New(input)
	}
	return &TestRequest{input: input}, nil
}

func (t *TestRequest) Handle() {
	t.ts = time.Now()
	// todo: inject things here
}

func (t *TestRequest) WriteResponse(writer *bufio.Writer) error {
	_, err := writer.WriteString(fmt.Sprintf("%d\n%s", t.ts.UnixMicro(), t.input))
	if err != nil {
		return err
	}
	return nil
}

func (s *ServerSuite) TestServer() {
	h := &TestRequestHandler{}
	th := WithTracking(NewServerFactory(h))

	l := s.setupListener(th)

	conn, err := net.Dial("tcp", l.Address().String())
	s.Require().NoError(err)

	reader := bufio.NewReader(conn)

	s.testMessage(conn, reader, "hello\n")

	s.Require().NoError(l.Close())
	s.Require().NoError(conn.Close())
	<-th.Done()
}

func (s *ServerSuite) TestServerWithErrors() {
	h := &TestRequestHandler{}
	th := WithTracking(NewServerFactory(h))

	l := s.setupListener(th)

	conn, err := net.Dial("tcp", l.Address().String())
	s.Require().NoError(err)

	reader := bufio.NewReader(conn)

	s.testMessage(conn, reader, "hello\n")

	_, err = conn.Write([]byte("err:triggering error\n"))
	s.Require().NoError(err)

	_, err = reader.ReadString('\n')
	s.Require().Error(err)

	// try again, with a new connection, it should succeed

	conn, err = net.Dial("tcp", l.Address().String())
	s.Require().NoError(err)

	reader = bufio.NewReader(conn)

	s.testMessage(conn, reader, "hello\n")

	s.Require().NoError(l.Close())
	s.Require().NoError(conn.Close())
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
		s.Require().NoError(err)
		reader := bufio.NewReader(conn)

		for i := 1; i <= iterations; i++ {
			text := fmt.Sprintf("hello %d from worker %d\n", i, worker)
			s.testMessage(conn, reader, text)
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

func (s *ServerSuite) testMessage(conn net.Conn, reader *bufio.Reader, text string) {
	sendTime := s.nowUnixMicro()
	_, err := conn.Write([]byte(text))
	s.Require().NoError(err)

	resTS, err := reader.ReadString('\n')
	s.Require().NoError(err)
	s.NotEmpty(resTS)
	ts, err := strconv.ParseInt(strings.TrimSpace(resTS), 10, 64)
	s.Require().NoError(err)
	s.LessOrEqual(sendTime.UnixMicro(), ts)
	s.GreaterOrEqual(time.Now().UnixNano(), ts)

	resText, err := reader.ReadString('\n')
	s.Require().NoError(err)
	s.Equal(text, resText)
}
