# Deploy

This folder runs the app as one instance with Docker Compose. Decision 00022 has the reasons.

| Service | What |
| --- | --- |
| `app` | the app server with its data in the `pb_data` volume |
| `caddy` | the reverse proxy. It gets the TLS certificate of the site. |

## Start

You need a server with Docker, and a DNS name that points to it. The ports 80 and 443 must be open.

```bash
cp deploy/.env.example deploy/.env    # then set SITE_ADDRESS and PB_ENCRYPTION_KEY
docker compose -f deploy/compose.yaml up -d --build
```

`PB_ENCRYPTION_KEY` must have 32 characters. `openssl rand -hex 16` makes one. See the Settings encryption section.

Make the first superuser:

```bash
docker compose -f deploy/compose.yaml exec app reviews-app superuser create EMAIL PASSWORD --dir /pb_data --encryptionEnv PB_ENCRYPTION_KEY
```

Each `reviews-app` command in the container needs `--dir /pb_data --encryptionEnv PB_ENCRYPTION_KEY`.

Then open `https://SITE_ADDRESS/_/` and set these values. AGENTS.md has the reasons in its Production section.

- the Application URL: `https://SITE_ADDRESS`
- the trusted proxy header: `X-Real-IP`
- the superuser IP allow-list
- the backups. See the Backups section.
- the mail server and the MFA of the superusers. See the Mail and superuser MFA section.

## Settings encryption

The PocketBase settings have secrets, such as the SMTP password and the S3 secret of the backups.
The app encrypts the settings in the database with `PB_ENCRYPTION_KEY` of `deploy/.env`.

- The app does not start without a key of 32 characters.
- The backups do not have the key. A restore needs the same key, or the app cannot read the settings.
- Keep a copy of the key in a different place than the server and the bucket.

## Mail and superuser MFA

The dashboard is the only admin surface. Its sign in needs a second factor.
The second factor is a code that PocketBase sends by email, so set the mail server first.

1. Open Settings > Mail settings. Set the sender address, turn on "Use SMTP mail server", and set the host, the port, the user name, and the password.
2. Select "Send test email" and check that the email arrives.
3. Open the `_superusers` collection. It is one of the system collections.
4. Open its settings, and select Options.
5. Turn on "One-time password (OTP)" and "Multi-factor authentication (MFA)", and save.
6. Sign out and sign in again. The dashboard asks for the password and then for the code of the email.

Each superuser needs an email address that gets mail.
When the mail server does not work, make a code on the server:

```bash
docker compose -f deploy/compose.yaml exec app reviews-app superuser otp EMAIL --dir /pb_data --encryptionEnv PB_ENCRYPTION_KEY
```

## Limits of the containers

| Limit | Default | Where |
| --- | --- | --- |
| memory of the `app` container | 768 MB | `APP_MEMORY_LIMIT` in `deploy/.env` |
| memory target of the Go runtime | 600 MiB | `APP_GOMEMLIMIT` in `deploy/.env` |
| open files of the `app` container | 65536 | `deploy/compose.yaml` |
| request body in Caddy | 11 MB | `deploy/Caddyfile` |

The memory defaults are for a server with 1 GB. On a larger server, raise both values.
Keep `APP_GOMEMLIMIT` at about 80 percent of `APP_MEMORY_LIMIT`. Docker stops a container that gets to its memory limit.
The app accepts a photo of up to 10 MiB. The limit of Caddy must stay above it.

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
Such a user can sign in only when `verified` is also on. Without it, the person sets a password with "Forgot your password?".

The app sends emails for the sign up, the password reset, and the approval. See decision 00037.

- Each superuser gets an email when a new user confirms the email address. The user gets an email after the approval.
- Set the mail server first. See the Mail and superuser MFA section. Without it, the app writes each email with its link to the log of the dashboard, and no person gets it.
- Set the Application name and the Application URL in the dashboard. The emails use both.

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

On a new server, set `PB_ENCRYPTION_KEY` in `deploy/.env` to the key of the old server.
Download the ZIP file from the bucket to the folder of the repository. Then put its content into the volume:

```bash
docker compose -f deploy/compose.yaml up -d --build
docker compose -f deploy/compose.yaml stop app
docker compose -f deploy/compose.yaml run --rm --no-deps -v "$PWD/BACKUP.zip:/tmp/backup.zip:ro" --entrypoint sh app -c '
  rm -rf /pb_data/* && unzip -q /tmp/backup.zip -d /pb_data'
docker compose -f deploy/compose.yaml up -d
```

Test a restore on a second computer before you need it.
