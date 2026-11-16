package ttrace

import (
	"fmt"
	"math"
	"sync"
	"time"

	"go.opentelemetry.io/otel/sdk/trace"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// RateLimiter reports whether a given cost may be admitted by a credit-based limiter.
//
// Deprecated: use [ReconfigurableRateLimiter].
type RateLimiter interface {
	// CheckCredit reports whether itemCost credits may be consumed.
	CheckCredit(itemCost float64) bool
}

// ReconfigurableRateLimiter implements a token bucket expressed in abstract credits.
// On each [ReconfigurableRateLimiter.CheckCredit] call, the balance refills in proportion to
// elapsed time at a rate of creditsPerSecond, up to maxBalance. CheckCredit deducts itemCost when
// sufficient balance is available and reports whether the deduction succeeded.
//
// Typical uses include limiting events per second (for example, calling CheckCredit(1.0) per
// message) or bytes per second (treat creditsPerSecond as throughput and pass message size as
// itemCost).
type ReconfigurableRateLimiter struct {
	lock sync.Mutex

	creditsPerSecond float64
	balance          float64
	maxBalance       float64
	lastTick         time.Time

	timeNow func() time.Time
}

// NewRateLimiter constructs a [ReconfigurableRateLimiter] with the specified refill rate and maximum
// balance. Invalid (negative, NaN, or infinite) configuration disables the limiter
// until a valid [ReconfigurableRateLimiter.Update] call. The zero value is disabled.
func NewRateLimiter(creditsPerSecond, maxBalance float64) *ReconfigurableRateLimiter {
	limiter := &ReconfigurableRateLimiter{
		lastTick: time.Now(),
		timeNow:  time.Now,
	}

	limiter.Update(creditsPerSecond, maxBalance)

	return limiter
}

// CheckCredit attempts to deduct itemCost from the current balance and reports whether the
// deduction succeeded. Negative, NaN, or infinite costs are rejected without changing the balance.
func (limiter *ReconfigurableRateLimiter) CheckCredit(itemCost float64) bool {
	if math.IsNaN(itemCost) || math.IsInf(itemCost, 0) || itemCost < 0 {
		return false
	}

	limiter.lock.Lock()
	defer limiter.lock.Unlock()

	// Refill before every deduction so idle time cannot be credited twice.
	limiter.updateBalance()
	if limiter.balance >= itemCost {
		limiter.balance -= itemCost

		return true
	}

	return false
}

// Update replaces the refill rate and balance cap, rescaling the current balance to the new maximum.
// Invalid configuration clears the balance and disables the limiter until the next valid update.
func (limiter *ReconfigurableRateLimiter) Update(creditsPerSecond, maxBalance float64) {
	limiter.lock.Lock()
	defer limiter.lock.Unlock()

	if math.IsNaN(creditsPerSecond) || math.IsInf(creditsPerSecond, 0) || creditsPerSecond < 0 || math.IsNaN(maxBalance) || math.IsInf(maxBalance, 0) || maxBalance < 0 {
		creditsPerSecond = 0
		maxBalance = 0
	}

	limiter.updateBalance()
	// Keep an unchanged capacity exact; repeated rescaling can otherwise lose fractional credit.
	if limiter.maxBalance != maxBalance {
		if limiter.maxBalance > 0 {
			// Divide first to avoid overflowing when both balance caps are large but finite.
			limiter.balance = limiter.balance / limiter.maxBalance * maxBalance
		} else {
			limiter.balance = maxBalance
		}
	}

	limiter.creditsPerSecond = creditsPerSecond
	limiter.maxBalance = maxBalance
}

// RateLimitingSampler returns a parent-based [trace.Sampler] that limits sampled root traces to
// approximately maxTracesPerSecond per second. Zero or invalid rates drop all root traces;
// valid parent sampling decisions are still inherited.
func RateLimitingSampler(maxTracesPerSecond float64) trace.Sampler {
	return trace.ParentBased(newRateLimitingRootSampler(maxTracesPerSecond))
}

// GuaranteedThroughputProbabilitySampler returns a parent-based [trace.Sampler] that first applies
// [trace.TraceIDRatioBased] to root traces and then enforces a throughput cap. Invalid fractions
// outside [0, 1], invalid rates, and zero rates drop root traces without affecting valid parents.
func GuaranteedThroughputProbabilitySampler(fraction, maxTracesPerSecond float64) trace.Sampler {
	if math.IsNaN(fraction) || math.IsInf(fraction, 0) || fraction < 0 || fraction > 1 {
		return trace.ParentBased(trace.NeverSample())
	}

	rootSampler := &guaranteedThroughputProbabilitySampler{
		probabilitySampler:  trace.TraceIDRatioBased(fraction),
		rateLimitingSampler: newRateLimitingRootSampler(maxTracesPerSecond),
	}

	return trace.ParentBased(rootSampler)
}

// updateBalance accrues credits up to the capacity. limiter.lock must be held.
func (limiter *ReconfigurableRateLimiter) updateBalance() {
	currentTime := time.Now()
	if limiter.timeNow != nil {
		currentTime = limiter.timeNow()
	}

	if !currentTime.After(limiter.lastTick) {
		return
	}

	elapsedTime := currentTime.Sub(limiter.lastTick)
	limiter.lastTick = currentTime
	limiter.balance = math.Min(limiter.maxBalance, limiter.balance+elapsedTime.Seconds()*limiter.creditsPerSecond)
}

// rateLimitingSampler limits root sampling with a refill rate and burst capacity through
// [ReconfigurableRateLimiter].
type rateLimitingSampler struct {
	maxTracesPerSecond float64
	rateLimiter        *ReconfigurableRateLimiter
}

// Description returns a concise label describing the sampler configuration.
func (sampler *rateLimitingSampler) Description() string {
	return fmt.Sprintf("rateLimitingSampler(maxTracesPerSecond=%v)", sampler.maxTracesPerSecond)
}

// ShouldSample returns RecordAndSample when the rate limiter grants one credit for this decision, otherwise Drop.
func (sampler *rateLimitingSampler) ShouldSample(parameters trace.SamplingParameters) trace.SamplingResult {
	samplingResult := trace.SamplingResult{
		Decision:   trace.Drop,
		Tracestate: oteltrace.SpanContextFromContext(parameters.ParentContext).TraceState(),
	}

	if sampler.rateLimiter.CheckCredit(1.0) {
		samplingResult.Decision = trace.RecordAndSample
	}

	return samplingResult
}

// newRateLimitingRootSampler creates a shared root policy, disabling invalid or zero rates.
func newRateLimitingRootSampler(maxTracesPerSecond float64) trace.Sampler {
	if math.IsNaN(maxTracesPerSecond) || math.IsInf(maxTracesPerSecond, 0) || maxTracesPerSecond <= 0 {
		return trace.NeverSample()
	}

	rootSampler := &rateLimitingSampler{
		maxTracesPerSecond: maxTracesPerSecond,
		rateLimiter:        NewRateLimiter(maxTracesPerSecond, math.Max(maxTracesPerSecond, 1.0)),
	}

	return rootSampler
}

// guaranteedThroughputProbabilitySampler applies trace ID ratio sampling, then a per-second rate limit.
type guaranteedThroughputProbabilitySampler struct {
	probabilitySampler  trace.Sampler
	rateLimitingSampler trace.Sampler
}

// ShouldSample returns the probability sampler result when that stage drops the span; otherwise it
// delegates to the rate limiter while preserving sampling attributes and tracestate.
func (sampler guaranteedThroughputProbabilitySampler) ShouldSample(parameters trace.SamplingParameters) trace.SamplingResult {
	samplingResult := sampler.probabilitySampler.ShouldSample(parameters)
	if samplingResult.Decision == trace.Drop {
		return samplingResult
	}

	samplingResult.Decision = sampler.rateLimitingSampler.ShouldSample(parameters).Decision

	return samplingResult
}

// Description returns the constant string "GuaranteedThroughputProbabilitySampler".
func (sampler guaranteedThroughputProbabilitySampler) Description() string {
	return "GuaranteedThroughputProbabilitySampler"
}
