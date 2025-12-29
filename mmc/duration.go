package mmc

import "time"

// Provides converters for duration/TTL, as memcached protocol makes this area more complex than expected.
// Below:
// * ttl: time.Duration expiration time
// * exptime: memcached protocol expiration time (input)

const (
	Time30days = 30 * 24 * time.Hour
)

func ttlToExptime(ttl time.Duration) int32 {
	if ttl >= Time30days {
		return int32(time.Now().Add(ttl).Unix())
	}
	return int32(ttl.Truncate(time.Second).Seconds())
}
