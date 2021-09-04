package future

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	future := New(func() (interface{}, error) {
		return nil, nil
	})

	if future == nil {
		t.Error("Future instance is not returned.")
	}
}

func TestNewError(t *testing.T) {
	t.Run("a single error", func(t *testing.T) {
		err := errors.New("an error")
		future := NewError(err)

		if future == nil {
			t.Error("Future instance is not returned.")
		}

		result := future.Block()
		if result.Error != err {
			t.Errorf("The given error is not returned: %+v", result.Error)
		}
	})

	t.Run("chained methods", func(t *testing.T) {
		err := errors.New("an error")
		future := NewError(err).
			Then(func(_ interface{}) (interface{}, error) {
				return nil, nil
			})

		if future == nil {
			t.Error("Future instance is not returned.")
		}

		result := future.Block()
		if result.Error != err {
			t.Errorf("The given error is not returned: %+v", result.Error)
		}
	})
}

func TestFuture_Block(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		result := New(func() (interface{}, error) {
			time.Sleep(1 * time.Second)
			return true, nil
		}).Block()

		if result.Error != nil {
			t.Errorf("Unexpected error is returned: %+v", result.Error)
		}

		val, ok := result.Value.(bool)
		if !ok {
			t.Fatalf("Unexpected return type: %T", result.Value)
		}

		if !val {
			t.Errorf("Unexpected return value: %t", val)
		}
	})

	t.Run("panic recovery", func(t *testing.T) {
		result := New(func() (interface{}, error) {
			panic("PANIC!!!")
		}).Block()

		if result.Error == nil {
			t.Fatalf("Expected error is not returned.")
		}

		if !strings.Contains(result.Error.Error(), "PANIC!!!") {
			t.Errorf("Expected error string is not contained: %s", result.Error.Error())
		}
	})

	t.Run("panic recovery with wrapped error", func(t *testing.T) {
		err := errors.New("DUMMY")
		result := New(func() (interface{}, error) {
			panic(err)
		}).Block()

		if result.Error == nil {
			t.Fatalf("Expected error is not returned.")
		}

		if !errors.Is(result.Error, err) {
			t.Error("Expected error is not wrapped")
		}
	})
}

func TestFuture_Wait(t *testing.T) {
	t.Run("timeout", func(t *testing.T) {
		result := New(func() (interface{}, error) {
			time.Sleep(1 * time.Second)
			return true, nil
		}).Wait(100 * time.Millisecond)

		err := result.Error
		if err == nil {
			t.Fatal("Expected error is not returned.")
		}

		if err != ErrTimeout {
			t.Errorf("Returned error type differs from expectation: %+v", err)
		}

		if result.Value != nil {
			t.Errorf("Unexpected value is returned: %+v", result.Value)
		}
	})

	t.Run("success", func(t *testing.T) {
		startAt := time.Now()
		result := New(func() (interface{}, error) {
			time.Sleep(100 * time.Millisecond)
			return true, nil
		}).Wait(1 * time.Second)
		elapsed := time.Now().Sub(startAt)

		// Give a buffer of give or take 10 milliseconds
		if elapsed >= 110*time.Millisecond {
			t.Errorf("Task waited too long: %d millisecond.", elapsed/time.Millisecond)
		}

		if result.Error != nil {
			t.Errorf("Unexpected error is returned: %+v", result.Error)
		}

		val, ok := result.Value.(bool)
		if !ok {
			t.Fatalf("Unexpected return type: %T", result.Value)
		}

		if !val {
			t.Errorf("Unexpected return value: %t", val)
		}
	})
}

func TestFuture_Then(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		result := New(func() (interface{}, error) {
			return 1, nil
		}).Then(func(data interface{}) (interface{}, error) {
			i := data.(int)
			return i + 1, nil
		}).Block()

		if result.Error != nil {
			t.Errorf("Unexpected error is returned: %+v", result.Error)
		}

		val, ok := result.Value.(int)
		if !ok {
			t.Fatalf("Unexpected return type: %T", val)
		}

		if val != 2 {
			t.Errorf("Unexpeccted value is returned: %d", val)
		}
	})

	t.Run("first task fails", func(t *testing.T) {
		called := make(chan struct{}, 1)
		err := errors.New("DUMMY")

		result := New(func() (interface{}, error) {
			return nil, err
		}).Then(func(data interface{}) (interface{}, error) {
			called <- struct{}{}
			return struct{}{}, nil
		}).Block()

		if result.Error == nil {
			t.Fatal("Expected error is not returned.")
		}
		if result.Error != err {
			t.Errorf("Unexpected error is returned: %+v", result.Error)
		}

		select {
		case <-called:
			t.Error("Second task is unexpectedly called.")

		default:
			// O.K.
		}
	})
}
