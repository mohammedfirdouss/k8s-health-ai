package llm

import (
	"context"
	"os"
	"strconv"
	"strings"

	"golang.org/x/time/rate"
)

// RateLimitedProvider wraps a Provider and enforces a requests-per-minute ceiling via golang.org/x/time/rate.
// Env LLM_RPM: default 120; if 0 or negative, limiting is skipped (inner is called directly).
type RateLimitedProvider struct {
	Inner   Provider
	Limiter *rate.Limiter
}

// Diagnose implements Provider.
func (r *RateLimitedProvider) Diagnose(ctx context.Context, fc FailureContext) (Diagnosis, error) {
	if r.Limiter != nil {
		if err := r.Limiter.Wait(ctx); err != nil {
			return Diagnosis{}, err
		}
	}
	return r.Inner.Diagnose(ctx, fc)
}

// WrapWithLLMRateLimit wraps inner when LLM_RPM is positive (default 120 when unset).
// If LLM_RPM is 0 or negative, returns inner without wrapping.
func WrapWithLLMRateLimit(inner Provider) Provider {
	rpm := 120
	if s := strings.TrimSpace(os.Getenv("LLM_RPM")); s != "" {
		if n, err := strconv.Atoi(s); err == nil {
			rpm = n
		}
	}
	if rpm <= 0 {
		return inner
	}
	lim := rate.NewLimiter(rate.Limit(float64(rpm)/60.0), 1)
	return &RateLimitedProvider{Inner: inner, Limiter: lim}
}
