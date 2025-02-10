package gonet

import (
	"bufio"
	"context"
	"fmt"
	"github.com/stretchr/testify/suite"
	"net"
	"testing"
	"time"
)

type ServerSuite struct {
	suite.Suite
}

func TestServerSuite(t *testing.T) {
	suite.Run(t, new(ServerSuite))
}

func (s *ServerSuite) setupListener(hf ConnectionHandler) *Listener {
	l := NewListener(0, hf)
	err := l.Start(context.Background())
	s.Require().Nil(err)

	return l
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

	resText, err := reader.ReadString('\n')
	s.Require().Nil(err)
	s.Assert().Equal("hello\n", resText)

	s.Require().Nil(l.Close())
	<-th.Done()
}
