package honorscope

import (
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// Selectors for the public profile DOM observed on 2026-11-17.
//
// Fetch downloads https://github.com/{user} (no query string). That document's
// sidebar lists every public achievement twice, once for desktop
// (d-none d-md-block) and once for mobile (d-md-none d-block):
//
//	img.achievement-badge-sidebar
//	  alt="Achievement: {Name}"
//	  data-hovercard-url="/users/{user}/achievements/{slug}/detail?hovercard=1"
//	parent a[href*="achievement={slug}"]
//	span.achievement-tier-label  text x2 / x3 / x4 when GitHub shows a multiplier
//
// Highlights follow an h2 ("Highlights", or "ハイライト" if the page is localized)
// and a ul.list-style-none of li items. PRO is a span.text-uppercase.
//
// The achievements tab (details.js-achievement-card-details[data-achievement-slug])
// repeats the same badges with an h3 name. Fetch does not request ?tab= because
// GitHub's robots.txt disallows /*?tab=* for User-agent: *. Parse still reads
// those cards when they are present in saved HTML, and prefers them over the
// sidebar copy of the same slug.
const (
	selAchievementCard  = `details.js-achievement-card-details[data-achievement-slug]`
	selCardName         = `summary h3`
	selCardTier         = `summary .achievement-tier-label`
	selCardImage        = `summary img.achievement-badge-card`
	selCardUnlocked     = `summary [datetime]`
	selSidebarBadge     = `img.achievement-badge-sidebar`
	selTierLabel        = `.achievement-tier-label`
	selProfileUsername  = `meta[property="profile:username"]`
	selUppercase        = `.text-uppercase`
	attrAchievementSlug = `data-achievement-slug`
)

var (
	tierRE         = regexp.MustCompile(`(?i)^x[0-9]+$`)
	hoverSlugRE    = regexp.MustCompile(`/achievements/([a-z0-9-]+)`)
	leadingCountRE = regexp.MustCompile(`^[0-9]+\s+`)
)

func profileUsername(doc *goquery.Document, pageURL string) string {
	if doc != nil {
		if v, ok := doc.Find(selProfileUsername).Attr("content"); ok {
			v = strings.TrimSpace(v)
			if validUsername(v) {
				return v
			}
		}
	}
	u, err := url.Parse(pageURL)
	if err != nil {
		return ""
	}
	seg := strings.Trim(u.Path, "/")
	if i := strings.IndexByte(seg, '/'); i >= 0 {
		seg = seg[:i]
	}
	if validUsername(seg) {
		return seg
	}
	return ""
}

func extractAchievements(doc *goquery.Document, pageURL string) []Achievement {
	out := []Achievement{}
	index := map[string]int{}
	add := func(a Achievement) {
		if a.Slug == "" {
			return
		}
		if i, ok := index[a.Slug]; ok {
			// Keep fields already filled. Cards are added before the sidebar,
			// so a heading or card image wins and a sidebar tier can fill in.
			out[i] = mergeAchievement(out[i], a)
			return
		}
		index[a.Slug] = len(out)
		out = append(out, a)
	}

	doc.Find(selAchievementCard).Each(func(_ int, card *goquery.Selection) {
		add(cardAchievement(card, pageURL))
	})
	doc.Find(selSidebarBadge).Each(func(_ int, img *goquery.Selection) {
		add(sidebarAchievement(img, pageURL))
	})
	return out
}

func cardAchievement(card *goquery.Selection, pageURL string) Achievement {
	slug, _ := card.Attr(attrAchievementSlug)
	name := cleanText(card.Find(selCardName).First().Text())
	if name == "" {
		alt, _ := card.Find(selCardImage).First().Attr("alt")
		name = nameFromAlt(alt)
	}
	src, _ := card.Find(selCardImage).First().Attr("src")
	var unlocked *time.Time
	if dt, ok := card.Find(selCardUnlocked).First().Attr("datetime"); ok {
		unlocked = parseTimestamp(dt)
	}
	return Achievement{
		Slug:       strings.TrimSpace(slug),
		Name:       name,
		Tier:       parseTier(card.Find(selCardTier).First().Text()),
		ImageURL:   strPtr(resolveRef(pageURL, src)),
		UnlockedAt: unlocked,
	}
}

func sidebarAchievement(img *goquery.Selection, pageURL string) Achievement {
	alt, _ := img.Attr("alt")
	src, _ := img.Attr("src")
	anchor := img.ParentsFiltered("a").First()
	if anchor.Length() == 0 {
		anchor = img.Parent()
	}
	href, _ := anchor.Attr("href")
	slug := slugFromRef(href)
	if slug == "" {
		hover, _ := img.Attr("data-hovercard-url")
		slug = slugFromRef(hover)
	}
	return Achievement{
		Slug:     slug,
		Name:     nameFromAlt(alt),
		Tier:     parseTier(anchor.Find(selTierLabel).First().Text()),
		ImageURL: strPtr(resolveRef(pageURL, src)),
	}
}

func mergeAchievement(primary, extra Achievement) Achievement {
	if primary.Name == "" {
		primary.Name = extra.Name
	}
	if primary.Tier == nil {
		primary.Tier = extra.Tier
	}
	if primary.ImageURL == nil {
		primary.ImageURL = extra.ImageURL
	}
	if primary.UnlockedAt == nil {
		primary.UnlockedAt = extra.UnlockedAt
	}
	return primary
}

func extractHighlights(doc *goquery.Document) []Highlight {
	out := []Highlight{}
	seen := map[string]struct{}{}
	doc.Find("h2").Each(func(_ int, h *goquery.Selection) {
		if !isHighlightHeading(h.Text()) {
			return
		}
		ul := h.NextAllFiltered("ul").First()
		if ul.Length() == 0 {
			ul = h.Parent().ChildrenFiltered("ul").First()
		}
		ul.ChildrenFiltered("li").Each(func(_ int, li *goquery.Selection) {
			name := highlightName(li)
			slug := highlightSlug(name)
			if name == "" || slug == "" {
				return
			}
			if _, ok := seen[slug]; ok {
				return
			}
			seen[slug] = struct{}{}
			out = append(out, Highlight{Slug: slug, Name: name})
		})
	})
	return out
}

func isHighlightHeading(text string) bool {
	switch strings.ToLower(cleanText(text)) {
	case "highlights", "ハイライト":
		return true
	default:
		return false
	}
}

func highlightName(li *goquery.Selection) string {
	clone := li.Clone()
	clone.Find("svg, script, style").Remove()
	clone.Find(selUppercase).Each(func(_ int, s *goquery.Selection) {
		s.SetText(strings.ToUpper(strings.TrimSpace(s.Text())))
	})
	return cleanText(clone.Text())
}

func highlightSlug(name string) string {
	n := strings.ToLower(strings.TrimSpace(name))
	n = leadingCountRE.ReplaceAllString(n, "")
	return slugify(n)
}

func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	prevHyphen := false
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z' || r >= '0' && r <= '9':
			b.WriteRune(r)
			prevHyphen = false
		case r == ' ' || r == '_' || r == '-' || r == '/':
			if b.Len() > 0 && !prevHyphen {
				b.WriteByte('-')
				prevHyphen = true
			}
		}
	}
	return strings.Trim(b.String(), "-")
}

