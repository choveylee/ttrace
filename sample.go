/**
 * @Author: lidonglin
 * @Description:
 * @File:  sample.go
 * @Version: 1.0.0
 * @Date: 2023/02/27 12:38
 */

package ttrace

import (
	"fmt"
	"math"
	"sync"
	"time"

	"go.opentelemetry.io/otel/sdk/trace"
)

// RateLimiter decides whether a request costing itemCost credits is within limits.
//
// Deprecated: use [ReconfigurableRateLimiter].
type RateLimiter interface {
	// CheckCredit reports whether the limiter admits a charge of itemCost credits.
	CheckCredit(itemCost float64) bool
}

// ReconfigurableRateLimiter implements a leaky-bucket limiter in credit units. Credits refill on each
// [ReconfigurableRateLimiter.CheckCredit] call proportional to elapsed time, up to creditsPerSecond,
// capped by maxBalance. CheckCredit deducts itemCost when the balance suffices and returns true.
//
// Typical uses include capping events per second (CheckCredit(1.0) per message) or bytes per second
// (creditsPerSecond as throughput; CheckCredit with message size).
type ReconfigurableRateLimiter struct {
	lock sync.Mutex

	creditsPerSecond float64
	balance          float64
	maxBalance       float64
	lastTick         time.Time

	timeNow func() time.Time
}

// NewRateLimiter returns a [ReconfigurableRateLimiter] with the given refill rate and balance cap.
func NewRateLimiter(creditsPerSecond, maxBalance float64) *ReconfigurableRateLimiter {
	return &ReconfigurableRateLimiter{
		creditsPerSecond: creditsPerSecond,
		balance:          maxBalance,
		maxBalance:       maxBalance,
		lastTick:         time.Now(),
		timeNow:          time.Now,
	}
}

// CheckCredit deducts itemCost from the balance when possible and returns whether the charge succeeded.
func (rl *ReconfigurableRateLimiter) CheckCredit(itemCost float64) bool {
	rl.lock.Lock()
	defer rl.lock.Unlock()

	// if we have enough credits to pay for current item, then reduce balance and allow
	if rl.balance >= itemCost {
		rl.balance -= itemCost
		return true
	}
	// otherwise check if balance can be increased due to time elapsed, and try again
	rl.updateBalance()
	if rl.balance >= itemCost {
		rl.balance -= itemCost
		return true
	}
	return false
}

// updateBalance refills credits from elapsed time. Must be called with rl.lock held.
func (rl *ReconfigurableRateLimiter) updateBalance() {
	// calculate how much time passed since the last tick, and update current tick
	currentTime := rl.timeNow()
	elapsedTime := currentTime.Sub(rl.lastTick)
	rl.lastTick = currentTime
	// calculate how much credit have we accumulated since the last tick
	rl.balance += elapsedTime.Seconds() * rl.creditsPerSecond
	if rl.balance > rl.maxBalance {
		rl.balance = rl.maxBalance
	}
}

// Update changes creditsPerSecond and maxBalance in place and scales the current balance to the new cap.
func (rl *ReconfigurableRateLimiter) Update(creditsPerSecond, maxBalance float64) {
	rl.lock.Lock()
	defer rl.lock.Unlock()

	rl.updateBalance() // get up to date balance
	if rl.maxBalance > 0 {
		rl.balance = rl.balance * maxBalance / rl.maxBalance
	} else {
		rl.balance = maxBalance
	}
	rl.creditsPerSecond = creditsPerSecond
	rl.maxBalance = maxBalance
}

// rateLimitingSampler caps trace sampling decisions per second using a [ReconfigurableRateLimiter].
type rateLimitingSampler struct {
	maxTracesPerSecond float64
	rateLimiter        *ReconfigurableRateLimiter
}

// init configures the limiter for maxTracesPerSecond and updates s.maxTracesPerSecond.
func (s *rateLimitingSampler) init(maxTracesPerSecond float64) *rateLimitingSampler {
	if s.rateLimiter == nil {
		s.rateLimiter = NewRateLimiter(maxTracesPerSecond, math.Max(maxTracesPerSecond, 1.0))
	} else {
		s.rateLimiter.Update(maxTracesPerSecond, math.Max(maxTracesPerSecond, 1.0))
	}
	s.maxTracesPerSecond = maxTracesPerSecond
	return s
}

// Description returns a short human-readable sampler name.
func (s *rateLimitingSampler) Description() string {
	return fmt.Sprintf("rateLimitingSampler(maxTracesPerSecond=%v)", s.maxTracesPerSecond)
}

// ShouldSample records the span when the limiter grants one credit for this decision.
func (s *rateLimitingSampler) ShouldSample(p trace.SamplingParameters) trace.SamplingResult {
	if s.rateLimiter.CheckCredit(1.0) {
		return trace.SamplingResult{Decision: trace.RecordAndSample}
	}
	return trace.SamplingResult{Decision: trace.Drop}
}

// RateLimitingSampler returns a [trace.Sampler] that admits at most about maxTracesPerSecond traces per second.
func RateLimitingSampler(maxTracesPerSecond float64) trace.Sampler {
	s := new(rateLimitingSampler)
	return s.init(maxTracesPerSecond)
}

// guaranteedThroughputProbabilitySampler applies ratio sampling then a per-second rate limit.
type guaranteedThroughputProbabilitySampler struct {
	probabilitySampler  trace.Sampler
	rateLimitingSampler trace.Sampler
}

// GuaranteedThroughputProbabilitySampler combines [trace.TraceIDRatioBased] with fraction and a
// [RateLimitingSampler] capped at maxTracesPerSecond traces per second.
func GuaranteedThroughputProbabilitySampler(fraction float64, maxTracesPerSecond float64) trace.Sampler {
	return &guaranteedThroughputProbabilitySampler{
		probabilitySampler:  trace.TraceIDRatioBased(fraction),
		rateLimitingSampler: RateLimitingSampler(maxTracesPerSecond),
	}
}

// ShouldSample returns the ratio sampler result when it drops; otherwise it defers to the rate limiter.
func (s guaranteedThroughputProbabilitySampler) ShouldSample(p trace.SamplingParameters) trace.SamplingResult {
	samplingResult := s.probabilitySampler.ShouldSample(p)
	if samplingResult.Decision == trace.Drop {
		return samplingResult
	}

	return s.rateLimitingSampler.ShouldSample(p)
}

// Description returns a fixed label for logging.
func (s guaranteedThroughputProbabilitySampler) Description() string {
	return "GuaranteedThroughputProbabilitySampler"
}
