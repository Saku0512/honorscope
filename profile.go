package honorscope

import "time"

// SourcePublicHTML is the Profile.Source value. It records that the result
// was parsed from public profile HTML rather than an official API.
const SourcePublicHTML = "parsed from public GitHub profile HTML"

// Version is the module version. Release builds override it with ldflags.
var Version = "0.1.0"

// Profile is the public Achievements and Highlights visible on a GitHub
// profile. Slices are empty, not null, when the user has none.
type Profile struct {
	User         string        `json:"user"`
	FetchedAt    time.Time     `json:"fetchedAt,omitempty"`
	Achievements []Achievement `json:"achievements"`
	Highlights   []Highlight   `json:"highlights"`
	Source       string        `json:"source"`
}

// Achievement is one public profile achievement. Tier is the multiplier
// GitHub prints on the badge (x2, x3, x4, …) or nil when the badge has no
// multiplier. ImageURL and UnlockedAt are set only when the HTML includes
// them. Detail dialogs that would carry an unlock time are loaded separately
// by GitHub, and this package does not request them.
type Achievement struct {
	Slug       string     `json:"slug"`
	Name       string     `json:"name"`
	Tier       *string    `json:"tier"`
	ImageURL   *string    `json:"imageURL,omitempty"`
	UnlockedAt *time.Time `json:"unlockedAt,omitempty"`
}

// Highlight is one badge in the profile sidebar Highlights list, such as
// PRO, GitHub Star, Developer Program Member, or security advisory credits.
type Highlight struct {
	Slug string `json:"slug"`
	Name string `json:"name"`
}
