package ratelimiter

import (
	"context"
	"sort"
	"time"

	"golang.org/x/time/rate"
)

type RateLimiter interface {
	Wait(context.Context) error
	Limit() rate.Limit
}

func NewMultiLimiter(limits ...RateLimiter) *multiLimiter {
	byLimit := func(i, j int) bool {
		return limits[i].Limit() < limits[j].Limit()
	}
	sort.Slice(limits, byLimit)

	return &multiLimiter{limiters: limits}
}

type multiLimiter struct {
	limiters []RateLimiter
}

func (l *multiLimiter) Wait(ctx context.Context) error {
	for _, l := range l.limiters {
		if err := l.Wait(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (l *multiLimiter) Limit() rate.Limit {
	return l.limiters[0].Limit()
}

func Open() *APIConnection {

	return &APIConnection{
		networkLimit: NewMultiLimiter(
			rate.NewLimiter(Per(3, time.Second), 3),
		),
		diskLimit: NewMultiLimiter(
			rate.NewLimiter(rate.Limit(1), 1),
		),
		apiLimit: NewMultiLimiter(
			rate.NewLimiter(Per(2, time.Second), 1),
			rate.NewLimiter(Per(10, time.Minute), 10),
		),
	}
}

type APIConnection struct {
	networkLimit,
	diskLimit,
	apiLimit RateLimiter
}

func (a *APIConnection) Readfile(ctx context.Context) error {
	if err := NewMultiLimiter(a.apiLimit, a.networkLimit).Wait(ctx); err != nil {
		return err
	}
	return nil
}

func (a *APIConnection) ResolveAddress(ctx context.Context) error {
	if err := NewMultiLimiter(a.apiLimit, a.networkLimit).Wait(ctx); err != nil {
		return err
	}

	return nil
}

func Per(eventCount int, duration time.Duration) rate.Limit {
	return rate.Every(duration / time.Duration(eventCount))
}
