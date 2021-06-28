package main

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/3vilive/distlock"
	"github.com/go-redis/redis"
)

func main() {
	r := redis.NewClient(&redis.Options{
		Addr: "127.0.0.1:6379",
	})

	if err := r.Ping().Err(); err != nil {
		panic(err)
	}

	var wg sync.WaitGroup
	var count int
	var opCount int32

	for i := 0; i < 10; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()
			for {
				if count >= 10000 {
					break
				}
				count += 1
				atomic.AddInt32(&opCount, 1)
			}
		}()
	}

	wg.Wait()
	fmt.Printf("[without lock] count: %d op: %d\n", count, opCount)

	count = 0
	opCount = 0

	for i := 0; i < 10; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()
			for {
				lock, err := distlock.AcquireWithRedis("inc_count", r, distlock.WithTimeout(10*time.Second))
				if err != nil {
					panic(err)
				}

				if count >= 10000 {
					lock.Release()
					break
				}

				count += 1
				atomic.AddInt32(&opCount, 1)
				lock.Release()
			}
		}()
	}

	wg.Wait()
	fmt.Printf("[with lock] count: %d op: %d\n", count, opCount)
}
