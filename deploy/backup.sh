#!/bin/sh
# Makes a backup of /pb_data in /backups each BACKUP_INTERVAL seconds.
# BACKUP_KEEP database copies stay. "backup once" makes one backup and stops.
set -u

DATA=/pb_data
OUT=/backups
INTERVAL="${BACKUP_INTERVAL:-86400}"
KEEP="${BACKUP_KEEP:-14}"
# The health check of the service fails while this file exists.
FAILED=/tmp/backup-failed

# Each step returns at its first error. set -e does not stop a function
# that runs on the left of ||.
copy_database() {
  name="$1"
  stamp="$2"
  tmp="$OUT/db/$name-$stamp.db.tmp"
  # .backup makes a consistent copy while the app writes.
  sqlite3 "$DATA/$name.db" ".timeout 10000" ".backup '$tmp'" || return 1
  [ "$(sqlite3 "$tmp" 'PRAGMA integrity_check;')" = "ok" ] || return 1
  gzip -f "$tmp" || return 1
  mv "$tmp.gz" "$OUT/db/$name-$stamp.db.gz" || return 1
  # The names sort by time. The oldest copies go.
  ls -1 "$OUT/db/$name"-*.db.gz | sort -r | tail -n "+$((KEEP + 1))" | xargs -r rm -f
}

backup() {
  stamp="$(date -u +%Y%m%dT%H%M%SZ)"
  mkdir -p "$OUT/db" "$OUT/storage" || return 1
  rm -f "$OUT"/db/*.tmp "$OUT"/db/*.tmp.gz
  for name in data auxiliary; do
    if [ -f "$DATA/$name.db" ]; then
      copy_database "$name" "$stamp" || return 1
    fi
  done
  # The photo files do not change after the upload. rsync copies only new files.
  if [ -d "$DATA/storage" ]; then
    rsync -a --delete "$DATA/storage/" "$OUT/storage/" || return 1
  fi
  echo "backup $stamp done"
}

if [ "${1:-}" = "once" ]; then
  backup
  exit $?
fi

while true; do
  if backup; then
    rm -f "$FAILED"
  else
    echo "backup failed" >&2
    touch "$FAILED"
  fi
  sleep "$INTERVAL"
done
