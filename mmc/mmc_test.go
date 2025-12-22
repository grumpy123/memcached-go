package mmc

import (
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

func (s *MmcSuite) TestMmc() {
	mmcAddr := s.StrEnv("MEMCACHED_ADDR", "")
	if mmcAddr == "" {
		s.T().Skip("MEMCACHED_ADDR not set, skipping test")
		return
	}

	_, err := gonet.NewClient(mmcAddr, 1, 1)
	s.Require().NoError(err)
}
