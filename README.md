# honorscope

honorscope extracts a GitHub user's public **Achievements** and **Highlights** as JSON.

GitHub has no official API for profile Achievements. The badges on a profile (Pull Shark, Starstruck, Galaxy Brain, Pair Extraordinaire, Quickdraw, YOLO, Public Sponsor, Mars 2020 Helicopter Contributor, Arctic Code Vault Contributor, and others) are public-preview UI, and a user can hide them. honorscope reads the public profile HTML once and returns only what that page shows.

It is an unofficial parser for projects such as [GitHub-Contribution-Visualizer](https://github.com/Saku0512/GitHub-Contribution-Visualizer) ([issue #89](https://github.com/Saku0512/GitHub-Contribution-Visualizer/issues/89)). It is not affiliated with GitHub, and it does not ship GitHub logos, the Octocat, or badge images.

## Install

```bash
go install github.com/Saku0512/honorscope/cmd/honorscope@latest
```

Or build from this repository:

```bash
go build -o honorscope ./cmd/honorscope
```

The library module is `github.com/Saku0512/honorscope`.

## CLI

```text
honorscope <user> [--json] [--pretty]
```

The default is a short text report. `--json` writes compact JSON. `--pretty` writes indented JSON. Flags may come before or after the login.

```bash
honorscope Saku0512 --pretty
honorscope --json torvalds
```

Other flags: `--timeout` (default 20s), `--user-agent`, `--version`.

Exit codes: `0` success, `1` other error, `2` user not found, `3` rate limited.

## Library

```go
profile, err := honorscope.Fetch(ctx, "Saku0512", nil)
if err != nil {
    if errors.Is(err, honorscope.ErrUserNotFound) {
        // HTTP 404
    }
    if errors.Is(err, honorscope.ErrRateLimited) {
        // HTTP 429, or 403 with Retry-After, after backoff
    }
}
```

`Fetch` performs one `GET https://github.com/{user}`. Pass `*honorscope.Options` to set the User-Agent, timeout, base URL, or retry budget. `Parse` reads HTML you already have (for tests or a cache) and does not set `fetchedAt`.

Hidden achievements are not requested and cannot be recovered. The client sends no cookies and strips `Cookie` and `Authorization` headers.

## Output

`tier` is the multiplier printed on the badge (`x2`, `x3`, `x4`, …) or `null` when GitHub shows no multiplier. `imageURL` and `unlockedAt` are omitted when the page does not include them. Unlock times live in a separate detail fragment; honorscope does not request those.

Example from the public profile `Saku0512` on 2026-11-17:

```json
{
  "user": "Saku0512",
  "fetchedAt": "2026-11-17T04:54:36Z",
  "achievements": [
    {
      "slug": "pair-extraordinaire",
      "name": "Pair Extraordinaire",
      "tier": null,
      "imageURL": "https://github.githubassets.com/assets/pair-extraordinaire-default-579438a20e01.png"
    },
    {
      "slug": "yolo",
      "name": "YOLO",
      "tier": null,
      "imageURL": "https://github.githubassets.com/assets/yolo-default-be0bbff04951.png"
    },
    {
      "slug": "quickdraw",
      "name": "Quickdraw",
      "tier": null,
      "imageURL": "https://github.githubassets.com/assets/quickdraw-default-39c6aec8ff89.png"
    },
    {
      "slug": "pull-shark",
      "name": "Pull Shark",
      "tier": "x3",
      "imageURL": "https://github.githubassets.com/assets/pull-shark-silver-0643f87ac9fd.png"
    }
  ],
  "highlights": [
    {
      "slug": "security-advisory-credits",
      "name": "8 security advisory credits"
    }
  ],
  "source": "parsed from public GitHub profile HTML"
}
```

Highlights seen on public profiles include `pro` (the sidebar label is styled as PRO), `github-star`, `developer-program-member`, `security-bug-bounty-hunter`, `github-campus-expert`, `github-sponsors`, and `security-advisory-credits`. A leading count stays in the name (`8 security advisory credits`) and is left out of the slug. Tiers are not official thresholds; they are only the label on the badge.

`source` is always `parsed from public GitHub profile HTML`.

## How the HTML is chosen

GitHub's [robots.txt](https://github.com/robots.txt) allows a normal profile URL and explicitly allows `/*?tab=achievements&achievement=*`, but `User-agent: *` disallows `/*?tab=*`. honorscope therefore does **not** request `?tab=achievements`.

The profile sidebar already lists each public achievement (slug, name, tier, image) and the Highlights list. The achievements tab repeats the same badges as cards (`details.js-achievement-card-details`). `Parse` understands both shapes. Responsive desktop and mobile copies of the sidebar are deduplicated by slug.

Selectors live in `selectors.go`.

## Limitations

- Unofficial. GitHub can change the HTML, the badge set, or the tiers at any time. Achievements are documented as a public preview.
- A hidden achievement is simply missing. honorscope will not estimate one from commits, stars, or other API data, and it will not mark a result as "GitHub-awarded" beyond what the public page shows.
- `unlockedAt` is usually empty. Fetching each badge's detail fragment would be a crawl, so it is not done.
- Not affiliated with GitHub. Do not copy GitHub logos or the Octocat. This repository stores HTML excerpts for tests, not badge image files. `imageURL` points at GitHub's asset host.
- The client asks for English HTML (`Accept-Language: en`). The Highlights heading is also recognized as `ハイライト`.
- Names and tiers are whatever the page shows. There is no official tier API.

## Responsible use

Use this on public profiles, infrequently, and cache the JSON. The default User-Agent identifies honorscope and this repository. One call is one request unless GitHub returns HTTP 429 (or 403 with `Retry-After`), in which case the client backs off and retries a couple of times. A `Retry-After` longer than 10 seconds is returned as `ErrRateLimited` instead of sleeping. Do not send cookies or tokens. Do not walk a list of users from this tool in a tight loop.

## Tests

```bash
go test ./...
go vet ./...
HONORSCOPE_LIVE=1 go test -race -run TestLivePublicProfiles ./...
```

Unit tests use saved excerpts of public profiles (`Saku0512`, `torvalds`, `kentcdodds`) under `testdata/`. `HONORSCOPE_LIVE=1` fetches `Saku0512` and `torvalds` and fails if the achievement markers in the HTML no longer match the parser. [`.github/workflows/live.yml`](.github/workflows/live.yml) runs that check every Monday. [`.github/workflows/ci.yml`](.github/workflows/ci.yml) runs `go vet`, `go test -race`, and golangci-lint.

## 日本語

honorscope は、GitHub ユーザーの公開プロフィールから Achievements（Pull Shark など）と Highlights（PRO、GitHub Star、Developer Program Member など）を読み、JSON にする非公式ツールです。公式 API はありません。

取得は `https://github.com/{user}` を 1 回だけです。`?tab=achievements` は GitHub の robots.txt が一般クローラに対して拒否しているため、リクエストしません。公開サイドバーに同じバッジ（slug、名前、x2 などのティア、画像 URL）と Highlights が出ます。非表示にした Achievement は HTML に含まれないので、結果にも出ません。詳細ダイアログを追加で取りに行かないため、`unlockedAt` は通常ありません。

```bash
go install github.com/Saku0512/honorscope/cmd/honorscope@latest
honorscope Saku0512 --pretty
```

GitHub とは無関係です。ロゴや Octocat、バッジ画像は同梱しません。短い間隔での大量取得は避け、結果はキャッシュしてください。HTML は変わることがあり、毎週の live test でセレクタの破損に気づくようにしています。

## License

[MIT](LICENSE). Copyright (c) 2026 Saku0512.
