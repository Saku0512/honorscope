package honorscope

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func parseFixture(t *testing.T, name, pageURL string) *Profile {
	t.Helper()
	p, err := Parse(pageURL, bytes.NewReader(readFixture(t, name)))
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestParseSaku0512ProfileSidebar(t *testing.T) {
	p := parseFixture(t, "saku0512-profile.html", "https://github.com/Saku0512")
	if p.User != "Saku0512" {
		t.Fatalf("user %q", p.User)
	}
	if p.Source != SourcePublicHTML {
		t.Fatalf("source %q", p.Source)
	}
	if !p.FetchedAt.IsZero() {
		t.Fatalf("Parse should leave fetchedAt zero, got %s", p.FetchedAt)
	}
	want := []struct {
		slug, name, tier, image string
	}{
		{"pair-extraordinaire", "Pair Extraordinaire", "", "pair-extraordinaire-default-579438a20e01.png"},
		{"yolo", "YOLO", "", "yolo-default-be0bbff04951.png"},
		{"quickdraw", "Quickdraw", "", "quickdraw-default-39c6aec8ff89.png"},
		{"pull-shark", "Pull Shark", "x3", "pull-shark-silver-0643f87ac9fd.png"},
	}
	if len(p.Achievements) != len(want) {
		t.Fatalf("achievements %d, want %d (responsive copies must be deduped)", len(p.Achievements), len(want))
	}
	for i, w := range want {
		a := p.Achievements[i]
		if a.Slug != w.slug || a.Name != w.name {
			t.Fatalf("[%d] got %s %q", i, a.Slug, a.Name)
		}
		if w.tier == "" {
			if a.Tier != nil {
				t.Fatalf("%s tier %q, want null", a.Slug, *a.Tier)
			}
		} else if a.Tier == nil || *a.Tier != w.tier {
			t.Fatalf("%s tier %#v, want %s", a.Slug, a.Tier, w.tier)
		}
		if a.ImageURL == nil || !strings.Contains(*a.ImageURL, w.image) {
			t.Fatalf("%s image %#v", a.Slug, a.ImageURL)
		}
		if a.UnlockedAt != nil {
			t.Fatalf("%s unlockedAt set from a page that does not include it", a.Slug)
		}
	}
	if len(p.Highlights) != 1 {
		t.Fatalf("highlights %#v", p.Highlights)
	}
	h := p.Highlights[0]
	if h.Slug != "security-advisory-credits" || h.Name != "8 security advisory credits" {
		t.Fatalf("highlight %#v", h)
	}

	raw, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(raw, []byte(`"tier":null`)) {
		t.Fatalf("missing null tier: %s", raw)
	}
	if !bytes.Contains(raw, []byte(`"tier":"x3"`)) {
		t.Fatalf("missing x3: %s", raw)
	}
	if bytes.Contains(raw, []byte(`"unlockedAt"`)) {
		t.Fatalf("unlockedAt should be omitted: %s", raw)
	}
	if bytes.Contains(raw, []byte(`"starstruck"`)) {
		t.Fatal("achievement absent from the HTML was returned")
	}
	for _, needle := range []string{`"achievements":[]`, `"highlights":[]`} {
		if bytes.Contains(raw, []byte(needle)) {
			t.Fatalf("unexpected empty list encoding %s in %s", needle, raw)
		}
	}
}

func TestParseTorvaldsCardsAndSidebar(t *testing.T) {
	p := parseFixture(t, "torvalds-achievements.html", "https://github.com/torvalds?tab=achievements")
	if p.User != "torvalds" {
		t.Fatalf("user %q", p.User)
	}
	want := []struct{ slug, name, tier string }{
		{"pair-extraordinaire", "Pair Extraordinaire", "x4"},
		{"mars-2020-contributor", "Mars 2020 Contributor", ""},
		{"arctic-code-vault-contributor", "Arctic Code Vault Contributor", ""},
		{"pull-shark", "Pull Shark", "x2"},
		{"starstruck", "Starstruck", "x4"},
	}
	if len(p.Achievements) != len(want) {
		t.Fatalf("got %d achievements, want %d", len(p.Achievements), len(want))
	}
	for i, w := range want {
		a := p.Achievements[i]
		if a.Slug != w.slug || a.Name != w.name {
			t.Fatalf("[%d] %s %q", i, a.Slug, a.Name)
		}
		if w.tier == "" && a.Tier != nil {
			t.Fatalf("%s tier %q", a.Slug, *a.Tier)
		}
		if w.tier != "" && (a.Tier == nil || *a.Tier != w.tier) {
			t.Fatalf("%s tier %#v", a.Slug, a.Tier)
		}
		if a.ImageURL == nil || !strings.HasPrefix(*a.ImageURL, "https://github.githubassets.com/assets/") {
			t.Fatalf("%s image %#v", a.Slug, a.ImageURL)
		}
	}
	if len(p.Highlights) != 0 {
		t.Fatalf("highlights %#v", p.Highlights)
	}
	raw, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(raw, []byte(`"highlights":null`)) || bytes.Contains(raw, []byte(`"achievements":null`)) {
		t.Fatalf("null slice: %s", raw)
	}
}

func TestParseKentcdoddsHighlights(t *testing.T) {
	p := parseFixture(t, "kentcdodds-highlights.html", "https://github.com/kentcdodds")
	if p.User != "kentcdodds" {
		t.Fatalf("user %q", p.User)
	}
	if len(p.Highlights) != 2 {
		t.Fatalf("highlights %#v", p.Highlights)
	}
	if p.Highlights[0].Slug != "github-star" || p.Highlights[0].Name != "GitHub Star" {
		t.Fatalf("star %#v", p.Highlights[0])
	}
	if p.Highlights[1].Slug != "pro" || p.Highlights[1].Name != "PRO" {
		t.Fatalf("pro %#v", p.Highlights[1])
	}
	if strings.Contains(p.Highlights[0].Name, "M8") {
		t.Fatal("svg path leaked into the highlight name")
	}
}

func TestParseNoAchievements(t *testing.T) {
	p := parseFixture(t, "no-achievements.html", "https://github.com/nobody")
	if p.User != "nobody" {
		t.Fatalf("user %q", p.User)
	}
	if len(p.Achievements) != 0 || len(p.Highlights) != 0 {
		t.Fatalf("got %+v", p)
	}
	raw, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(raw, []byte(`"achievements":[]`)) || !bytes.Contains(raw, []byte(`"highlights":[]`)) {
		t.Fatalf("empty slices must encode as arrays: %s", raw)
	}
}

func TestParseUnlockedAtAndRelativeImage(t *testing.T) {
	p := parseFixture(t, "unlocked-at.html", "https://github.com/ada")
	if len(p.Achievements) != 1 {
		t.Fatalf("%+v", p.Achievements)
	}
	a := p.Achievements[0]
	if a.Slug != "quickdraw" || a.Name != "Quickdraw" || a.Tier != nil {
		t.Fatalf("%+v", a)
	}
	want := time.Date(2024, 5, 1, 12, 30, 0, 0, time.UTC)
	if a.UnlockedAt == nil || !a.UnlockedAt.Equal(want) {
		t.Fatalf("unlocked %#v", a.UnlockedAt)
	}
	if a.ImageURL == nil || *a.ImageURL != "https://github.com/assets/quickdraw-default.png" {
		t.Fatalf("image %#v", a.ImageURL)
	}
}

func TestParseHighlightVariants(t *testing.T) {
	p := parseFixture(t, "highlight-variants.html", "https://github.com/example")
	want := []Highlight{
		{Slug: "developer-program-member", Name: "Developer Program Member"},
		{Slug: "security-bug-bounty-hunter", Name: "Security Bug Bounty Hunter"},
		{Slug: "github-campus-expert", Name: "GitHub Campus Expert"},
		{Slug: "github-sponsors", Name: "GitHub Sponsors"},
	}
	if len(p.Highlights) != len(want) {
		t.Fatalf("got %#v", p.Highlights)
	}
	for i, w := range want {
		if p.Highlights[i] != w {
			t.Fatalf("[%d] got %#v want %#v", i, p.Highlights[i], w)
		}
	}
}

func TestParseMergesSidebarTierIntoCard(t *testing.T) {
	const html = `
	<meta property="profile:username" content="ada" />
	<details class="js-achievement-card-details" data-achievement-slug="pull-shark">
	  <summary>
	    <img class="achievement-badge-card" alt="Achievement: Pull Shark" src="https://example.com/card.png" />
	    <h3>Pull Shark</h3>
	  </summary>
	</details>
	<a href="/ada?achievement=pull-shark&amp;tab=achievements">
	  <img class="achievement-badge-sidebar" alt="Achievement: Pull Shark" src="https://example.com/side.png" />
	  <span class="achievement-tier-label">X2</span>
	</a>`
	p, err := Parse("https://github.com/ada", strings.NewReader(html))
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Achievements) != 1 {
		t.Fatalf("%+v", p.Achievements)
	}
	a := p.Achievements[0]
	if a.Name != "Pull Shark" || a.Tier == nil || *a.Tier != "x2" {
		t.Fatalf("%+v", a)
	}
	if a.ImageURL == nil || *a.ImageURL != "https://example.com/card.png" {
		t.Fatalf("image %#v", a.ImageURL)
	}
}

