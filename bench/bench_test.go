package bench

import (
	"context"
	"fmt"
	memcached_go "memcached-go"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func BenchmarkClient(b *testing.B) {
	// todo: make proper benchmark, integrate iterations with framework, etc.
	workers := 10
	iterations := 1000
	conns := 5

	// Require memcached container
	mmcAddr := "localhost:11211"
	cli, err := memcached_go.NewClient(mmcAddr, conns, conns)
	require.NoError(b, err)
	defer cli.Close()

	wg := &sync.WaitGroup{}
	wg.Add(workers)

	for i := 1; i <= iterations; i++ {
		key := fmt.Sprintf("bench-%d", i)
		val := []byte(fmt.Sprintf("value-%d-blahblahblah", i))
		err = cli.Set(context.Background(), key, 0, val, time.Hour)
		require.NoError(b, err)
	}

	clientTest := func(worker int) {
		defer wg.Done()
		for i := 1; i <= iterations; i++ {
			key := fmt.Sprintf("bench-%d", i)
			_, _, err = cli.Get(context.Background(), key)
		}
	}

	for i := 1; i <= workers; i++ {
		go clientTest(i)
	}
	wg.Wait()
}
