package honorscope

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const maxProfileBytes = 8 << 20

// Fetch loads the public profile for username and parses Achievements and
// Highlights. It performs a single GET of {BaseURL}/{username} unless GitHub
// responds with 429 (or 403 plus Retry-After), in which case it backs off and
// retries. It does not request ?tab=achievements: robots.txt disallows ?tab=
// for a generic user agent, and the profile sidebar already lists the public
// badges. Hidden achievements are not in that HTML, so they are not returned.
func Fetch(ctx context.Context, username string, opts *Options) (*Profile, error) {
	p, _, err := fetch(ctx, username, opts)
	return p, err
}

func fetch(ctx context.Context, username string, opts *Options) (*Profile, []byte, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	username = strings.TrimSpace(username)
	if !validUsername(username) {
		return nil, nil, fmt.Errorf("%w: %q", ErrInvalidUsername, username)
	}
	o := resolveOptions(opts)
	pageURL, err := profileURL(o.BaseURL, username)
	if err != nil {
		return nil, nil, err
	}
	if o.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, o.Timeout)
		defer cancel()
	}

	client := o.HTTPClient
	if client == nil {
		client = defaultClient(pageURL)
	}

	attempts := o.MaxRetries + 1
	for attempt := 0; attempt < attempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return nil, nil, err
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, pageURL, nil)
		if err != nil {
			return nil, nil, err
		}
		prepareRequest(req, o.UserAgent)

		resp, err := client.Do(req)
		if err != nil {
			if ctx.Err() != nil {
				return nil, nil, ctx.Err()
			}
			return nil, nil, fmt.Errorf("honorscope: GET %s: %w", pageURL, err)
		}

		body, readErr := readLimited(resp.Body, maxProfileBytes)
		closeErr := resp.Body.Close()
		if readErr != nil {
			return nil, nil, readErr
		}
		if closeErr != nil {
			return nil, nil, fmt.Errorf("honorscope: close response: %w", closeErr)
		}

		loc := finalURL(resp, pageURL)
		switch {
		case resp.StatusCode == http.StatusOK:
			if ct := resp.Header.Get("Content-Type"); ct != "" && !isHTML(ct) {
				return nil, body, &RequestError{
					URL:        loc,
					StatusCode: resp.StatusCode,
					Err:        fmt.Errorf("unexpected content type %q", ct),
				}
			}
			p, err := Parse(loc, bytes.NewReader(body))
			if err != nil {
				return nil, body, err
			}
			if p.User == "" {
				p.User = username
			}
			p.FetchedAt = o.Now().UTC().Truncate(time.Second)
			p.Source = SourcePublicHTML
			return p, body, nil
		case resp.StatusCode == http.StatusNotFound:
			return nil, body, &RequestError{URL: loc, StatusCode: http.StatusNotFound, Err: ErrUserNotFound}
		case rateLimited(resp):
			if attempt+1 >= attempts {
				return nil, body, &RequestError{URL: loc, StatusCode: resp.StatusCode, Err: ErrRateLimited}
			}
			wait, ok := retryWait(resp, attempt, o.RetryBase, o.MaxRetryWait)
			if !ok {
				return nil, body, &RequestError{URL: loc, StatusCode: resp.StatusCode, Err: ErrRateLimited}
			}
			if err := sleepCtx(ctx, wait); err != nil {
				return nil, body, err
			}
		default:
			return nil, body, &RequestError{URL: loc, StatusCode: resp.StatusCode}
		}
	}
	return nil, nil, &RequestError{URL: pageURL, StatusCode: http.StatusTooManyRequests, Err: ErrRateLimited}
}

func profileURL(base, username string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(base))
	if err != nil {
		return "", fmt.Errorf("honorscope: base url: %w", err)
	}
	if (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return "", fmt.Errorf("honorscope: base url must be http(s) with a host")
	}
	u.Path = strings.TrimRight(u.Path, "/") + "/" + url.PathEscape(username)
	u.RawQuery = ""
	u.Fragment = ""
	u.User = nil
	return u.String(), nil
}

func defaultClient(pageURL string) *http.Client {
	host := ""
	if u, err := url.Parse(pageURL); err == nil {
		host = u.Host
	}
	transport, ok := http.DefaultTransport.(*http.Transport)
	rt := http.DefaultTransport
	if ok {
		rt = transport.Clone()
	}
	return &http.Client{
		Transport: rt,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("stopped after 10 redirects")
			}
			if host != "" && !strings.EqualFold(req.URL.Host, host) {
				return fmt.Errorf("refusing redirect to %s", req.URL.Host)
			}
			req.Header.Del("Cookie")
			req.Header.Del("Authorization")
			return nil
		},
	}
}

func prepareRequest(req *http.Request, userAgent string) {
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml;q=0.9")
	// English markup keeps achievement names and the Highlights heading stable.
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Del("Cookie")
	req.Header.Del("Authorization")
}

func readLimited(r io.Reader, limit int64) ([]byte, error) {
	b, err := io.ReadAll(io.LimitReader(r, limit+1))
	if err != nil {
		return nil, fmt.Errorf("honorscope: read profile: %w", err)
	}
	if int64(len(b)) > limit {
		return nil, fmt.Errorf("honorscope: profile HTML exceeds %d bytes", limit)
	}
	return b, nil
}

func isHTML(contentType string) bool {
	ct := strings.ToLower(contentType)
	return strings.Contains(ct, "html") || strings.Contains(ct, "xml")
}

func rateLimited(resp *http.Response) bool {
	if resp.StatusCode == http.StatusTooManyRequests {
		return true
	}
	return resp.StatusCode == http.StatusForbidden && resp.Header.Get("Retry-After") != ""
}

// retryWait reports how long to sleep before another attempt.
// ok is false when Retry-After asks for longer than maxWait.
func retryWait(resp *http.Response, attempt int, base, maxWait time.Duration) (time.Duration, bool) {
	if v := strings.TrimSpace(resp.Header.Get("Retry-After")); v != "" {
		if secs, err := strconv.Atoi(v); err == nil {
			if secs < 0 {
				secs = 0
			}
			d := time.Duration(secs) * time.Second
			if d > maxWait {
				return 0, false
			}
			return d, true
		}
		if when, err := http.ParseTime(v); err == nil {
			d := time.Until(when)
			if d < 0 {
				d = 0
			}
			if d > maxWait {
				return 0, false
			}
			return d, true
		}
	}
	d := base
	for i := 0; i < attempt; i++ {
		d *= 2
		if d > maxWait {
			return maxWait, true
		}
	}
	if d > maxWait {
		return maxWait, true
	}
	return d, true
}

func sleepCtx(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func finalURL(resp *http.Response, fallback string) string {
	if resp != nil && resp.Request != nil && resp.Request.URL != nil {
		return resp.Request.URL.String()
	}
	return fallback
}
