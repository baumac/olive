---
description: Deploy the app as one instance with Docker Compose, Caddy for TLS, and a backup service for pb_data.
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
- a backup of the database and the photos is made each day
- a person can restore the app from a backup with written steps
- CI fails when the image does not build

# Scope
[scope]: #scope

In scope:
- `deploy/Dockerfile`, `deploy/compose.yaml`, `deploy/Caddyfile`, `deploy/backup.sh`, `deploy/.env.example`, and `deploy/README.md`
- the `make build-image` target and the `docker` job of CI
- the rule of one instance

Out of scope:
- a push of the image to a registry, and a deploy from CI
- the copy of the backups to a different computer. `deploy/README.md` tells the operator to do it.
- more than one instance
- the document about OAuth2 sign in. It stays in the backlog.
- monitoring and log collection

# Guide-level explanation
[guide-level-explanation]: #guide-level-explanation

`deploy/README.md` is the guide for an operator. It has the steps to start, update, back up, and restore.

Docker Compose runs three services:

| Service | Image | What |
| --- | --- | --- |
| `app` | `reviews-app:local` | the app server on port 8090 with the `pb_data` volume |
| `caddy` | `caddy` | the reverse proxy on the ports 80 and 443. It gets the TLS certificate of `SITE_ADDRESS`. |
| `backup` | `reviews-app:local` | runs `backup` in a loop |

`deploy/.env` has `SITE_ADDRESS`, the backup settings, and the `REVIEWS_` variables of the app.
`compose.yaml` names each `REVIEWS_` variable. A new variable of the app needs a line there.

## One instance

The app runs as one instance.

- SQLite has one writer process
- the sign in limits, the photo upload limit, the place search limit, and the place cache are in the memory of the process
- a restart sets the limits to zero. This is accepted.
- a second instance needs a shared store for these values. That is a new decision.

## Backups

- `backup` copies `data.db` and `auxiliary.db` with the `.backup` command of `sqlite3`. The copy is consistent while the app writes.
- each copy must pass `PRAGMA integrity_check`. It is then a `gzip` file with the time in its name. The newest 14 copies stay.
- a step that fails stops the backup. The old copies stay, and the health check of the `backup` service fails until the next good backup.
- `rsync` mirrors `pb_data/storage` to `storage/` of the backup folder. Photo files do not change after the upload, so only new files are copied.
- the photos have no history. A photo that the app deleted is deleted in the mirror at the next backup.
  A restore of an old database copy can then have photo records without files. The photo cleanup job of decision 00009 does not remove these records.
- the backup folder is a folder of the host. The operator copies it to a different computer.

# Implementation Details
[implementation-details]: #implementation-details

## Image

- the build context is the root of the repository. `.dockerignore` keeps `node_modules`, `bin`, `app/pb_data`, the docs, and the tests out of it.
- each base image has an exact version tag. Renovate updates the tags. See decision 00024.
- stage `ui` uses the Node.js 22 image. It runs `npm ci` for the SDK and UI workspaces and builds both.
- stage `build` uses the Go image with the toolchain version of `app/go.mod`. It copies the UI build into `app/internal/webui/dist` and builds the app with `CGO_ENABLED=0`.
- the last stage uses Alpine with `sqlite`, `rsync`, `tzdata`, and `ca-certificates`. The app runs as the user `reviews` with the ID 10001.
- the image has a health check on `/api/health`
- the image size is about 67 MB

| Option for the backup tools | Result |
| --- | --- |
| `sqlite` and `rsync` in the app image | Selected. One image to build and to update. |
| a second image for the backup | Not selected. It needs a second Dockerfile and a second build in CI. |
| the backup feature of PocketBase | Not selected. Each backup is one zip file of all of `pb_data`. The photos make each file large. |

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
- two runs of `backup once` with `BACKUP_KEEP=1` leave one copy of each database
- a copy passes `PRAGMA integrity_check` after `gunzip`

Not tested: the `caddy` service with a real DNS name, and the restore steps on a second computer.

# Unresolved questions
[unresolved-questions]: #unresolved-questions

None.
