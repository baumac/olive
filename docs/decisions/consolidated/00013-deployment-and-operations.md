---
description: The app runs as one instance on the home server with Docker Compose and Caddy for TLS. PocketBase saves the backups of pb_data on the local file system, and the backup task of the host copies them off the server. The PocketBase settings are encrypted, and the operator sets the mail server, the approval, and the admin access in the dashboard.
status: implemented
---

# Motivation
[motivation]: #motivation

Decision 00000 hosts the app on our home server at reviews.sponga47.com.
The operator needs one way to build, start, update, back up, and restore the app.
`pb_data` has the database and the photo files. Nothing else keeps a copy of them.
Some state is in the memory of the process, so the app cannot run as many instances.

Expected outcome:
- one command builds the image, and one command starts the app with TLS
- a backup of the database and the photos goes to a different computer each day
- a person can restore the app from a backup with written steps
- the settings with secrets, such as the SMTP password, are encrypted in the database
- the memory of the app has a limit below the memory of the server
- CI fails when the image does not build

# Scope
[scope]: #scope

In scope:
- `deploy/Dockerfile`, `deploy/compose.yaml`, `deploy/Caddyfile`, `deploy/.env.example`, and `deploy/README.md`
- the rule of one instance
- the backup plan and the restore
- the settings encryption, the memory limits, and the body limit of the proxy
- the PocketBase settings that the operator sets
- the environment variables of the app

Out of scope:
- a push of the image to a registry, and a deploy from CI
- the backup task of the host. The operator sets it up.
- more than one instance
- monitoring and log collection
- OAuth2 sign in. The backlog has it under "Won't do".

# Guide-level explanation
[guide-level-explanation]: #guide-level-explanation

`deploy/README.md` is the guide for the operator. It has the steps to start, update, back up, and restore.

| Service | Image | Use |
|---|---|---|
| `app` | `reviews-app:local` | the app server on port 8090 with the `pb_data` volume |
| `caddy` | `caddy` | the reverse proxy on the ports 80 and 443. It gets the TLS certificate of `SITE_ADDRESS`. |

- `deploy/.env` has `SITE_ADDRESS`, `PB_ENCRYPTION_KEY`, and the `REVIEWS_` variables of the app
- `compose.yaml` names each `REVIEWS_` variable. A new variable of the app needs a line there.
- `make build-image` builds the image `reviews-app:local` from `deploy/Dockerfile`
- the app applies the migrations when it starts
- to make the first superuser, run `reviews-app superuser create EMAIL PASSWORD` with `--encryptionEnv=PB_ENCRYPTION_KEY`

## One instance

- SQLite has one writer process
- the rate limits of decision 00004 and the place cache of decision 00010 are in the memory of the process. A restart sets the limits to zero. This is accepted.
- a second instance needs a shared store for these values. That needs a new decision.

## Settings of the operator

In the PocketBase dashboard at `/_/`:
- **Backups**: turn on the auto backups. Without them, the app has no backup.
- **Application URL**: the public URL of the site. The link previews of decision 00012 start with it.
- **Trusted proxy**: the header `X-Real-IP`. Otherwise all clients have the IP address of the proxy, and the sign in limit counts them together.
- **Superuser IP allow-list**: only the admin networks can open the dashboard
- **Mail server**: set it before the first sign up. Without SMTP, the app writes each email to the log.
- **OTP and MFA** for the `_superusers` collection, after the mail server
- **Approval**: approve new users in the `users` collection. See decision 00003.

The reverse proxy must not set a `Referrer-Policy` of `no-referrer`, and must not turn off `geolocation` in a `Permissions-Policy`. See decision 00010.

## Backups

- PocketBase makes each backup as one ZIP file of all of `pb_data`, with the databases and the photo files
- a cron expression sets the time, and a number sets how many backups stay
- the backups go to `pb_data/backups` on the local file system
- the backup task of the host copies this folder to a different computer. It starts after the auto backups.
- a restore on a server that runs is one action in the dashboard. A new server gets the content of the ZIP file in its volume.
- the backups do not have `PB_ENCRYPTION_KEY`. The operator keeps a copy, and a restore needs it.

## Environment variables

