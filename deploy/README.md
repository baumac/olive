# Deploy

This folder runs the app as one instance with Docker Compose. Decision 00022 has the reasons.

| Service | What |
| --- | --- |
| `app` | the app server with its data in the `pb_data` volume |
| `caddy` | the reverse proxy. It gets the TLS certificate of the site. |
| `backup` | makes a backup of `pb_data` each day |

## Start

You need a server with Docker, and a DNS name that points to it. The ports 80 and 443 must be open.

```bash
cp deploy/.env.example deploy/.env    # then set SITE_ADDRESS
mkdir -p deploy/backups
sudo chown 10001 deploy/backups       # the user ID of the containers
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

The app applies the migrations when it starts. Make a backup first:

```bash
docker compose -f deploy/compose.yaml run --rm backup once
```

## Backups

The `backup` service writes to `BACKUP_DIR` on the host. The default is `deploy/backups`.

- `db/data-TIME.db.gz` and `db/auxiliary-TIME.db.gz` are copies of the databases. SQLite makes them while the app runs. The newest `BACKUP_KEEP` copies stay.
- `storage/` is a mirror of the photo files. A photo that the app deleted is also deleted there at the next backup.

`docker compose -f deploy/compose.yaml ps` shows the `backup` service as `unhealthy` after a backup that failed. Its log has the error.
A backup fails when the user ID 10001 cannot write to `BACKUP_DIR`.

A backup on the same disk does not help when the disk fails.
Copy `BACKUP_DIR` to a different computer, for example with `restic` or `rclone` from a cron job of the host.

## Restore

```bash
docker compose -f deploy/compose.yaml stop app backup
docker compose -f deploy/compose.yaml run --rm --no-deps --entrypoint sh backup -c '
  rm -f /pb_data/data.db-wal /pb_data/data.db-shm /pb_data/auxiliary.db-wal /pb_data/auxiliary.db-shm &&
  gunzip -c /backups/db/data-TIME.db.gz > /pb_data/data.db &&
  gunzip -c /backups/db/auxiliary-TIME.db.gz > /pb_data/auxiliary.db &&
  rsync -a /backups/storage/ /pb_data/storage/'
docker compose -f deploy/compose.yaml up -d
```

Replace `TIME` with the time in the file names. Test a restore on a second computer before you need it.
