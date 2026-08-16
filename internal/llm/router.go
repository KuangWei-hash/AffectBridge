package llm

import (
	"context"
	"errors"
)

// Router tries providers in order. On any non-nil error, it falls
// through to the next provider. If all fail, the last error is
// returned.
//
// This is the multi-provider failover design documented in
// docs/計畫中工作/如何利用LLM.md. Pair each entry with a Limiter so
// capacity exhaustion on one provider is also a fallback signal:
//
//	NewRouter(
//	    NewLimiter(openaiClient, 32),
//	    NewLimiter(claudeClient, 32),
//	    NewLimiter(ollamaClient,  8),
//	)
type Router struct {
	providers []Client
}

func NewRouter(providers ...Client) *Router {
	return &Router{providers: providers}
}

func (r *Router) Complete(ctx context.Context, prompt string, opts ...Option) (string, error) {
	if len(r.providers) == 0 {
		return "", errors.New("llm: router has no providers")
	}
	var lastErr error
	for _, p := range r.providers {
		resp, err := p.Complete(ctx, prompt, opts...)
		if err == nil {
			return resp, nil
		}
		lastErr = err
	}
	return "", lastErr
}
