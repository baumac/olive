# PocketBase production review of 2026-11-14

This file is not a decision. It compares `main` at a222d83 with the page "Going to production" of the PocketBase docs.
A suggestion that is accepted needs a decision or a backlog item.

Source: https://pocketbase.io/docs/going-to-production/

Method: read the page, the migrations, `app/main.go`, `deploy/`, and the Production section of `AGENTS.md`.
The page shows only the headings of its recommendations until a person opens them. The column "What it asks" comes from the reviewer's knowledge of PocketBase.
Did not start a server. Did not test a setting in the dashboard.

## Recommendations of the page

| Recommendation | Rating of the page | What it asks | This repository | Status |
| --- | --- | --- | --- | --- |
| Enable rate limiter | highly recommended | Turn on the rate limiter of PocketBase. | The migration `1794182400_enable_rate_limits.go` turns it on. The app adds its own limits for sign up, sign in, photo upload, and place search. | Done |
| Restrict superusers to specific IPs | highly recommended | Limit who can open the dashboard. | `AGENTS.md` and `deploy/README.md` tell the operator to set the allow-list. Nothing sets it. | Done as a manual step |
| Use SMTP mail server | highly recommended | Set an SMTP server. The default is `sendmail`. | Not set and not in the docs. The Public API sends no email. The password reset and the sign in alerts of superusers need email. The Alpine image has no `sendmail`. | Missing |
| Enable MFA for superusers | optional | Ask for a one-time code at the dashboard sign in. | Not set. The code goes by email, so it needs SMTP. | Missing |
| Enable settings encryption | optional | Start the app with `--encryptionEnv` and a key of 32 characters. PocketBase then encrypts the settings in the database. | Not set. The S3 keys of the backups are in the database as plain text. Each backup ZIP file has this database. | Missing |
| Set `GOMEMLIMIT` | optional | Give the Go runtime a memory limit on a small server. | Not set. A photo upload can hold a large decoded image in memory. | Missing |
| Increase the open file descriptors limit | optional | Raise `nofile` for many open connections. | Not set in `deploy/compose.yaml`. The UI does not use the realtime connections of PocketBase, so the need is low. | Missing, low impact |

## Other sections of the page

- Reverse proxy: the Caddy example of the page has a request body limit of 10 MB and a read timeout of 360 seconds. `deploy/Caddyfile` has neither.
  The app already refuses a photo of more than 10 MB. The timeout is for realtime connections, which the UI does not use. Only the body limit is useful.
- Trusted proxy header: the page tells to set the proxy headers behind a reverse proxy. `deploy/Caddyfile` sends `X-Real-IP`, and `deploy/README.md` tells the operator to set it.
- Backup and restore: decision 00022 uses the backups of PocketBase with S3 storage. This is what the page describes.
- Docker: the page has a small example Dockerfile. `deploy/Dockerfile` builds the app of this repository and runs it as a user that is not root.

## Suggestions, highest value first

1. Turn on the settings encryption. Add `--encryptionEnv` to the start command of the image and a key variable to `deploy/.env.example`.
   The key must stay out of the backups. A lost key makes the settings of a restored backup unreadable, so the operator keeps a copy.
2. Document SMTP and superuser MFA in `deploy/README.md`. Both are dashboard settings. The dashboard is the only admin surface, so its sign in matters most.
3. Set `GOMEMLIMIT` and a memory limit for the `app` service in `deploy/compose.yaml`. Check the value against the photo memory limit of decision 00008.
4. Add `ulimits.nofile` to the `app` service.
5. Add `request_body { max_size 10MB }` to `deploy/Caddyfile`.

Suggestions 1, 3, 4, and 5 change only files in `deploy/`. Suggestion 2 changes only the README.
