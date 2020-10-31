package future

import (
	"errors"
	"fmt"
	"time"
)

// ErrTimeout indicates that a call to Future.Wait() timed out.
var ErrTimeout = errors.New("task execution timed out")

// Future provides access to ongoing events that is not finished, yet.
type Future struct {
	finished chan struct{}
	result   *Result
}

// New creates a Future instance with the given executable function.
// Subscribing to this instance gives asynchronous access to the result.
func New(fn func() (interface{}, error)) *Future {
	f := &Future{
		finished: make(chan struct{}),
	}

	go func() {
		f.result = execute(fn)
		close(f.finished)
	}()

	return f
}

func execute(fn func() (interface{}, error)) (result *Result) {
	defer func() {
		r := recover()
		if r != nil {
			switch typed := r.(type) {
			case error:
				err := fmt.Errorf("panic on Future: %w", typed)
				result = &Result{
					Error: err,
				}

			default:
				result = &Result{
					Error: fmt.Errorf("panic on Future: %+v", typed),
				}
			}
		}
	}()

	r, err := fn()
	return &Result{
		Value: r,
		Error: err,
	}
}

// Block waits until the given function finishes and returns the Result.
func (f *Future) Block() *Result {
	<-f.finished
	return f.result
}

// Wait waits until the given duration passes and returns the Result.
// If the task is not finished within the given interval, the returning Result contains an error of ErrTimeout.
func (f *Future) Wait(d time.Duration) *Result {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-f.finished:
		return f.result

	case <-timer.C:
		return &Result{
			Error: ErrTimeout,
		}
	}
}

// Then lets a given function to be executed after the preceding task.
// When the preceding task finishes successfully, then the result is passed to fn;
// When the preceding task finishes with an error, then this immediately returns a Future with the preceding error.
func (f *Future) Then(fn func(data interface{}) (interface{}, error)) *Future {
	return New(func() (interface{}, error) {
		// Wait till the initial task ends
		result := f.Block()

		// If the initial task returns an error, then return such a state without executing the next task
		if result.Error != nil {
			return nil, result.Error
		}

		// Pass the initial task's result value to the next task
		return fn(result.Value)
	})
}

// Result represents the result of the given task.
// Always check its Error first before referring to Value.
type Result struct {
	// Value represents the result of finished task.
	Value interface{}

	// Error provides access to the error returned by finished/canceled task.
	Error error
}
