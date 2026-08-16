package llm

import "context"

// Limiter caps the number of in-flight Complete calls against the
// wrapped client. When the cap is reached, Complete returns ErrBusy
// immediately rather than queueing. This is fail-fast, not backpressure.
//
// maxConcurrent <= 0 disables the limit (no semaphore, passthrough).
type Limiter struct {
	inner Client
	sem   chan struct{} // buffered channel used as a counting semaphore
}

func NewLimiter(inner Client, maxConcurrent int) *Limiter {
	if maxConcurrent <= 0 {
		return &Limiter{inner: inner}
	}
	return &Limiter{
		inner: inner,
		sem:   make(chan struct{}, maxConcurrent),
	}
}

func (l *Limiter) Complete(ctx context.Context, prompt string, opts ...Option) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if l.sem == nil {
		return l.inner.Complete(ctx, prompt, opts...)
	}
	select {
	case l.sem <- struct{}{}:
		defer func() { <-l.sem }()
		return l.inner.Complete(ctx, prompt, opts...)
	default:
		return "", ErrBusy
	}
}
