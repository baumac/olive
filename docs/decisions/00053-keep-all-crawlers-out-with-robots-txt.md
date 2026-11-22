---
description: robots.txt disallows all paths for all crawlers. The pages have no canonical link. Link preview bots that obey robots.txt show no preview.
status: implemented
---

# Motivation
[motivation]: #motivation

Decision 00049 removes the sitemap. The app does not need search engine optimization.
The robots.txt of decision 00049 still lets crawlers read each restaurant, recipe, review, reviewer, and list page.
Decision 00013 also gives each of these pages a canonical link.

Search engines can then show the names and the reviews of friends and family.
Decision 00033 makes all lists public. The reviewer pages show the names of users.
AI crawlers can also collect the text of the reviews.

Decision 00000 wants to share with "close friends, family, and fellow foodies".
A friend who gets a shared link must read the page without an account.
The pages do not need to be in search results.

Expected outcome:
- crawlers that obey robots.txt read no page and no photo of the site
- the pages stay open without an account
- the pages have no canonical link
- the link previews of bots that do not read robots.txt do not change

# Scope
[scope]: #scope

In scope:
- the text of `/robots.txt`
- the removal of the canonical link of decision 00013
- the tests and the decisions that this decision changes

Out of scope:
- an `X-Robots-Tag: noindex` header or a `robots` meta tag.
  A crawler that obeys `Disallow: /` does not get the page. It then does not see a noindex.
- an `Allow` group for the link preview bots that obey robots.txt. A later decision can add it.
- blocks of crawlers that do not obey robots.txt. They need firewall rules.
- the removal of indexed pages. The app has no deployment yet.

# Guide-level explanation
[guide-level-explanation]: #guide-level-explanation

`/robots.txt` has one group:

```
User-agent: *
Disallow: /
```

The rule covers the UI, the photo files under `/v1`, the dashboard, and the PocketBase API.
It also covers AI crawlers that obey robots.txt, such as GPTBot, ClaudeBot, and CCBot.
They need no group of their own.

The pages keep the title, the description, and the link preview tags of decision 00013.
`og:url` stays. Only the `<link rel="canonical">` goes.

## Link previews

A link preview bot that obeys robots.txt does not get the page or the photo.
The chat app then shows the plain link.

| Bot | Reads robots.txt | Source |
| --- | --- | --- |
| Twitterbot of X | Yes | The X card troubleshooting guide. It follows the robots.txt specification of Google. |
| LinkedInBot | Yes | The LinkedIn Post Inspector reports a URL that robots.txt blocks. |
| Slackbot-LinkExpanding | No | `https://api.slack.com/robots` |
| facebookexternalhit, Meta-ExternalFetcher | Not always | The Meta crawler documentation. A fetch that a user asks for can bypass robots.txt. |
| Discordbot, TelegramBot, WhatsApp, iMessage, Signal | Not known | No official documentation |

The friends and family share the links in chat apps. X and LinkedIn are not the usual place.

## Search results

Google can still show a URL without a title or text when a public page links to it.
The links of the app go to people in chat apps. Few public pages link to them.

# Implementation Details
[implementation-details]: #implementation-details

- `robotsTxt` in `app/api/internal/webui/robots.go` has the two lines. The route keeps `Cache-Control: public, max-age=3600`.
- `metaTemplate` in `pagemeta.go` has no canonical link. `og:url` keeps the URL of the Application URL.
- the warning of a private Application URL says that pages get no `og:url`
- `TestRobotsTxt` compares the body with the two lines
- `TestPageMeta` checks that no page has a canonical link
- `tests/e2e/link-previews.spec.ts` checks that the restaurant and recipe pages have no canonical link
- decisions 00013 and 00049 have a note about this decision

# Unresolved questions
[unresolved-questions]: #unresolved-questions

None.
