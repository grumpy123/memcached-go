package gonet

import (
	"bufio"
	"context"
	"github.com/stretchr/testify/suite"
	"strconv"
	"strings"
	"testing"
	"time"
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

	msg := &TestMessage{inText: "hello world"}
	err = conn.Call(context.Background(), msg)
	s.Require().Nil(err)

	s.Assert().LessOrEqual(time.Now().Add(-time.Minute), msg.outTS)
	s.Assert().GreaterOrEqual(time.Now(), msg.outTS)
	s.Assert().Equal(msg.outText, msg.inText)

	s.Require().Nil(l.Close())
	<-th.Done()
}
