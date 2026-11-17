package honorscope

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestFetchProfile(t *testing.T) {
	body := readFixture(t, "saku0512-profile.html")
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		if r.URL.RawQuery != "" {
			t.Errorf("query %q; profile fetch must not use ?tab=", r.URL.RawQuery)
		}
		if r.URL.Path != "/Saku0512" {
			t.Errorf("path %s", r.URL.Path)
		}
		if c := r.Header.Get("Cookie"); c != "" {
			t.Errorf("cookie %q", c)
		}
		if r.Header.Get("Authorization") != "" {
			t.Error("authorization header sent")
		}
		if !strings.Contains(r.Header.Get("User-Agent"), "honorscope/") {
			t.Errorf("user agent %q", r.Header.Get("User-Agent"))
		}
		if !strings.Contains(r.Header.Get("Accept-Language"), "en") {
			t.Errorf("accept-language %q", r.Header.Get("Accept-Language"))
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	when := time.Date(2026, 11, 17, 4, 41, 45, 0, time.UTC)
	p, err := Fetch(context.Background(), "Saku0512", &Options{
		BaseURL:   srv.URL,
		Timeout:   5 * time.Second,
		Now:       func() time.Time { return when },
		UserAgent: "",
	})
	if err != nil {
		t.Fatal(err)
	}
	if hits.Load() != 1 {
		t.Fatalf("requests %d, want 1", hits.Load())
	}
	if !p.FetchedAt.Equal(when) {
		t.Fatalf("fetchedAt %s", p.FetchedAt)
	}
	if p.User != "Saku0512" || len(p.Achievements) != 4 || p.Source != SourcePublicHTML {
		t.Fatalf("%+v", p)
	}
}

func TestFetchNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer srv.Close()

	_, err := Fetch(context.Background(), "missing-user", &Options{BaseURL: srv.URL, MaxRetries: -1})
	if !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("err %v", err)
	}
	var reqErr *RequestError
	if !errors.As(err, &reqErr) || reqErr.StatusCode != http.StatusNotFound {
		t.Fatalf("err %#v", err)
	}
}

func TestFetchRateLimitRetriesThenSucceeds(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if hits.Add(1) == 1 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		_, _ = io.WriteString(w, `<meta property="profile:username" content="ada" />`)
	}))
	defer srv.Close()

	p, err := Fetch(context.Background(), "ada", &Options{
		BaseURL:      srv.URL,
		MaxRetries:   2,
		RetryBase:    time.Millisecond,
		MaxRetryWait: time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	if hits.Load() != 2 {
		t.Fatalf("requests %d", hits.Load())
	}
	if p.User != "ada" || len(p.Achievements) != 0 {
		t.Fatalf("%+v", p)
	}
}

func TestFetchRateLimitStopsWhenRetryAfterIsTooLong(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.Header().Set("Retry-After", "120")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	start := time.Now()
	_, err := Fetch(context.Background(), "ada", &Options{
		BaseURL:      srv.URL,
		MaxRetries:   4,
		MaxRetryWait: 20 * time.Millisecond,
	})
	if !errors.Is(err, ErrRateLimited) {
		t.Fatalf("err %v", err)
	}
	if hits.Load() != 1 {
		t.Fatalf("requests %d, want 1", hits.Load())
	}
	if time.Since(start) > time.Second {
		t.Fatal("waited on a Retry-After above the cap")
	}
}

func TestFetchForbiddenWithRetryAfterIsRateLimited(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "60")
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()
	_, err := Fetch(context.Background(), "ada", &Options{
		BaseURL:      srv.URL,
		MaxRetries:   1,
		MaxRetryWait: time.Millisecond,
	})
	if !errors.Is(err, ErrRateLimited) {
		t.Fatalf("err %v", err)
	}
}

func TestFetchDoesNotFollowOffHostRedirect(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "https://example.com/phish", http.StatusFound)
	}))
	defer srv.Close()
	_, err := Fetch(context.Background(), "ada", &Options{BaseURL: srv.URL, MaxRetries: -1})
	if err == nil {
		t.Fatal("expected redirect error")
	}
	if errors.Is(err, ErrUserNotFound) {
		t.Fatal("off-host redirect reported as not found")
	}
}

func TestFetchFollowsSameHostRename(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/OldName", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/NewName", http.StatusFound)
	})
	mux.HandleFunc("/NewName", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Cookie") != "" || r.Header.Get("Authorization") != "" {
			http.Error(w, "auth leaked", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		_, _ = io.WriteString(w, `<meta property="profile:username" content="NewName" />`)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	p, err := Fetch(context.Background(), "OldName", &Options{BaseURL: srv.URL, MaxRetries: -1})
	if err != nil {
		t.Fatal(err)
	}
	if p.User != "NewName" {
		t.Fatalf("user %q", p.User)
	}
}

func TestFetchInvalidUsername(t *testing.T) {
	_, err := Fetch(context.Background(), "../admin", nil)
	if !errors.Is(err, ErrInvalidUsername) {
		t.Fatalf("err %v", err)
	}
}

func TestFetchTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer srv.Close()
	ctx := context.Background()
	_, err := Fetch(ctx, "ada", &Options{BaseURL: srv.URL, Timeout: 150 * time.Millisecond, MaxRetries: -1})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err %v", err)
	}
}

func TestFetchUnexpectedStatus(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	_, err := Fetch(context.Background(), "ada", &Options{BaseURL: srv.URL, MaxRetries: 3})
	var reqErr *RequestError
	if !errors.As(err, &reqErr) || reqErr.StatusCode != http.StatusInternalServerError {
		t.Fatalf("err %#v", err)
	}
	if hits.Load() != 1 {
		t.Fatalf("500 was retried %d times", hits.Load())
	}
}

func TestFetchRejectsNonHTML(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"ok":true}`)
	}))
	defer srv.Close()
	_, err := Fetch(context.Background(), "ada", &Options{BaseURL: srv.URL, MaxRetries: -1})
	if err == nil || !strings.Contains(err.Error(), "content type") {
		t.Fatalf("err %v", err)
	}
}
