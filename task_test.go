package task

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestWorker(t *testing.T) {
	ctx := context.Background()

	// #1 timeout worker
	w1 := NewTimeoutWorker(func(ctx context.Context) error {
		time.Sleep(time.Second)
		return nil
	}, time.Millisecond*500)
	err1 := w1.work(ctx)
	assert.Equal(t, ErrWorkerTimeout, err1)

	// #2 no timeout worker
	w2 := NewWorker(func(ctx context.Context) error {
		time.Sleep(time.Second)
		return nil
	})
	err2 := w2.work(ctx)
	assert.Equal(t, nil, err2)
}

func TestTimeout(t *testing.T) {
	ctx := context.Background()

	tt, cancel := WithTimeout(ctx, time.Millisecond*100)
	defer cancel(context.Canceled)

	w1 := NewWorker(func(ctx context.Context) error {
		time.Sleep(time.Second)
		fmt.Println("w1")
		return nil
	})

	for i := 0; i < 100; i++ {
		tt.Add(w1)
	}

	err := tt.Run()
	fmt.Println(err)
	assert.Equal(t, context.DeadlineExceeded, err)
}
