package scheduler

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

type TaskContext struct {
	Context context.Context

	id string
}

type Task struct {
	sync.Mutex
	id          string
	TaskContext TaskContext
	Interval    time.Duration
	Execute     func() error
	ErrFunc     func(error)
	timer       *time.Timer
	ctx         context.Context
	cancel      context.CancelFunc
}

type Scheduler struct {
	sync.RWMutex

	tasks map[string]*Task
}

func New() *Scheduler {
	s := &Scheduler{
		tasks: make(map[string]*Task),
	}

	return s
}

func (s *Scheduler) AddTask(id string, t *Task) error {
	t.id = id

	if t.Execute == nil {
		return errors.New("task execute function cannot be nil")
	}

	if id == "" {
		return errors.New("task id cannot be empty")
	}

	if t.Interval <= time.Duration(0) {
		return errors.New("task interval must be defined")
	}

	if s.tasks[t.id] != nil {
		return errors.New("task already exists with id: " + t.id)
	}

	t.ctx, t.cancel = context.WithCancel(context.Background())

	s.Lock()
	defer s.Unlock()
	s.tasks[t.id] = t
	s.scheduleTask(t)

	return nil
}

func (s *Scheduler) scheduleTask(t *Task) {
	t.timer = time.AfterFunc(t.Interval, func() {
		fmt.Println("scheduling task:", t.id)

		if t.ctx.Err() != nil {
			fmt.Println("canceling task:", t.id)
			return
		}

		if err := t.Execute(); err != nil {
			t.ErrFunc(err)
			return
		}

		t.timer.Reset(t.Interval)
	})
}

func (s *Scheduler) ResetTaskSchedule(id string, interval time.Duration) {
	t := s.tasks[id]
	fmt.Println("resetting task schedule", id, "to", interval)
	t.timer.Reset(interval)
}

func (s *Scheduler) RemoveTask(id string) {
	fmt.Println("removing task", id)

	t := s.tasks[id]

	if t == nil {
		return
	}

	defer t.cancel()

	if t.timer != nil {
		defer t.timer.Stop()
	}

	s.Lock()
	defer s.Unlock()
	delete(s.tasks, id)
}
