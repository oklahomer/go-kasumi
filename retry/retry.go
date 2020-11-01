// Package retry provides general retry logic and designated error structure that contains each retrial's error.
package retry

import (
	"math/rand"
	"strings"
	"time"
)

// Errors is an alias of a slice of errors that contains ordered errors that occurred during retrials.
// This satisfies error interface, and a call to Error() returns concatenated message of all belonging errors.
//
// Since this is an alias of []error, each belonging error is accessible in a way such as:
//
//  for i, err := range *errs { ... }
type Errors []error

// Error returns the concatenated message of all belonging errors.
// All err.Err() strings are joined with "\n".
func (e *Errors) Error() string {
	var errs []string
	for _, err := range *e {
		errs = append(errs, err.Error())
	}
	return strings.Join(errs, "\n")
}

func (e *Errors) appendError(err error) {
	*e = append(*e, err)
}

// NewPolicy creates and returns a new retrial policy.
// Rather than selecting specific retrial functions -- Retry(), WithInterval() and WithBackOff() -- depending on usages,
// developers are free to pass a configurable retrial policy to WithPolicy().
//
// With the given policy, WithPolicy() will decide how retrials are executed so developers may modify the retrial behavior
// by changing the policy.
// This is especially effective when the Policy is mapped from JSON/YAML file in a way such as below:
//
//   policy := NewPolicy()
//   configBytes, _ := ioutil.ReadFile(filename)
//   json.Unmarshal(configBytes, policy)
//
// To manually set each configurable field, Call WithXxx methods of returning Policy.
//
//   policy := NewPolicy().
//       WithTrial(3).
//       WithInterval(3*time.Second)
//
func NewPolicy() *Policy {
	return &Policy{
		Trial:      1,
		Interval:   0,
		RandFactor: 0,
	}
}

// Policy represents a configuration value for retrial logic.
type Policy struct {
	Trial      int           `json:"trial" yaml:"trial"`
	Interval   time.Duration `json:"interval" yaml:"interval"`
	RandFactor float64       `json:"random_factor" yaml:"random_factor"`
}

// WithTrial sets the maximum trial count.
func (p *Policy) WithTrial(cnt int) *Policy {
	p.Trial = cnt
	return p
}

// WithInterval sets interval for each retrial.
// When RandFactor is set, this interval is used as the base interval.
func (p *Policy) WithInterval(d time.Duration) *Policy {
	p.Interval = d
	return p
}

// WithRandFactor sets randomization factor.
// Make sure to set Interval with WithInterval() or JSON/YAML deserialization.
func (p *Policy) WithRandFactor(factor float64) *Policy {
	p.RandFactor = factor
	return p
}

// WithPolicy receives retrial policy and an executable function.
// The passed function is repeatedly executed until no error is returned or the retrial count exceeds the given configuration value.
// Unlike other retrial functions, this function is among the most flexible since a developer has maximum freedom on the configuration.
func WithPolicy(policy *Policy, function func() error) error {
	return WithBackOff(policy.Trial, function, policy.Interval, policy.RandFactor)
}

// Retry tries to execute the given function as many times as the maximum trial count.
// It quits retrying when the function returns no error, which is nil.
// When all trials fail, Errors is returned to notify such error occurrences.
func Retry(trial int, function func() error) error {
	return WithInterval(trial, function, 0*time.Second)
}

// WithInterval retries the given function at interval.
func WithInterval(trial int, function func() error, interval time.Duration) error {
	return WithBackOff(trial, function, interval, 0)
}

// WithBackOff retries given function at interval, but the interval differs every time.
// The base interval and randomization factor are specified by 3rd and 4th arguments.
func WithBackOff(trial int, function func() error, meanInterval time.Duration, randFactor float64) error {
	errs := &Errors{}
	for trial > 0 {
		trial--
		err := function()
		if err == nil {
			return nil
		}
		errs.appendError(err)

		if trial <= 0 {
			// All trials failed
			break
		}

		if randFactor <= 0 || meanInterval <= 0 {
			time.Sleep(meanInterval)
		} else {
			interval := randInterval(meanInterval, randFactor)
			time.Sleep(interval)
		}
	}

	return errs
}

func randInterval(intervalDuration time.Duration, randFactor float64) time.Duration {
	if randFactor < 0 {
		randFactor = 0
	} else if randFactor > 1 {
		randFactor = 1
	}

	interval := float64(intervalDuration)
	delta := randFactor * interval
	minInterval := interval - delta
	maxInterval := interval + delta

	return time.Duration(minInterval + (rand.Float64() * (maxInterval - minInterval + 1)))
}