func TestParseJapaneseHighlightHeading(t *testing.T) {
	const html = `<h2>ハイライト</h2><ul><li>GitHub Campus Expert</li></ul>`
	p, err := Parse("https://github.com/ada", strings.NewReader(html))
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Highlights) != 1 || p.Highlights[0].Slug != "github-campus-expert" {
		t.Fatalf("%#v", p.Highlights)
	}
}

func TestParseTierWordsAreNotMultipliers(t *testing.T) {
	const html = `<a href="/ada?achievement=yolo"><img class="achievement-badge-sidebar" alt="Achievement: YOLO" src="https://example.com/y.png" /><span class="achievement-tier-label">gold</span></a>`
	p, err := Parse("https://github.com/ada", strings.NewReader(html))
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Achievements) != 1 || p.Achievements[0].Tier != nil {
		t.Fatalf("%+v", p.Achievements)
	}
}

func TestParseNilReader(t *testing.T) {
	if _, err := Parse("https://github.com/ada", nil); err == nil {
		t.Fatal("expected error")
	}
}

func TestFixturesAreNotBadgeImages(t *testing.T) {
	entries, err := os.ReadDir("testdata")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		switch strings.ToLower(filepath.Ext(e.Name())) {
		case ".png", ".jpg", ".jpeg", ".gif", ".svg", ".webp", ".ico":
			t.Fatalf("do not commit GitHub badge images: %s", e.Name())
		}
	}
}

func TestValidUsername(t *testing.T) {
	ok := []string{"a", "Saku0512", "torvalds", "user-name", strings.Repeat("a", 39)}
	bad := []string{"", " ", "-abc", "abc-", "a--b", "user name", "user/name", "../etc", "a@b", "user?x=1", strings.Repeat("a", 40)}
	for _, name := range ok {
		if !validUsername(name) {
			t.Errorf("valid %q rejected", name)
		}
	}
	for _, name := range bad {
		if validUsername(name) {
			t.Errorf("invalid %q accepted", name)
		}
	}
}
