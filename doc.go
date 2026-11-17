// Package honorscope reads a GitHub user's public Achievements and Highlights
// from profile HTML.
//
// GitHub does not provide an official API for profile Achievements. Fetch
// loads https://github.com/{user} once (the profile sidebar already lists
// public badges and highlights) and Parse turns that HTML into a Profile.
// Achievements the user has hidden are absent from the public page, so they
// are absent from the result as well.
//
// The client does not send cookies or credentials. Callers should cache
// results and avoid repeat lookups; GitHub rate-limits unauthenticated
// traffic, and robots.txt disallows ?tab= URLs for generic clients.
package honorscope
