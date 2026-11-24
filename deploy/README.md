# Deploy

This folder runs the app as one instance with Docker Compose. Decision 00010 has the reasons.

| Service | What |
| --- | --- |
| `app` | the app server with its data in the `pb_data` volume |
| `cloudflared` | the Cloudflare Tunnel. Cloudflare has the TLS certificate of the site. |

AGENTS.md has the reasons for the settings in its Production section.

## First deploy

Do these steps in this sequence. Do not let users sign up before step 10.

1. Get a server with Docker. The server needs no open port for the site. Close the ports 80 and 443.

2. Make the file `deploy/.env`:

   ```bash
   cp deploy/.env.example deploy/.env
   ```

3. Make the tunnel. See [Cloudflare Tunnel](#cloudflare-tunnel). In `deploy/.env`, set `CLOUDFLARE_TUNNEL_TOKEN` to its token.
   `HOST` in the next steps is the public host name of the tunnel.

4. In `deploy/.env`, set `PB_ENCRYPTION_KEY` to a key of 32 characters. This command makes one:

   ```bash
   openssl rand -hex 16
   ```

   Keep a copy of the key in a different place than the server and its host backup. See [Settings encryption](#settings-encryption).

5. On a server with more than 1 GB of memory, raise `APP_MEMORY_LIMIT` and `APP_GOMEMLIMIT` in `deploy/.env`. See [Limits of the containers](#limits-of-the-containers).

6. Start the app:

   ```bash
   docker compose -f deploy/compose.yaml up -d --build
   ```

7. Make the first superuser:

   ```bash
   docker compose -f deploy/compose.yaml exec app reviews-app superuser create EMAIL PASSWORD --dir /pb_data --encryptionEnv PB_ENCRYPTION_KEY
   ```

   Each `reviews-app` command in the container needs `--dir /pb_data --encryptionEnv PB_ENCRYPTION_KEY`.

8. Open `https://HOST/_/` and sign in. Open Settings > Application and set:

   - the Application name: `Sponga47 Food`. The emails use it.
   - the Application URL: `https://HOST`
   - the trusted proxy header: `CF-Connecting-IP`
   - the superuser IP allow-list

9. Open Settings > Mail settings. Set the sender name to `Sponga47 Food` and set the sender address. Turn on "Use SMTP mail server", and set the host, the port, the user name, and the password.

10. Select "Send test email" and check that the email arrives.
    Without a mail server, the app writes each email with its link to the log of the dashboard, and no person gets it.

11. Open the `_superusers` collection. It is one of the system collections. Open its settings, and select Options.

12. Turn on "One-time password (OTP)" and "Multi-factor authentication (MFA)", and save.

13. Sign out and sign in again. The dashboard asks for the password and then for the code of the email.

14. Open Settings > Backups and set:

    - the auto backups, with a cron expression. `0 3 * * *` makes one backup each day at 03:00 UTC.
    - the maximum number of auto backups to keep, for example `7`

    Keep the S3 storage off. PocketBase then saves the backups in the `backups` folder of `pb_data` on the local file system.

15. Select "New backup". Check that the file is in the `backups` folder of the volume:

    ```bash
    sudo ls -l "$(docker volume inspect deploy_pb_data --format '{{ .Mountpoint }}')/backups"
    ```

    Docker Compose adds the project name `deploy` to the volume name. `docker volume ls` shows the name.

16. Make sure that the backup task of the host copies this folder to a different computer.
    Many host backups skip `/var/lib/docker`. Add the folder of step 15 to the task.
    Start the task after the auto backups of PocketBase, for example at 04:00 UTC.

17. Test a restore on a second computer. See [Restore on a new server](#restore-on-a-new-server).

## Approve new users

A new user cannot add or change data before a superuser approves the account. See decision 00003.
Each superuser gets an email when a new user confirms the email address.

1. Open `https://HOST/_/` and select the `users` collection.
2. Enter `approved = false` in the search field. The list then has the users that wait.
3. Open a user, turn on `approved`, and save. The user gets an email.

Turn `approved` off again to stop the writes of a user. The user can still sign in and read.
Other people cannot open the reviewer page and the lists of a user without approval. The reviews and restaurants of the user stay public.
A daily job deletes a user that waits for more than 30 days and has no review, no restaurant, and no restaurant in a list. The person can sign up again.
The users from before the update to this version are approved.
A user that a superuser makes in the dashboard is approved only when `approved` is on.
Such a user can sign in only when `verified` is also on. Without it, the person sets a password with "Forgot your password?".

## Update

1. In the dashboard, open Settings > Backups and select "New backup".
   Do not continue before the backup is in the list. A large backup can get the error 524 while it continues. See [Cloudflare Tunnel](#cloudflare-tunnel).
2. Get the new version and start it:

   ```bash
   git pull
   docker compose -f deploy/compose.yaml up -d --build
   ```

The app applies the migrations when it starts.

## Restore on a running server

1. In the dashboard, open Settings > Backups.
2. Select a backup and select "Restore". The app replaces `pb_data` and starts again.

## Restore on a new server

1. Do steps 1 to 3 of [First deploy](#first-deploy).
2. In `deploy/.env`, set `PB_ENCRYPTION_KEY` to the key of the old server.
3. Get the ZIP file of the backup from the host backup. Put it in the folder of the repository.
4. Start the containers, then stop the app:

   ```bash
   docker compose -f deploy/compose.yaml up -d --build
   docker compose -f deploy/compose.yaml stop app
   ```

5. Put the content of the ZIP file into the volume:

   ```bash
   docker compose -f deploy/compose.yaml run --rm --no-deps -v "$PWD/BACKUP.zip:/tmp/backup.zip:ro" --entrypoint sh app -c '
     rm -rf /pb_data/* && unzip -q /tmp/backup.zip -d /pb_data'
   ```

6. Start the app:

   ```bash
   docker compose -f deploy/compose.yaml up -d
   ```

## Reference

### Cloudflare Tunnel

The zone of the site must be in the Cloudflare account. The tunnel needs no open port and no certificate on the server.

1. In the Cloudflare dashboard, open Networks > Tunnels and make a tunnel of the type Cloudflared.
2. Select Docker as the environment. Copy the token from the `docker run` command. The token is a secret.
3. Add a public hostname: the host name of the site, the type `HTTP`, and the URL `app:8090`.

Set these options of the zone:

- SSL/TLS > Edge Certificates: turn on "Always Use HTTPS".
- Speed > Optimization: turn off Rocket Loader.
- Scrape Shield: turn off "Email Address Obfuscation".
- Web Analytics: do not use the automatic setup for the site.
- Security > Bots: turn off "Bot Fight Mode". Its challenge blocks the SDK clients and the link preview bots.

Rocket Loader, the email obfuscation, and the automatic analytics add scripts to the pages. The Content Security Policy of the app blocks them, and the UI can break.

Cloudflare caches only the paths with a file extension of its default list, such as `.js` and `.css`.
It then keeps the `Cache-Control` of the app. The UI files in `assets/` get one year, and the other UI files get `no-cache`.
The photo paths have no file extension, so Cloudflare does not cache them.

Cloudflare stops a request after 100 seconds with the error 524. The work on the server continues.
"New backup" in the dashboard waits for the ZIP file. With a large `pb_data`, it can get the 524.
After a 524, wait some minutes and look for the new backup in the list.

Cloudflare decrypts each request at its edge. It sees the passwords and the session cookies of the users.

### Settings encryption

The PocketBase settings have secrets, such as the SMTP password.
The app encrypts the settings in the database with `PB_ENCRYPTION_KEY` of `deploy/.env`.

- The app does not start without a key of 32 characters.
- The backups do not have the key. A restore needs the same key, or the app cannot read the settings.

### Superuser sign in without mail

Each superuser needs an email address that gets mail.
When the mail server does not work, make a code on the server:

```bash
docker compose -f deploy/compose.yaml exec app reviews-app superuser otp EMAIL --dir /pb_data --encryptionEnv PB_ENCRYPTION_KEY
```

### Limits of the containers

| Limit | Default | Where |
| --- | --- | --- |
| memory of the `app` container | 768 MB | `APP_MEMORY_LIMIT` in `deploy/.env` |
| memory target of the Go runtime | 600 MiB | `APP_GOMEMLIMIT` in `deploy/.env` |
| open files of the `app` container | 65536 | `deploy/compose.yaml` |
| request body in Cloudflare | 100 MB | the plan of the Cloudflare account |

The memory defaults are for a server with 1 GB. On a larger server, raise both values.
Keep `APP_GOMEMLIMIT` at about 80 percent of `APP_MEMORY_LIMIT`. Docker stops a container that gets to its memory limit.
The app accepts a photo of up to 10 MiB. It refuses a larger body with its own 413.

### Backups

PocketBase makes the backups and saves them in `pb_data/backups` on the local file system.
The backup task of the host copies them off the server. Nothing in this folder makes a backup.
Without the backup settings of the dashboard, the app has no backup.
Each backup is one ZIP file of all of `pb_data`: the databases and the photo files.

- Restore from a ZIP file. A copy of the database files that the host task makes while the app runs can be broken.
- Each backup has all photos again. The disk needs space for `pb_data` and for `pb_data` times the number of backups.
- The app can be slower while a backup runs.
- PocketBase advises a different plan for a `pb_data` of 5 GB or more. Decision 00010 has the plan with `sqlite3` and `rsync`.
- A failed backup shows only in the logs of the dashboard. Look at the list of backups from time to time.

### One instance only

Do not start a second `app` container on the same data.

- SQLite has one writer process.
- The sign in limits, the photo upload limit, the place search limit, and the place cache are in the memory of the process.
  A second instance has its own counts. A restart sets them to zero.