| Variable | Decision |
|---|---|
| `REVIEWS_SIGN_IN_LIMIT`, `REVIEWS_SIGN_IN_FAILURE_LIMIT`, `REVIEWS_CREATE_LIMIT`, `REVIEWS_EDIT_LIMIT`, `REVIEWS_PHOTO_UPLOAD_LIMIT`, `REVIEWS_PLACE_SEARCH_LIMIT`, `REVIEWS_EMAIL_LIMIT` | 00004 |
| `REVIEWS_PLACES_URL`, `REVIEWS_MAP_TILE_URL`, `REVIEWS_MAP_ATTRIBUTION` | 00010 |
| `REVIEWS_VALIDATE_RESPONSES`, `REVIEWS_VALIDATE_REQUESTS` | 00001 |
| `PB_ENCRYPTION_KEY` | this decision |
| `APP_MEMORY_LIMIT`, `APP_GOMEMLIMIT` | this decision |

# Implementation Details
[implementation-details]: #implementation-details

## Image

- the build context is the root of the repository. `.dockerignore` keeps `node_modules`, `bin`, `pb_data`, the docs, and the tests out of it.
- each base image has an exact version tag. Renovate updates the tags. See decision 00015.
- stage `ui` uses the Node.js 22 image. It runs `npm ci` for the SDK and UI workspaces and builds both.
- stage `build` uses the Go image with the toolchain version of `app/api/go.mod`. It copies the UI build into `app/api/internal/webui/dist` and builds the app with `CGO_ENABLED=0`.
- the last stage uses Alpine with `tzdata` and `ca-certificates`. The app runs as the user `reviews` with the ID 10001.
- the image has a health check on `/api/health`

## Settings encryption

- the start command has `--encryptionEnv=PB_ENCRYPTION_KEY`. `compose.yaml` stops without the variable.
- the app does not start with a key that does not have 32 characters. The example value of `.env.example` is such a key.
- a `reviews-app` command with `docker compose exec` does not use the start command. It needs the flag, or it cannot read the settings.

## Limits

| Limit | Value | Reason |
|---|---|---|
| `mem_limit` of `app` | 768 MB | The server has 1 GB. Caddy and the system get the rest. |
| `GOMEMLIMIT` | 600 MiB | The Go runtime frees memory before Docker stops the container. The photo uploads of decision 00009 peak at about 264 MB. |
| `ulimits.nofile` | 65536 | Each connection and each SQLite file uses a file descriptor. |
| `request_body` of Caddy | 11 MB | The limit of the app is 10 MiB. A body between the two limits gets the `413` of the app with Problem Details. |

## Reverse proxy

- Caddy gets and renews the certificate without more setup. It compresses the responses with zstd and gzip.
- Caddy sets `X-Real-IP` to the address of the client
- the session cookie of decision 00003 gets `Secure` from the `X-Forwarded-Proto` header that Caddy sets

## Limits of the backup plan

- each ZIP file has all photos again. The disk needs space for `pb_data` and for `pb_data` times the number of backups.
- the app can be slower while a backup runs
- PocketBase advises the `sqlite3` and `rsync` plan for a `pb_data` of 5 GB or more. The site then needs a new decision.
- a failed backup shows only in the logs of the dashboard
- many host backups skip `/var/lib/docker`. The operator adds the folder of the volume to the task.
- a copy of the database files that the host task makes while the app runs can be broken. A restore uses a ZIP file.

## Daily jobs

- `photo-cleanup` deletes the photo files without a record. See decision 00009.
- `user-cleanup` deletes the users that wait for more than 30 days and have no data. See decision 00003.
- superusers see and start both under Settings, Crons in the dashboard

## CI and tests

- the `docker` job of CI runs `make build-image` and checks `deploy/compose.yaml` with `docker compose config`. See decision 00015.
- tested by hand: the image builds and gets the state `healthy`, a backup ZIP file restores on a new server, the key rules, the memory limits, and the two `413` limits
- not tested: the `caddy` service with a real DNS name, the copy by a real host backup task, and the steps for the mail server and the MFA

# Alternatives
[alternatives]: #alternatives

| Alternative | Result |
|---|---|
| A `backup` service with `sqlite3 .backup` and `rsync` | Removed. It needs a script, tools in the image, and a second step to get the files off the server. |
| The backups of PocketBase on S3 storage | Removed. The host backup already copies the files off the server. A bucket and an access key are one more thing to keep. |
| `DefaultEncryptionEnv` in `main.go` | Not selected. A local run with the variable in its environment would encrypt the local `pb_data`. |
| nginx with certbot | Not selected. It needs a second container and a renew job. |
| PocketBase serves TLS without a proxy | Not selected. The compression, HTTP/3, and a later second site need a proxy. |
| More than one instance | Not selected. The limits and the cache are in memory, and SQLite has one writer. |

# Unresolved questions
[unresolved-questions]: #unresolved-questions

None.
