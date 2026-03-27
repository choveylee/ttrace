package ttrace

import (
	"testing"
	"time"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func TestReconfigurableRateLimiter_Update_ZeroOldMaxBalance(t *testing.T) {
	t.Parallel()

	rl := NewRateLimiter(1.0, 0)
	rl.timeNow = func() time.Time { return time.Unix(0, 0) }

	// should not panic when maxBalance was 0
	rl.Update(2.0, 100)
	if rl.maxBalance != 100 {
		t.Fatalf("maxBalance: got %v", rl.maxBalance)
	}
	if rl.balance != 100 {
		t.Fatalf("balance after update from zero max: got %v want 100", rl.balance)
	}
}

func TestRateLimitingSampler_Systematic(t *testing.T) {
	t.Parallel()

	const maxPerSec = 100.0
	s := RateLimitingSampler(maxPerSec)
	var sampled int
	const n = 200
	for i := 0; i < n; i++ {
		r := s.ShouldSample(sdktrace.SamplingParameters{})
		if r.Decision == sdktrace.RecordAndSample {
			sampled++
		}
	}
	if sampled == 0 || sampled == n {
		t.Fatalf("expected partial sampling, got %d/%d", sampled, n)
	}
}

func TestGuaranteedThroughputProbabilitySampler_DropsByRate(t *testing.T) {
	t.Parallel()

	s := GuaranteedThroughputProbabilitySampler(0.0, 100)
	r := s.ShouldSample(sdktrace.SamplingParameters{})
	if r.Decision != sdktrace.Drop {
		t.Fatalf("fraction 0 should drop first stage: %v", r.Decision)
	}
}

func TestGuaranteedThroughputProbabilitySampler_Description(t *testing.T) {
	t.Parallel()

	s := GuaranteedThroughputProbabilitySampler(0.5, 10)
	if s.Description() == "" {
		t.Fatal("want non-empty description")
	}
}
