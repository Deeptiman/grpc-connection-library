package main

import (
	"math"
	"math/rand"
	"time"
)

type BackoffStrategy int

var (
	DefaultBackoff   = 2
	LinearBackoff    = DefaultBackoff
	ExponetialFactor = 1.5
)

const (
	Default BackoffStrategy = iota
	Jitter
	Linear
	Exponetial
)

type Backoff struct {
	strategy BackoffStrategy
}

func getExponentialBackoff(retry int) float64 {
	return math.Pow(2, float64(retry)) * ExponetialFactor
}

func (b *Backoff) ApplyBackoffDuration(retry int) {

	var backoffDuration int64

	switch b.strategy {
	case Jitter:
		v := getExponentialBackoff(retry)
		backoffDuration = int64(v/2 + float64(rand.Intn(int(v/2))))
	case Linear:
		LinearBackoff++
		backoffDuration = int64(LinearBackoff)
	case Exponetial:
		backoffDuration = int64(getExponentialBackoff(retry))
	default:
		backoffDuration = int64(DefaultBackoff)
	}

	time.Sleep(time.Duration(backoffDuration) * time.Second)
}
