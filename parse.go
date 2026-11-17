package honorscope

import (
	"fmt"
	"io"

	"github.com/PuerkitoBio/goquery"
)

// Parse reads a public profile document and returns the achievements and
// highlights it contains. pageURL is the document location, used to resolve
// relative image URLs and, if the username meta tag is missing, the login.
// FetchedAt is left zero; Fetch sets it.
func Parse(pageURL string, r io.Reader) (*Profile, error) {
	if r == nil {
		return nil, fmt.Errorf("honorscope: nil html")
	}
	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		return nil, fmt.Errorf("honorscope: parse html: %w", err)
	}
	p := &Profile{
		User:         profileUsername(doc, pageURL),
		Achievements: extractAchievements(doc, pageURL),
		Highlights:   extractHighlights(doc),
		Source:       SourcePublicHTML,
	}
	if p.Achievements == nil {
		p.Achievements = []Achievement{}
	}
	if p.Highlights == nil {
		p.Highlights = []Highlight{}
	}
	return p, nil
}
