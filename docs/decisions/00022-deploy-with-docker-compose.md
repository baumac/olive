---
description: Deploy the app as one instance with Docker Compose and Caddy for TLS. PocketBase saves the backups of pb_data on the local file system. The backup task of the host copies them off the server.
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
- the backup task of the host. The operator sets it up and adds the folder of the `pb_data` volume to it.
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

`deploy/.env` has `SITE_ADDRESS`, `PB_ENCRYPTION_KEY`, and the `REVIEWS_` variables of the app.
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
- the backups go to `pb_data/backups` on the local file system
- the backup task of the host copies this folder to a different computer. It starts after the auto backups.
- each backup is one ZIP file of all of `pb_data`. It has the databases and the photo files.
- a restore on a server that runs is one action in the dashboard. A new server gets the content of the ZIP file in its volume.
- the compose file has no backup service. Without the dashboard settings, the app has no backup.

The first version of this decision had a `backup` service with `sqlite3 .backup` and `rsync` to a folder of the host.
It was removed on 2026-11-14. Its backups stayed on the server. The S3 storage of PocketBase needs no code.

On 2026-11-15 the backups moved from S3 to the local file system.
The host has a backup task that copies the entire host to a different computer.
This task gets the ZIP files off the server, so the site needs no bucket and no access key.

## Settings encryption and limits

These parts were added on 2026-11-15.

- the app encrypts the PocketBase settings in the database with `PB_ENCRYPTION_KEY`. The settings have the SMTP password.
- the backups do not have the key. The operator keeps a copy, and a restore needs it.
- the `app` container has a memory limit, a `GOMEMLIMIT` below it, and a limit of open files
- Caddy stops a request body of more than 11 MB
- `deploy/README.md` has the steps for the mail server and the MFA of the superusers

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
| the backup feature of PocketBase with local storage | Selected. It needs only settings. The backup task of the host copies the files to a different computer. |
| the backup feature of PocketBase with S3 storage | Not selected on 2026-11-15. The host backup already copies the files off the server. A bucket and an access key are one more thing to keep. |
| a service with `sqlite3 .backup` and `rsync` | Not selected. It copies each photo only one time, but it needs a script, tools in the image, and a second step to get the files off the server. |

Limits of the selected option:
- each ZIP file has all photos again. The disk needs space for `pb_data` and for `pb_data` times the number of backups.
- the app can be slower while a backup runs
- PocketBase advises the `sqlite3` and `rsync` plan for a `pb_data` of 5 GB or more. The site then needs a new decision.
- a failed backup shows only in the logs of the dashboard
- many host backups skip `/var/lib/docker`. The operator adds the folder of the volume to the task.
- a copy of the database files that the host task makes while the app runs can be broken. A restore uses a ZIP file.

## Settings encryption

- the start command of the image has `--encryptionEnv=PB_ENCRYPTION_KEY`. `compose.yaml` stops without the variable.
- the app does not start with a key that does not have 32 characters. The example value of `.env.example` is such a key.
- a `reviews-app` command with `docker compose exec` does not use the start command. It needs the flag, or it cannot read the settings.

| Option | Result |
| --- | --- |
| the flag in the start command | Selected. The app code does not change. |
| `DefaultEncryptionEnv` in `app/main.go` | Not selected. A local run with the variable in its environment then encrypts `app/pb_data`. |

## Limits

| Limit | Value | Reason |
| --- | --- | --- |
| `mem_limit` of `app` | 768 MB | The server has 1 GB. Caddy and the system get the other memory. |
| `GOMEMLIMIT` | 600 MiB | The Go runtime frees memory before Docker stops the container. Decision 00008 gives a peak of about 264 MB for the photos. |
| `ulimits.nofile` | 65536 | Each connection and each SQLite file uses a file descriptor. |
| `request_body` of Caddy | 11 MB | The limit of the app is 10 MiB, which is more than 10 MB. A body between the two limits gets the `413` of the app with Problem Details. |

`APP_MEMORY_LIMIT` and `APP_GOMEMLIMIT` of `deploy/.env` change the two memory values.

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

The settings encryption and the limits were tested by hand with `localhost` as `SITE_ADDRESS`:
- the app does not start with a key of 9 characters
- with a key of 32 characters, the `settings` row of `_params` is not readable JSON, and the app starts again after a restart
- `superuser create` fails without `--encryptionEnv` and passes with it
- `docker inspect` shows the memory limit and the limit of open files
- a body of 12 MB gets the `413` of Caddy. A body of 10.6 MB gets the `413` of the app.

Not tested: the `caddy` service with a real DNS name, the copy of the backups by a real host backup task, and the steps for the mail server and the MFA.

# Unresolved questions
[unresolved-questions]: #unresolved-questions

None.
