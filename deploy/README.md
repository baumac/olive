# Deploy

This folder runs the app as one instance with Docker Compose. Decision 00022 has the reasons.

| Service | What |
| --- | --- |
| `app` | the app server with its data in the `pb_data` volume |
| `caddy` | the reverse proxy. It gets the TLS certificate of the site. |

## Start

You need a server with Docker, and a DNS name that points to it. The ports 80 and 443 must be open.

```bash
cp deploy/.env.example deploy/.env    # then set SITE_ADDRESS
docker compose -f deploy/compose.yaml up -d --build
```

Make the first superuser:

```bash
docker compose -f deploy/compose.yaml exec app reviews-app superuser create EMAIL PASSWORD --dir /pb_data
```

Then open `https://SITE_ADDRESS/_/` and set these values. AGENTS.md has the reasons in its Production section.

- the Application URL: `https://SITE_ADDRESS`
- the trusted proxy header: `X-Real-IP`
- the superuser IP allow-list
- the backups. See the Backups section.

## Approve new users

A new user cannot add or change data before a superuser approves the account. See decision 00034.

1. Open `https://SITE_ADDRESS/_/` and select the `users` collection.
2. Enter `approved = false` in the search field. The list then has the users that wait.
3. Open a user, turn on `approved`, and save.

Turn `approved` off again to stop the writes of a user. The user can still sign in and read.
Other people cannot open the reviewer page and the lists of a user without approval. The reviews and restaurants of the user stay public.
A daily job deletes a user that waits for more than 30 days and has no review, no restaurant, and no restaurant in a list. The person can sign up again.
The users from before the update to this version are approved.
A user that a superuser makes in the dashboard is approved only when `approved` is on.

## One instance only

Do not start a second `app` container on the same data.

- SQLite has one writer process.
- The sign in limits, the photo upload limit, the place search limit, and the place cache are in the memory of the process.
  A second instance has its own counts. A restart sets them to zero.

## Update

```bash
git pull
docker compose -f deploy/compose.yaml up -d --build
```

The app applies the migrations when it starts. Make a backup first with "New backup" in Settings > Backups of the dashboard.

## Backups

PocketBase makes the backups and sends them to an S3 compatible storage. Nothing in this folder makes a backup.
Without these settings, the app has no backup.

You need a bucket and an access key that can read and write it. Use a bucket that has only the backups.
In the dashboard, open Settings > Backups and set:

- the auto backups, with a cron expression. `0 3 * * *` makes one backup each day at 03:00 UTC.
- the maximum number of auto backups to keep, for example `7`
- the S3 storage, with the endpoint, the bucket, the region, the access key, and the secret

Then make one backup by hand and check that the file is in the bucket.

Each backup is one ZIP file of all of `pb_data`: the databases and the photo files.

- Each backup has all photos again. The bucket needs the size of `pb_data` times the number of backups.
- The app can be slower while a backup runs.
- PocketBase advises a different plan for a `pb_data` of 5 GB or more. Decision 00022 has the plan with `sqlite3` and `rsync`.
- A failed backup shows only in the logs of the dashboard. Look at the list of backups from time to time.
- The S3 settings are in the database. Keep a copy of the access key in a different place, or a lost server also loses the way to its backups.

## Restore

On a server that still runs, select a backup in Settings > Backups and select "Restore". The app replaces `pb_data` and starts again.

On a new server, download the ZIP file from the bucket to the folder of the repository. Then put its content into the volume:

```bash
docker compose -f deploy/compose.yaml up -d --build
docker compose -f deploy/compose.yaml stop app
docker compose -f deploy/compose.yaml run --rm --no-deps -v "$PWD/BACKUP.zip:/tmp/backup.zip:ro" --entrypoint sh app -c '
  rm -rf /pb_data/* && unzip -q /tmp/backup.zip -d /pb_data'
docker compose -f deploy/compose.yaml up -d
```

Test a restore on a second computer before you need it.
