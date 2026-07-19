#!/usr/bin/env bash
# Nightly MySQL backup for classyfm. Installed as a systemd timer (see
# classyfm-backup.timer / .service, wired up by install.sh).
set -euo pipefail

APP_DIR=/home/remorac/classyfm
BACKUP_DIR=/home/remorac/classyfm/backups
RETENTION_DAYS=14

mkdir -p "$BACKUP_DIR"

DSN=$(grep '^DATABASE_DSN=' "$APP_DIR/.env" | cut -d= -f2-)
DB_USER=$(echo "$DSN" | sed -E 's#^([^:]+):.*#\1#')
DB_PASS=$(echo "$DSN" | sed -E 's#^[^:]+:([^@]+)@.*#\1#')
DB_NAME=$(echo "$DSN" | sed -E 's#^.*\)/([^?]+).*#\1#')

TIMESTAMP=$(date +%Y%m%d-%H%M%S)
OUT="$BACKUP_DIR/classyfm-$TIMESTAMP.sql.gz"

MYSQL_PWD="$DB_PASS" mysqldump --single-transaction --routines --user="$DB_USER" "$DB_NAME" | gzip > "$OUT"

# Prune backups older than RETENTION_DAYS so this doesn't grow unbounded.
find "$BACKUP_DIR" -name 'classyfm-*.sql.gz' -mtime "+$RETENTION_DAYS" -delete

echo "Backup written to $OUT"
