package testutil

import (
	"os"
	"strconv"
	"time"

	"github.com/stretchr/testify/suite"
)

type BaseSuite struct {
	suite.Suite
}

func (s *BaseSuite) StrEnv(env string, defaultValue string) string {
	strValue := os.Getenv(env)
	if strValue == "" {
		return defaultValue
	}

	return strValue
}

func (s *BaseSuite) IntEnv(env string, defaultValue int) int {
	strValue := os.Getenv(env)
	if strValue == "" {
		return defaultValue
	}

	i, err := strconv.Atoi(strValue)
	s.Require().NoError(err)
	return i
}

func (s *BaseSuite) NowUnixMicro() time.Time {
	return time.Now().Truncate(time.Microsecond)
}
