package gonet

import (
	"context"
	"memcached-go/testutil"
)

func must(f func() error) {
	if err := f(); err != nil {
		panic(err)
	}
}

type BaseSuite struct {
	testutil.BaseSuite
}

func (s *BaseSuite) SetupListener(h ConnectionHandler) *Listener {
	l := NewListener(0, h)
	err := l.Start(context.Background())
	s.Require().NoError(err)

	return l
}
