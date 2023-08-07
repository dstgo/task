package task

import (
	"sync"
)

func NewTask(onPanic func(err any)) *Task {
	return &Task{
		jobs:    make([]Job, 0, 10),
		onPanic: onPanic,
		wait:    sync.WaitGroup{},
		mutex:   sync.Mutex{},
		running: false,
	}
}

type Job func()

type Task struct {
	jobs    []Job
	onPanic func(err any)
	wait    sync.WaitGroup
	mutex   sync.Mutex
	running bool
}

func (t *Task) setRunning(running bool) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	t.running = running
}

func (t *Task) ClearJobs() {
	t.mutex.Lock()
	t.jobs = t.jobs[:0]
	t.mutex.Unlock()
}

func (t *Task) OnPanic(onPanic func(err any)) {
	t.onPanic = onPanic
}

func (t *Task) AddJobs(j ...Job) {
	if !t.running {
		t.jobs = append(t.jobs, j...)
	}
}

func (t *Task) Run() {
	if !t.running {

		t.setRunning(true)
		t.wait.Add(len(t.jobs))

		for _, job := range t.jobs {
			go func(j Job) {
				defer t.wait.Done()
				defer func() {
					if err := recover(); err != nil {
						if t.onPanic != nil {
							t.onPanic(err)
						}
					}
				}()
				j()
			}(job)

		}
		t.wait.Wait()

		t.setRunning(false)
	}
}
