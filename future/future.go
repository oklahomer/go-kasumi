// Package future provides a mechanism to run a job in an asynchronous manner and subscribe to the result.
package future

import (
	"context"
	"errors"
	"fmt"
)

// ErrTimeout indicates that a call to Future.Wait() timed out.
var ErrTimeout = errors.New("task execution timed out")

// Future provides access to an ongoing event that is not finished, yet.
type Future struct {
	result chan *Result
}

// New creates a Future instance with the given executable function.
// Subscribing to this instance gives asynchronous access to the result.
func New(fn func() (interface{}, error)) *Future {
	f := newFuture()

	go func() {
		f.result <- execute(fn)
	}()

	return f
}

// NewError creates a Future instance that returns the given error.
// Since this does not dispatch a task to a new goroutine, this is more efficient than calling New() with a function that returns an error.
func NewError(err error) *Future {
	f := newFuture()
	f.result <- &Result{Error: err}

	return f
}

func newFuture() *Future {
	return &Future{
		result: make(chan *Result, 1),
	}
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
	return <-f.result
}

// Wait waits until its execution finishes or the given ctx is canceled, and then returns the Result.
// A typical usage is somewhat like below:
//
//	// Initiate a time-consuming task
//	f := future.New(func() (interface{}, error)) {
// 		// Do some time-consuming task
//		return struct{}{}, nil
// 	}
//
//	// Prepare a context.Context that is canceled when the parentCtx is canceled or the timeout interval passes.
//	ctx, _ := context.WithTimeout(parentCtx, 3*time.Second)
//
//	// Wait til the task execution ends or the ctx is canceled.
//	result := f.Wait(ctx)
//
// When the given ctx is canceled before the execution completion, the returning Result always contains an error value of ErrTimeout.
func (f *Future) Wait(ctx context.Context) *Result {
	select {
	case result := <-f.result:
		return result

	case <-ctx.Done():
		return &Result{
			Error: ErrTimeout,
		}
	}
}

// C returns a channel that passes the Result of execution.
// Reading this channel without cancellation mechanism is equivalent to the call to Block;
// The call to this with a select statement with one alternative case of context.Done or timer.C is equivalent to Wait.
// This method is beneficial when a more flexible select statement with multiple alternative cases is in use.
//
//	for {
//		select {
//		case result := <- future1.C():
//			return result, nil
//
//		case result := <- future2.C():
//			return result, nil
//
//		case <- ticker.C:
//			fmt.Println(`waiting...`)
//
//		case <- ctx.Done():
//			return nil, errors.New(`timed out`)
//		}
//	}
func (f *Future) C() <-chan *Result {
	return f.result
}

// Then lets a given function to be executed after the preceding task.
// When the preceding task finishes successfully, then the result value which is equivalent to Result.Value is passed to fn;
// When the preceding task finishes with an error, then this immediately returns a Future with the preceding error.
func (f *Future) Then(fn func(value interface{}) (interface{}, error)) *Future {
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
