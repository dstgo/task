package task

import (
	"context"
	"errors"
	"fmt"
	errors2 "github.com/pkg/errors"
	"sync"
	"time"
)

type Task interface {
	Add(j ...Worker)
	Run() error
	OnPanic(onPanic func(err any))
	Ctx() context.Context
	Cancel(err error)
	Clear() error
}

const DefaultJobSize = 10

var (
	ErrorTaskStillRunning = errors.New("task still running")
	ErrWorkerPanic        = errors.New("worker panic")
	ErrNilWorker          = errors.New("nil worker is not allowed")
	ErrWorkerTimeout      = errors.New("worker timeout")
)

var t Task = &task{}

func New(ctx context.Context) (Task, context.CancelCauseFunc) {
	ctx, cancel := context.WithCancel(ctx)
	task := newTask(ctx, cancel)
	return task, task.Cancel
}

func WithTimeout(ctx context.Context, timeout time.Duration) (Task, context.CancelCauseFunc) {
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	task := newTask(timeoutCtx, cancel)
	return task, task.Cancel
}

func WithDeadLine(ctx context.Context, deadline time.Time) (Task, context.CancelCauseFunc) {
	withDeadline, cancelFunc := context.WithDeadline(ctx, deadline)
	task := newTask(withDeadline, cancelFunc)
	return task, task.Cancel
}

func newTask(ctx context.Context, cancelFunc context.CancelFunc) *task {
	return &task{
		ctx:     ctx,
		cancel:  cancelFunc,
		workers: make([]Worker, 0, DefaultJobSize),
		wait:    sync.WaitGroup{},
		mutex:   sync.Mutex{},
		running: false,
	}
}

type WorkerFn func(ctx context.Context) error

func NewWorker(fn WorkerFn) Worker {
	return Worker{worker: fn}
}

func NewTimeoutWorker(fn WorkerFn, timeout time.Duration) Worker {
	return Worker{worker: fn, timeout: timeout}
}

type Worker struct {
	worker  func(ctx context.Context) error
	timeout time.Duration
}

func (w Worker) work(ctx context.Context) error {
	if w.timeout != 0 {
		return w.workTimeout(ctx)
	}
	return w.worker(ctx)
}

func (w Worker) workTimeout(ctx context.Context) error {
	done := make(chan struct{})
	var err error
	go func() {
		err = w.worker(ctx)
		done <- struct{}{}
		close(done)
	}()
	select {
	case <-time.After(w.timeout):
		return ErrWorkerTimeout
	case <-done:
		return err
	}
}

type task struct {
	mutex sync.Mutex
	wait  sync.WaitGroup
	once  sync.Once

	ctx    context.Context
	cancel context.CancelFunc

	workers []Worker
	waiter  []Worker
	onPanic func(err any)

	err     error
	running bool
}

func (t *task) Ctx() context.Context {
	return t.ctx
}

func (t *task) Cancel(err error) {
	t.once.Do(func() {
		t.err = err
		if t.cancel != nil {
			t.cancel()
		}
	})
}

func (t *task) setRunning(running bool) {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	t.running = running
}

func (t *task) Clear() error {
	if t.running {
		return ErrorTaskStillRunning
	}

	t.wait.Wait()
	t.workers = append(t.waiter)
	t.waiter = t.waiter[:0]

	return nil
}

func (t *task) OnPanic(onPanic func(err any)) {
	t.onPanic = onPanic
}

func (t *task) Add(j ...Worker) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	if !t.running {
		t.workers = append(t.workers, j...)
	} else {
		t.waiter = append(t.waiter, j...)
	}
}

// Run workers until all of them finished
// if finished all workers successfully, it will return nil
// else if ctx was cancelled, it will return context.Canceled
// else it will return the error that worker returned
func (t *task) Run() error {
	if !t.running {

		// iterate over a snapshot of a certain moment
		for _, job := range t.workers {
			if job.worker == nil {
				return ErrNilWorker
			}
		}

		t.setRunning(true)
		t.wait.Add(len(t.workers))

		for _, job := range t.workers {
			go func(w Worker) {
				// wait count--
				defer t.wait.Done()

				taskCh := make(chan struct{})

				// worker goroutine
				go func() {
					defer close(taskCh)
					// panic recovery
					defer func() {
						if err := recover(); err != nil {
							if t.onPanic != nil {
								t.onPanic(err)
							}
							t.Cancel(errors2.Wrap(ErrWorkerPanic, fmt.Sprintf("%+v", err)))
						}
					}()

					// call func
					if err := w.work(t.ctx); err != nil {
						t.Cancel(err)
					}

					taskCh <- struct{}{}
				}()

				select {
				case <-t.ctx.Done():
					return
				case <-taskCh:
					return
				}
			}(job)
		}

		t.wait.Wait()
		t.setRunning(false)
		t.Clear()
	}

	// if all workers finished normally, t.err should be nil
	if t.err == nil {
		// may be ctx was canceled
		t.err = t.ctx.Err()
	}

	return t.err
}
