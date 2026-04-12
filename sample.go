package ttrace

import (
	"fmt"
	"math"
	"sync"
	"time"

	"go.opentelemetry.io/otel/sdk/trace"
)

// RateLimiter defines a credit-based admission decision.
//
// Deprecated: use [ReconfigurableRateLimiter].
type RateLimiter interface {
	// CheckCredit reports whether itemCost credits may be consumed.
	CheckCredit(itemCost float64) bool
}

// ReconfigurableRateLimiter implements a leaky-bucket rate limiter expressed in abstract credits. The
// balance refills on each [ReconfigurableRateLimiter.CheckCredit] call in proportion to elapsed time,
// at a rate of creditsPerSecond, not exceeding maxBalance. CheckCredit deducts itemCost when the
// balance is sufficient and returns true.
//
// Typical uses include limiting events per second (for example CheckCredit(1.0) per message) or bytes
// per second (treat creditsPerSecond as throughput and pass message size as itemCost).
type ReconfigurableRateLimiter struct {
	lock sync.Mutex

	creditsPerSecond float64
	balance          float64
	maxBalance       float64
	lastTick         time.Time

	timeNow func() time.Time
}

// NewRateLimiter constructs a [ReconfigurableRateLimiter] with the specified refill rate and maximum balance.
func NewRateLimiter(creditsPerSecond, maxBalance float64) *ReconfigurableRateLimiter {
	return &ReconfigurableRateLimiter{
		creditsPerSecond: creditsPerSecond,
		balance:          maxBalance,
		maxBalance:       maxBalance,
		lastTick:         time.Now(),
		timeNow:          time.Now,
	}
}

// CheckCredit attempts to deduct itemCost from the current balance and reports whether the deduction succeeded.
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

// updateBalance accrues credits based on elapsed time since the last tick. rl.lock must be held.
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

// Update replaces the refill rate and balance cap, rescaling the current balance to the new maximum.
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

// rateLimitingSampler enforces a maximum number of trace sampling decisions per second using a [ReconfigurableRateLimiter].
type rateLimitingSampler struct {
	maxTracesPerSecond float64
	rateLimiter        *ReconfigurableRateLimiter
}

// init ensures the internal limiter matches maxTracesPerSecond and updates s.maxTracesPerSecond.
func (s *rateLimitingSampler) init(maxTracesPerSecond float64) *rateLimitingSampler {
	if s.rateLimiter == nil {
		s.rateLimiter = NewRateLimiter(maxTracesPerSecond, math.Max(maxTracesPerSecond, 1.0))
	} else {
		s.rateLimiter.Update(maxTracesPerSecond, math.Max(maxTracesPerSecond, 1.0))
	}
	s.maxTracesPerSecond = maxTracesPerSecond
	return s
}

// Description returns a concise label describing the sampler configuration.
func (s *rateLimitingSampler) Description() string {
	return fmt.Sprintf("rateLimitingSampler(maxTracesPerSecond=%v)", s.maxTracesPerSecond)
}

// ShouldSample returns RecordAndSample when the rate limiter grants one credit for this decision, otherwise Drop.
func (s *rateLimitingSampler) ShouldSample(p trace.SamplingParameters) trace.SamplingResult {
	if s.rateLimiter.CheckCredit(1.0) {
		return trace.SamplingResult{Decision: trace.RecordAndSample}
	}
	return trace.SamplingResult{Decision: trace.Drop}
}

// RateLimitingSampler returns a [trace.Sampler] that records at most approximately maxTracesPerSecond root spans per second.
func RateLimitingSampler(maxTracesPerSecond float64) trace.Sampler {
	s := new(rateLimitingSampler)
	return s.init(maxTracesPerSecond)
}

// guaranteedThroughputProbabilitySampler applies trace ID ratio sampling, then a per-second rate limit.
type guaranteedThroughputProbabilitySampler struct {
	probabilitySampler  trace.Sampler
	rateLimitingSampler trace.Sampler
}

// GuaranteedThroughputProbabilitySampler chains [trace.TraceIDRatioBased] sampling with fraction and a
// [RateLimitingSampler] limited to maxTracesPerSecond traces per second after the ratio stage.
func GuaranteedThroughputProbabilitySampler(fraction float64, maxTracesPerSecond float64) trace.Sampler {
	return &guaranteedThroughputProbabilitySampler{
		probabilitySampler:  trace.TraceIDRatioBased(fraction),
		rateLimitingSampler: RateLimitingSampler(maxTracesPerSecond),
	}
}

// ShouldSample returns the probability sampler result when that stage drops the span; otherwise it
// delegates to the rate-limiting sampler.
func (s guaranteedThroughputProbabilitySampler) ShouldSample(p trace.SamplingParameters) trace.SamplingResult {
	samplingResult := s.probabilitySampler.ShouldSample(p)
	if samplingResult.Decision == trace.Drop {
		return samplingResult
	}

	return s.rateLimitingSampler.ShouldSample(p)
}

// Description returns the constant string "GuaranteedThroughputProbabilitySampler".
func (s guaranteedThroughputProbabilitySampler) Description() string {
	return "GuaranteedThroughputProbabilitySampler"
}
