package retry

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestNewPolicy(t *testing.T) {
	policy := NewPolicy()
	if policy == nil {
		t.Fatal("Expected retrial policy is not returned.")
	}
}

func TestPolicy_WithTrial(t *testing.T) {
	policy := (&Policy{}).WithTrial(10)

	if policy.Trial != 10 {
		t.Errorf("Unexpected trial count is set: %d", policy.Trial)
	}
}

func TestPolicy_WithInterval(t *testing.T) {
	policy := (&Policy{}).WithInterval(3 * time.Second)

	if policy.Interval != 3*time.Second {
		t.Errorf("Unexpected interval is set: %s", policy.Interval)
	}
}

func TestPolicy_WithRandFactor(t *testing.T) {
	policy := (&Policy{}).WithRandFactor(1.1)

	if policy.RandFactor != 1.1 {
		t.Errorf("Unexpected randomization factor is set: %f", policy.RandFactor)
	}
}

func TestWithPolicy(t *testing.T) {
	policy := &Policy{Trial: 1}
	called := false
	_ = WithPolicy(policy, func() error {
		called = true
		return nil
	})
	if !called {
		t.Error("Passed function is not called.")
	}
}

func TestRetry(t *testing.T) {
	trial := 3

	tests := []struct {
		failCnt int
	}{
		{
			// Succeed on the initial trial
			failCnt: 0,
		},
		{
			// Succeed on the last trial
			failCnt: trial - 1,
		},
		{
			// Keep failing
			failCnt: trial,
		},
	}

	for i, tt := range tests {
		t.Run(fmt.Sprintf("test #%d", i), func(t *testing.T) {
			successStr := "abc"
			cnt := 0
			str := ""
			err := Retry(trial, func() error {
				cnt++
				if cnt <= tt.failCnt {
					return fmt.Errorf("error on %d", cnt)
				}

				str = successStr
				return nil
			})

			// If the expected error count is equal to or greater than the retrial count, then error must be present.
			if tt.failCnt >= trial {
				e, ok := err.(*Errors)
				if !ok {
					t.Errorf("Returned error is not Errors: %+v", err)
				}
				if len(*e) != trial {
					t.Errorf("Something is wrong with retrial: %s", e.Error())
				}

				return
			}

			if err != nil {
				t.Errorf("Error is returned where it was not expected: %+v", err)
			}

			if str != successStr {
				t.Errorf("Expected string is not returned: %s", str)
			}
		})
	}
}

func TestErrors_Error(t *testing.T) {
	errs := &Errors{}
	firstErr := errors.New("1st error")
	errs.appendError(firstErr)
	secondErr := errors.New("2nd error")
	errs.appendError(secondErr)

	if errs.Error() != strings.Join([]string{firstErr.Error(), secondErr.Error()}, "\n") {
		t.Errorf("Unexpected error message is returned: %s.", errs.Error())
	}

	for i, err := range []error{firstErr, secondErr} {
		if (*errs)[i] != err {
			t.Errorf("Expected error is not accessible at %d: %+v", i, (*errs)[i])
		}
	}
}

func TestWithInterval(t *testing.T) {
	// Execute the function twice and see the actual interval
	i := 0
	interval := 100 * time.Millisecond
	var initialTrialAt time.Time
	var secondTrialAt time.Time
	_ = WithInterval(2, func() error {
		i++
		if i == 1 {
			initialTrialAt = time.Now()
		} else {
			secondTrialAt = time.Now()
		}

		return errors.New("error")
	}, interval)

	elapsed := secondTrialAt.Sub(initialTrialAt)
	if elapsed <= interval {
		t.Errorf("Expected retry interval is %d, but actual interval was %d.", interval.Nanoseconds(), elapsed.Nanoseconds())
	}
	// Give a bit buffer time for execution itself
	if elapsed >= interval+10*time.Millisecond {
		t.Errorf("Expected retry interval is %d, but actual interval was %d.", interval.Nanoseconds(), elapsed.Nanoseconds())
	}
}

func TestWithBackOff(t *testing.T) {
	// Execute the function twice and see if the actual interval is in-between minimum and maximum interval
	i := 0
	interval := 100 * time.Millisecond
	factor := 0.01
	var initialTrialAt time.Time
	var secondTrialAt time.Time
	_ = WithBackOff(2, func() error {
		i++
		if i == 1 {
			initialTrialAt = time.Now()
		} else {
			secondTrialAt = time.Now()
		}

		return errors.New("error")
	}, interval, factor)

	// Calculate the min/max interval
	delta := factor * float64(interval)
	min := float64(interval) - delta
	max := float64(interval) + delta

	elapsed := secondTrialAt.Sub(initialTrialAt)
	if float64(elapsed.Nanoseconds()) <= min {
		t.Errorf("Expected minimum retry interval is %f, but actual interval was %d.", min, elapsed.Nanoseconds())
	}
	// Give a bit buffer time for execution itself
	if float64(elapsed.Nanoseconds()) >= max+float64(10*time.Millisecond) {
		t.Errorf("Expected maximum retry interval is %f, but actual interval was %d.", max, elapsed.Nanoseconds())
	}
}

func Test_randInterval(t *testing.T) {
	t.Run("without random factor", func(t *testing.T) {
		interval := randInterval(5*time.Second, 0)
		if interval != 5*time.Second {
			t.Errorf("Returned interval differs from input while random factor is 0: %d", interval)
		}
	})

	t.Run("properly randomized", func(t *testing.T) {
		mean := 100 * time.Second
		for i := range make([]int, 100) {
			factor := float64(i) / 100
			delta := factor * float64(mean)
			min := float64(mean) - delta
			max := float64(mean) + delta
			given := randInterval(mean, factor)
			if !(min <= float64(given) && float64(given) <= max) {
				t.Errorf("Returned interval is not in the range of expectation. Mean: %g. Factor: %g. Given: %g.", mean.Seconds(), factor, given.Seconds())
			}
		}
	})

	t.Run("given factor is larger than the maximum value of 1.0", func(t *testing.T) {
		factor := float64(100000)
		mean := 100 * time.Second
		delta := 1.0 * float64(mean) // exceeded factor falls to default maximum value of 1.0
		min := float64(mean) - delta
		max := float64(mean) + delta
		given := randInterval(mean, factor)
		if !(min <= float64(given) && float64(given) <= max) {
			t.Errorf("Returned interval is not in the range of expectation. Mean: %g. Factor: %g. Given: %g.", mean.Seconds(), factor, given.Seconds())
		}
	})

	t.Run("given factor is smaller than the minimum value of 0.0", func(t *testing.T) {
		// factor less than 0 falls to default minimum value of 0.0
		factor := float64(-100000)
		mean := 100 * time.Second
		expected := float64(mean)
		given := randInterval(mean, factor)
		if float64(given) != expected {
			t.Errorf("expected interval is not returned: %d", given)
		}
	})
}
