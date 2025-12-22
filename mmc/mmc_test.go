package mmc

import (
	"bufio"
	"context"
	"fmt"
	"memcached-go/gonet"
	"memcached-go/testutil"
	"testing"

	"github.com/stretchr/testify/suite"
)

type MmcSuite struct {
	testutil.BaseSuite
}

func TestMmcSuite(t *testing.T) {
	suite.Run(t, new(MmcSuite))
}

// Used to trigger error responses
type testMsg struct {
	req string
	err error
}

func (t *testMsg) WriteRequest(w *bufio.Writer) error {
	if _, err := w.WriteString(t.req); err != nil {
		return err
	}
	if err := w.Flush(); err != nil {
		return err
	}
	return nil
}

func (t *testMsg) ReadResponse(r *bufio.Reader) error {
	header, err := respHeader(r)
	if err != nil {
		return fmt.Errorf("read response header: %w", err)
	}
	t.err = maybeError(header)
	return nil
}

func unknownCmdErrorMsg() *testMsg {
	return &testMsg{req: "foo\r\n"}
}

func clientErrorMsg() *testMsg {
	return &testMsg{req: "mg foo XYZSJFTEEW\r\n"}
}

func (s *MmcSuite) TestMmc() {
	mmcAddr := s.StrEnv("MEMCACHED_ADDR", "")
	if mmcAddr == "" {
		s.T().Skip("MEMCACHED_ADDR not set, skipping test")
		return
	}

	cli, err := gonet.NewConnection(mmcAddr)
	s.Require().NoError(err)

	errMsg := unknownCmdErrorMsg()
	err = cli.Call(context.Background(), errMsg)
	s.Require().NoError(err)
	s.ErrorIs(errMsg.err, ErrGenError)

	errMsg = clientErrorMsg()
	err = cli.Call(context.Background(), errMsg)
	s.Require().NoError(err)
	s.ErrorIs(errMsg.err, ErrClientError)

	getMsg := NewGet("foo")
	err = cli.Call(context.Background(), getMsg)
	s.Require().NoError(err)
	s.ErrorIs(getMsg.Error, ErrMiss)

	setMsg := NewSet("bar", 5, []byte("baz"))
	err = cli.Call(context.Background(), setMsg)
	s.Require().NoError(err)
	s.NoError(setMsg.Error)

	getMsg = NewGet("bar")
	err = cli.Call(context.Background(), getMsg)
	s.Require().NoError(err)
	s.NoError(getMsg.Error)
	s.Equal("baz", string(getMsg.Value))
	s.Equal(5, int(getMsg.Flags))
}
