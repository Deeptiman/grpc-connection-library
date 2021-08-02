package retry

import (
	"math"
	"math/rand"
	"time"
)

type BackoffStrategy int

var (
	DefaultBackoff   = 2   // DefaultBackoff duration 2 secs
	LinearBackoff    = 1   // LinearBackoff duration starts from 1 second
	ExponetialFactor = 1.5 // ExponetialFactor 1.5 per retry attempts
)

const (
	Default BackoffStrategy = iota // BackoffStrategy constants
	Jitter
	Linear
	Exponetial
)

// Backoff struct defines the backoff strategy type
type Backoff struct {
	Strategy BackoffStrategy
}

// getExponentialBackoff will compute the exponential backoff time calculation for a retry attempt.
func getExponentialBackoff(attempt int) float64 {
	return math.Pow(2, float64(attempt)) * ExponetialFactor
}

// ApplyBackoffDuration will decide a backoff strategy for a retry attempt.
func (b *Backoff) ApplyBackoffDuration(attempt int) {

	var backoffDuration int64

	switch b.Strategy {
	case Jitter:
		v := getExponentialBackoff(attempt)
		backoffDuration = int64(v/2 + float64(rand.Intn(int(v/2))))
	case Linear:
		LinearBackoff = attempt + 1
		backoffDuration = int64(LinearBackoff)
	case Exponetial:
		backoffDuration = int64(getExponentialBackoff(attempt))
	default:
		backoffDuration = int64(DefaultBackoff)
	}
	log.Warningln("grpc:retry:attempt:backoff:", backoffDuration, " sec")
	time.Sleep(time.Duration(backoffDuration) * time.Second)
}
