// Command honorscope prints a GitHub user's public Achievements and Highlights.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/Saku0512/honorscope"
)

const (
	exitOK          = 0
	exitError       = 1
	exitNotFound    = 2
	exitRateLimited = 3
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("honorscope", flag.ContinueOnError)
	fs.SetOutput(stderr)
	jsonOut := fs.Bool("json", false, "write compact JSON to stdout")
	pretty := fs.Bool("pretty", false, "write indented JSON to stdout")
	timeout := fs.Duration("timeout", 20*time.Second, "request timeout, including rate-limit retries")
	userAgent := fs.String("user-agent", "", "User-Agent (default identifies honorscope)")
	baseURL := fs.String("base-url", "", "profile origin (default https://github.com)")
	version := fs.Bool("version", false, "print version and exit")
	fs.Usage = func() {
		_, _ = fmt.Fprintf(stderr, "honorscope reads public GitHub Achievements and Highlights from profile HTML.\n\n")
		_, _ = fmt.Fprintf(stderr, "Usage:\n  honorscope <user> [--json] [--pretty]\n\n")
		_, _ = fmt.Fprintf(stderr, "Exit codes: 0 ok, 1 error, 2 user not found, 3 rate limited.\n\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(normalizeArgs(args)); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return exitOK
		}
		return exitError
	}
	if *version {
		_, _ = fmt.Fprintln(stdout, honorscope.Version)
		return exitOK
	}
	if fs.NArg() != 1 {
		fs.Usage()
		return exitError
	}

	opts := &honorscope.Options{Timeout: *timeout}
	if *userAgent != "" {
		opts.UserAgent = *userAgent
	}
	if *baseURL != "" {
		opts.BaseURL = *baseURL
	}
	profile, err := honorscope.Fetch(context.Background(), fs.Arg(0), opts)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "%s\n", err.Error())
		switch {
		case errors.Is(err, honorscope.ErrUserNotFound):
			return exitNotFound
		case errors.Is(err, honorscope.ErrRateLimited):
			return exitRateLimited
		default:
			return exitError
		}
	}
	if *pretty || *jsonOut {
		enc := json.NewEncoder(stdout)
		enc.SetEscapeHTML(false)
		if *pretty {
			enc.SetIndent("", "  ")
		}
		if err := enc.Encode(profile); err != nil {
			_, _ = fmt.Fprintf(stderr, "honorscope: %s\n", err.Error())
			return exitError
		}
		return exitOK
	}
	if err := writeText(stdout, profile); err != nil {
		_, _ = fmt.Fprintf(stderr, "honorscope: %s\n", err.Error())
		return exitError
	}
	return exitOK
}

func writeText(w io.Writer, p *honorscope.Profile) error {
	var err error
	printf := func(format string, args ...any) {
		if err != nil {
			return
		}
		_, err = fmt.Fprintf(w, format, args...)
	}
	printf("%s\n", p.User)
	if !p.FetchedAt.IsZero() {
		printf("fetchedAt: %s\n", p.FetchedAt.Format(time.RFC3339))
	}
	printf("source: %s\n\n", p.Source)
	printf("Achievements (%d)\n", len(p.Achievements))
	if len(p.Achievements) == 0 {
		printf("(none)\n")
	}
	for _, a := range p.Achievements {
		if a.Tier != nil {
			printf("- %s (%s) [%s]\n", a.Name, a.Slug, *a.Tier)
		} else {
			printf("- %s (%s)\n", a.Name, a.Slug)
		}
	}
	printf("\nHighlights (%d)\n", len(p.Highlights))
	if len(p.Highlights) == 0 {
		printf("(none)\n")
	}
	for _, h := range p.Highlights {
		printf("- %s (%s)\n", h.Name, h.Slug)
	}
	return err
}

// normalizeArgs lets the login appear before or after flags, matching
// `honorscope <user> [--json] [--pretty]`.
func normalizeArgs(args []string) []string {
	var flags, pos []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--" {
			pos = append(pos, args[i+1:]...)
			break
		}
		if !strings.HasPrefix(a, "-") || a == "-" {
			pos = append(pos, a)
			continue
		}
		flags = append(flags, a)
		name := strings.TrimLeft(a, "-")
		if eq := strings.IndexByte(name, '='); eq >= 0 {
			continue
		}
		if flagTakesValue(name) && i+1 < len(args) {
			i++
			flags = append(flags, args[i])
		}
	}
	return append(flags, pos...)
}

func flagTakesValue(name string) bool {
	switch name {
	case "timeout", "user-agent", "base-url":
		return true
	default:
		return false
	}
}
