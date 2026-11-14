#!/bin/sh
# Makes one backup of /pb_data in /backups, then waits and makes the next.
# BACKUP_INTERVAL is the time between two backups in seconds.
# BACKUP_KEEP is the number of database copies that stay.
# "backup once" makes one backup and stops.
set -eu

DATA=/pb_data
OUT=/backups
INTERVAL="${BACKUP_INTERVAL:-86400}"
KEEP="${BACKUP_KEEP:-14}"

backup() {
  stamp="$(date -u +%Y%m%dT%H%M%SZ)"
  mkdir -p "$OUT/db" "$OUT/storage"
  for name in data auxiliary; do
    if [ -f "$DATA/$name.db" ]; then
      # .backup makes a consistent copy while the app writes.
      sqlite3 "$DATA/$name.db" ".timeout 10000" ".backup '$OUT/db/$name-$stamp.db.tmp'"
      gzip -f "$OUT/db/$name-$stamp.db.tmp"
      mv "$OUT/db/$name-$stamp.db.tmp.gz" "$OUT/db/$name-$stamp.db.gz"
      # The names sort by time. The oldest copies go.
      ls -1 "$OUT/db/$name"-*.db.gz | sort -r | tail -n "+$((KEEP + 1))" | xargs -r rm -f
    fi
  done
  # The photo files do not change after the upload. rsync copies only new files.
  if [ -d "$DATA/storage" ]; then
    rsync -a --delete "$DATA/storage/" "$OUT/storage/"
  fi
  echo "backup $stamp done"
}

if [ "${1:-}" = "once" ]; then
  backup
  exit 0
fi

while true; do
  backup || echo "backup failed" >&2
  sleep "$INTERVAL"
done