func slugFromRef(ref string) string {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return ""
	}
	u, err := url.Parse(ref)
	if err == nil {
		if s := u.Query().Get("achievement"); s != "" {
			return s
		}
		ref = u.Path
	}
	m := hoverSlugRE.FindStringSubmatch(ref)
	if len(m) == 2 {
		return m[1]
	}
	return ""
}

func nameFromAlt(alt string) string {
	alt = strings.TrimSpace(alt)
	if i := strings.Index(alt, ":"); i >= 0 {
		if rest := strings.TrimSpace(alt[i+1:]); rest != "" {
			return rest
		}
	}
	return alt
}

func parseTier(s string) *string {
	s = strings.ToLower(cleanText(s))
	if !tierRE.MatchString(s) {
		return nil
	}
	return &s
}

func parseTimestamp(s string) *time.Time {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	for _, layout := range []string{time.RFC3339, "2006-01-02T15:04:05.000Z07:00", "2006-01-02"} {
		if t, err := time.Parse(layout, s); err == nil {
			u := t.UTC()
			return &u
		}
	}
	return nil
}

func resolveRef(pageURL, ref string) string {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return ""
	}
	u, err := url.Parse(ref)
	if err != nil {
		return ref
	}
	if u.IsAbs() {
		return u.String()
	}
	base, err := url.Parse(pageURL)
	if err != nil || base.Scheme == "" || base.Host == "" {
		return ref
	}
	return base.ResolveReference(u).String()
}

func cleanText(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
