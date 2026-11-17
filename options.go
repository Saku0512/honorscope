package honorscope

import (
	"net/http"
	"time"
)

const (
	defaultBaseURL      = "https://github.com"
	defaultTimeout      = 20 * time.Second
	defaultMaxRetries   = 2
	defaultRetryBase    = 500 * time.Millisecond
	defaultMaxRetryWait = 10 * time.Second
)

// Options configures Fetch. Nil options use polite defaults: one profile
// request, a 20s timeout, an identifying User-Agent, and a short backoff if
// GitHub responds with 429.
type Options struct {
	// BaseURL is the site origin. Tests point this at httptest. Default https://github.com.
	BaseURL string

	// UserAgent is sent on every request. Empty uses an identifying honorscope UA.
	UserAgent string

	// HTTPClient overrides the default client. Requests still strip Cookie and
	// Authorization headers. The client should not enable a cookie jar.
	HTTPClient *http.Client

	// Timeout bounds the whole Fetch call, including retries. Zero uses 20s.
	Timeout time.Duration

	// MaxRetries is how many additional attempts to make after HTTP 429
	// (or 403 with Retry-After). Zero uses 2. Negative disables retries.
	MaxRetries int

	// RetryBase is the backoff used when Retry-After is absent. Zero uses 500ms.
	RetryBase time.Duration

	// MaxRetryWait caps a single wait. A Retry-After above the cap stops the
	// call with ErrRateLimited instead of sleeping. Zero uses 10s.
	MaxRetryWait time.Duration

	// Now overrides the clock used for FetchedAt. Zero uses time.Now.
	Now func() time.Time
}

func resolveOptions(opts *Options) Options {
	var o Options
	if opts != nil {
		o = *opts
	}
	if o.BaseURL == "" {
		o.BaseURL = defaultBaseURL
	}
	if o.UserAgent == "" {
		o.UserAgent = defaultUserAgent()
	}
	if o.Timeout == 0 {
		o.Timeout = defaultTimeout
	}
	if o.MaxRetries == 0 {
		o.MaxRetries = defaultMaxRetries
	}
	if o.MaxRetries < 0 {
		o.MaxRetries = 0
	}
	if o.RetryBase <= 0 {
		o.RetryBase = defaultRetryBase
	}
	if o.MaxRetryWait <= 0 {
		o.MaxRetryWait = defaultMaxRetryWait
	}
	if o.Now == nil {
		o.Now = time.Now
	}
	return o
}

func defaultUserAgent() string {
	return "honorscope/" + Version + " (+https://github.com/Saku0512/honorscope)"
}
