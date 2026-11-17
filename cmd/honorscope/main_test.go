package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCLIJSONAfterUser(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/ada" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<meta property="profile:username" content="ada" />
<a href="/ada?achievement=yolo"><img class="achievement-badge-sidebar" alt="Achievement: YOLO" src="https://example.com/yolo.png" /></a>`))
	}))
	defer srv.Close()

	var stdout, stderr bytes.Buffer
	code := run([]string{"ada", "--json", "--base-url", srv.URL, "--timeout", "5s"}, &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("code %d stderr %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"slug":"yolo"`) || !strings.Contains(stdout.String(), `"tier":null`) {
		t.Fatalf("stdout %s", stdout.String())
	}
	if strings.Contains(stdout.String(), "\n  ") {
		t.Fatalf("compact JSON was indented: %s", stdout.String())
	}
}

func TestCLIPrettyBeforeUser(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<meta property="profile:username" content="ada" /><h2>Highlights</h2><ul><li><span class="text-uppercase">Pro</span></li></ul>`))
	}))
	defer srv.Close()

	var stdout, stderr bytes.Buffer
	code := run([]string{"--pretty", "--base-url=" + srv.URL, "ada"}, &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("code %d stderr %s", code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "\n  ") || !strings.Contains(out, `"name": "PRO"`) {
		t.Fatalf("stdout %s", out)
	}
}

func TestCLINotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer srv.Close()
	var stdout, stderr bytes.Buffer
	code := run([]string{"--base-url", srv.URL, "missing-user"}, &stdout, &stderr)
	if code != exitNotFound {
		t.Fatalf("code %d stderr %s", code, stderr.String())
	}
}

func TestCLIUsage(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := run(nil, &stdout, &stderr); code != exitError {
		t.Fatalf("code %d", code)
	}
	if !strings.Contains(stderr.String(), "honorscope <user>") {
		t.Fatalf("usage %s", stderr.String())
	}
}
