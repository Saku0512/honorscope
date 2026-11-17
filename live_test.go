package honorscope

import (
	"context"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"
)

// TestLivePublicProfiles fetches two real public profiles when HONORSCOPE_LIVE=1.
// It fails if the achievement markers in the HTML no longer match what Parse
// returns, which is the signal that GitHub changed the DOM.
func TestLivePublicProfiles(t *testing.T) {
	if os.Getenv("HONORSCOPE_LIVE") == "" {
		t.Skip("set HONORSCOPE_LIVE=1 to fetch public profiles and detect DOM changes")
	}

	users := []string{"Saku0512", "torvalds"}
	for i, user := range users {
		if i > 0 {
			time.Sleep(2 * time.Second)
		}
		t.Run(user, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			p, raw, err := fetch(ctx, user, &Options{Timeout: 30 * time.Second, MaxRetries: 2})
			if err != nil {
				t.Fatalf("fetch %s: %v", user, err)
			}
			if p.Source != SourcePublicHTML {
				t.Fatalf("source %q", p.Source)
			}
			if !strings.EqualFold(p.User, user) {
				t.Fatalf("user %q", p.User)
			}
			html := string(raw)
			markers := liveAchievementSlugs(html)
			if len(markers) == 0 || !strings.Contains(html, "achievement-badge-sidebar") {
				t.Fatalf("DOM change: public profile for %s has no achievement-badge-sidebar markers", user)
			}
			got := map[string]struct{}{}
			for _, a := range p.Achievements {
				if a.Slug == "" || a.Name == "" {
					t.Fatalf("incomplete achievement %#v", a)
				}
				if a.Tier != nil && !tierRE.MatchString(*a.Tier) {
					t.Fatalf("tier %q", *a.Tier)
				}
				if a.ImageURL == nil || *a.ImageURL == "" {
					t.Fatalf("missing image for %s", a.Slug)
				}
				got[a.Slug] = struct{}{}
			}
			if len(got) != len(markers) {
				t.Fatalf("selector drift for %s: html slugs %v, parsed %v", user, markers, got)
			}
			for slug := range markers {
				if _, ok := got[slug]; !ok {
					t.Fatalf("selector drift: html has %s, parsed %v", slug, got)
				}
			}
			if strings.Contains(html, ">Highlights<") || strings.Contains(html, ">ハイライト<") {
				if len(p.Highlights) == 0 {
					t.Fatal("DOM change: Highlights heading is present but no highlights were parsed")
				}
				for _, h := range p.Highlights {
					if h.Slug == "" || h.Name == "" {
						t.Fatalf("incomplete highlight %#v", h)
					}
				}
			}
		})
	}
}

var (
	liveHoverRE = regexp.MustCompile(`data-hovercard-url="/users/[^"]+/achievements/([a-z0-9-]+)/detail`)
	liveCardRE  = regexp.MustCompile(`data-achievement-slug="([a-z0-9-]+)"`)
)

func liveAchievementSlugs(html string) map[string]struct{} {
	out := map[string]struct{}{}
	for _, re := range []*regexp.Regexp{liveHoverRE, liveCardRE} {
		for _, m := range re.FindAllStringSubmatch(html, -1) {
			out[m[1]] = struct{}{}
		}
	}
	return out
}
