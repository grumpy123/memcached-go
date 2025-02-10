package gonet

import (
	"context"
	"github.com/stretchr/testify/suite"
	"os"
	"strconv"
)

func must(f func() error) {
	if err := f(); err != nil {
		panic(err)
	}
}

type BaseSuite struct {
	suite.Suite
}

func (s *BaseSuite) setupListener(h ConnectionHandler) *Listener {
	l := NewListener(0, h)
	err := l.Start(context.Background())
	s.Require().Nil(err)

	return l
}

func (s *BaseSuite) intEnv(env string, defaultValue int) int {
	strValue := os.Getenv(env)
	if strValue == "" {
		return defaultValue
	}

	i, err := strconv.Atoi(strValue)
	s.Require().Nil(err)
	return i
}
