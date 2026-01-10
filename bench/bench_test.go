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
	// Require memcached container
	mmcAddr := "localhost:11211"
	keys := 1000

	prepData(b, mmcAddr, keys)

	for _, workers := range []int{1, 2} {
		b.Run(fmt.Sprintf("%d connections", workers), func(b *testing.B) {
			runBench(b, mmcAddr, keys, workers)
		})
	}
}

func prepData(b *testing.B, mmcAddr string, keys int) {
	cli, err := memcached_go.NewClient(mmcAddr, 1, 1)
	require.NoError(b, err)
	defer cli.Close()

	for i := 0; i < keys; i++ {
		key := fmt.Sprintf("bench-%d", i)
		val := []byte(fmt.Sprintf("value-%d-blahblahblah", i))
		err = cli.Set(context.Background(), key, 0, val, time.Hour)
		require.NoError(b, err)
	}
}

func runBench(b *testing.B, mmcAddr string, keys, conns int) {
	workers := conns * 100
	iterations := b.N / workers
	if iterations == 0 {
		iterations = 1
	}

	cli, err := memcached_go.NewClient(mmcAddr, conns, conns)
	require.NoError(b, err)
	defer cli.Close()

	wg := &sync.WaitGroup{}
	wg.Add(workers)

	b.ResetTimer()
	clientTest := func(worker int) {
		defer wg.Done()
		for i := 1; i <= iterations; i++ {
			key := fmt.Sprintf("bench-%d", i%keys)
			_, _ = cli.GetV(context.Background(), key)
		}
	}

	for i := 1; i <= workers; i++ {
		go clientTest(i)
	}
	wg.Wait()
}
