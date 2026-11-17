package honorscope_test

import (
	"fmt"
	"strings"

	"github.com/Saku0512/honorscope"
)

func ExampleParse() {
	const page = `
	<meta property="profile:username" content="octocat" />
	<a href="/octocat?achievement=pull-shark">
	  <img class="achievement-badge-sidebar" alt="Achievement: Pull Shark" src="https://github.githubassets.com/assets/pull-shark.png" />
	  <span class="achievement-tier-label">x2</span>
	</a>
	<h2>Highlights</h2>
	<ul><li><span class="text-uppercase">Pro</span></li></ul>`

	profile, err := honorscope.Parse("https://github.com/octocat", strings.NewReader(page))
	if err != nil {
		panic(err)
	}
	fmt.Println(profile.User)
	fmt.Printf("%s %s\n", profile.Achievements[0].Slug, *profile.Achievements[0].Tier)
	fmt.Printf("%s %s\n", profile.Highlights[0].Slug, profile.Highlights[0].Name)
	fmt.Println(profile.Source)
	// Output:
	// octocat
	// pull-shark x2
	// pro PRO
	// parsed from public GitHub profile HTML
}
