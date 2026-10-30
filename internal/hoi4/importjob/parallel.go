package importjob

import (
	"context"
	"runtime"
	"sync"
)

type fileRows[R any] struct {
	rel     string
	rows    []R
	invalid error
}

type parsedFile[T any] struct {
	value T
	err   error
}

func inFileOrder[T any](ctx context.Context, n int, parse func(i int) (T, error), use func(i int, value T) error) error {
	workers := min(runtime.GOMAXPROCS(0), n)
	if workers <= 1 {
		for i := 0; i < n; i++ {
			if err := ctx.Err(); err != nil {
				return err
			}
			value, err := parse(i)
			if err != nil {
				return err
			}
			if err := use(i, value); err != nil {
				return err
			}
		}
		return nil
	}
	stop, cancel := context.WithCancel(ctx)
	var wg sync.WaitGroup
	defer func() {
		cancel()
		wg.Wait()
	}()
	results := make([]chan parsedFile[T], n)
	for i := range results {
		results[i] = make(chan parsedFile[T], 1)
	}

	ahead := make(chan struct{}, 4*workers)
	next := make(chan int)
	wg.Add(1 + workers)
	go func() {
		defer wg.Done()
		defer close(next)
		for i := 0; i < n; i++ {
			select {
			case ahead <- struct{}{}:
			case <-stop.Done():
				return
			}
			select {
			case next <- i:
			case <-stop.Done():
				return
			}
		}
	}()
	for range workers {
		go func() {
			defer wg.Done()
			for i := range next {
				value, err := parse(i)
				results[i] <- parsedFile[T]{value, err}
			}
		}()
	}
	for i := 0; i < n; i++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		var file parsedFile[T]
		select {
		case file = <-results[i]:
		case <-ctx.Done():
			return ctx.Err()
		}
		<-ahead
		if file.err != nil {
			return file.err
		}
		if err := use(i, file.value); err != nil {
			return err
		}
	}
	return nil
}
