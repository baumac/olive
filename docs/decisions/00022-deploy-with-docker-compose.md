---
description: Deploy the app as one instance with Docker Compose and Caddy for TLS. PocketBase sends the backups of pb_data to an S3 compatible storage.
status: implemented
---

# Motivation
[motivation]: #motivation

The project had no way to run the app on a server.

- no Dockerfile existed. Decision 00010 left the `docker` job of CI for this decision.
- `pb_data` has the database and the photo files. Nothing made a backup of it.
- the rate limits and the place cache are in the memory of the process. No document told what this means for a deploy.

Expected outcome:
- one command builds the image, and one command starts the app with TLS
- a backup of the database and the photos goes to a different computer each day
- a person can restore the app from a backup with written steps
- CI fails when the image does not build

# Scope
[scope]: #scope

In scope:
- `deploy/Dockerfile`, `deploy/compose.yaml`, `deploy/Caddyfile`, `deploy/.env.example`, and `deploy/README.md`
- the backup plan
- the `make build-image` target and the `docker` job of CI
- the rule of one instance

Out of scope:
- a push of the image to a registry, and a deploy from CI
- the S3 bucket and its access key. The operator makes them and sets them in the dashboard.
- more than one instance
- the document about OAuth2 sign in. It stays in the backlog.
- monitoring and log collection

# Guide-level explanation
[guide-level-explanation]: #guide-level-explanation

`deploy/README.md` is the guide for an operator. It has the steps to start, update, back up, and restore.

Docker Compose runs two services:

| Service | Image | What |
| --- | --- | --- |
| `app` | `reviews-app:local` | the app server on port 8090 with the `pb_data` volume |
| `caddy` | `caddy` | the reverse proxy on the ports 80 and 443. It gets the TLS certificate of `SITE_ADDRESS`. |

`deploy/.env` has `SITE_ADDRESS` and the `REVIEWS_` variables of the app.
`compose.yaml` names each `REVIEWS_` variable. A new variable of the app needs a line there.

## One instance

The app runs as one instance.

- SQLite has one writer process
- the sign in limits, the photo upload limit, the place search limit, and the place cache are in the memory of the process
- a restart sets the limits to zero. This is accepted.
- a second instance needs a shared store for these values. That is a new decision.

## Backups

PocketBase makes the backups. The operator turns them on in Settings > Backups of the dashboard.

- a cron expression sets the time, and a number sets how many backups stay
- the backups go to an S3 compatible storage, so they are not on the server
- each backup is one ZIP file of all of `pb_data`. It has the databases and the photo files.
- a restore on a server that runs is one action in the dashboard. A new server gets the content of the ZIP file in its volume.
- the compose file has no backup service. Without the dashboard settings, the app has no backup.

The first version of this decision had a `backup` service with `sqlite3 .backup` and `rsync` to a folder of the host.
It was removed on 2026-11-14. Its backups stayed on the server. The S3 storage of PocketBase needs no code.

# Implementation Details
[implementation-details]: #implementation-details

## Image

- the build context is the root of the repository. `.dockerignore` keeps `node_modules`, `bin`, `app/pb_data`, the docs, and the tests out of it.
- each base image has an exact version tag. Renovate updates the tags. See decision 00024.
- stage `ui` uses the Node.js 22 image. It runs `npm ci` for the SDK and UI workspaces and builds both.
- stage `build` uses the Go image with the toolchain version of `app/go.mod`. It copies the UI build into `app/internal/webui/dist` and builds the app with `CGO_ENABLED=0`.
- the last stage uses Alpine with `tzdata` and `ca-certificates`. The app runs as the user `reviews` with the ID 10001.
- the image has a health check on `/api/health`

| Option for the backups | Result |
| --- | --- |
| the backup feature of PocketBase with S3 storage | Selected. It needs only settings. The backups are on a different computer. |
| a service with `sqlite3 .backup` and `rsync` | Not selected. It copies each photo only one time, but it needs a script, tools in the image, and a second step to get the files off the server. |

Limits of the selected option:
- each ZIP file has all photos again. The bucket needs the size of `pb_data` times the number of backups.
- the app can be slower while a backup runs
- PocketBase advises the `sqlite3` and `rsync` plan for a `pb_data` of 5 GB or more. The site then needs a new decision.
- a failed backup shows only in the logs of the dashboard
- the S3 settings are in the database. The operator keeps a copy of the access key in a different place.

## Reverse proxy

| Option | Result |
| --- | --- |
| Caddy | Selected. It gets and renews the certificate without more setup. The config has six lines. |
| nginx with certbot | Not selected. It needs a second container and a renew job. |
| no proxy, PocketBase serves TLS | Not selected. The compression, the HTTP/3 support, and a later second site need a proxy. |

- Caddy sets `X-Real-IP` to the address of the client. The operator sets the trusted proxy header of PocketBase to `X-Real-IP`. See decision 00004.

## CI

- `make build-image` runs `docker build -f deploy/Dockerfile -t reviews-app:local .`, as decision 00010 tells
- the `docker` job runs it and then checks `deploy/compose.yaml` with `docker compose config`
- `CI result` needs the `docker` job

## Tests

The change was tested by hand:
- the image builds, the container gets the state `healthy`, and it serves the UI and `/v1/restaurants`
- `POST /api/backups` of PocketBase makes a ZIP file in the container
- the restore steps of `deploy/README.md` for a new server put this file into the volume, and the app starts with it

Not tested: the `caddy` service with a real DNS name, and a backup to a real S3 bucket.

# Unresolved questions
[unresolved-questions]: #unresolved-questions

None.
