package usecase

import "time"

type Clock interface {
	Now() time.Time
}

type RNG interface {
	Int63() int64
}

type realClock struct{}

func (realClock) Now() time.Time { return time.Now() }
