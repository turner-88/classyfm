#!/usr/bin/env bash
# One-time provisioning for classyfm.co.id. Run as root: sudo bash deploy/install.sh
set -euo pipefail

DOMAIN=classyfm.co.id
ADMIN_EMAIL=remorac.14@gmail.com
APP_DIR=/home/remorac/classyfm

if [[ $EUID -ne 0 ]]; then
  echo "Run with sudo: sudo bash $0" >&2
  exit 1
fi

DSN=$(grep '^DATABASE_DSN=' "$APP_DIR/.env" | cut -d= -f2-)
DB_USER=$(echo "$DSN" | sed -E 's#^([^:]+):.*#\1#')
DB_PASS=$(echo "$DSN" | sed -E 's#^[^:]+:([^@]+)@.*#\1#')

echo "==> Installing nginx + certbot"
apt-get update
apt-get install -y nginx certbot python3-certbot-nginx

echo "==> Creating MariaDB database + user"
mysql <<SQL
CREATE DATABASE IF NOT EXISTS classyfm CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE USER IF NOT EXISTS '${DB_USER}'@'localhost' IDENTIFIED BY '${DB_PASS}';
GRANT ALL PRIVILEGES ON classyfm.* TO '${DB_USER}'@'localhost';
FLUSH PRIVILEGES;
SQL

echo "==> Applying migrations"
mysql classyfm < "$APP_DIR/internal/db/migrations/0001_init.up.sql"

echo "==> Creating upload directory"
mkdir -p "$APP_DIR/web/uploads/programs"
chown -R remorac:remorac "$APP_DIR/web/uploads"

echo "==> Installing systemd unit"
cp "$APP_DIR/deploy/classyfm.service" /etc/systemd/system/classyfm.service
systemctl daemon-reload
systemctl enable --now classyfm

echo "==> Installing nginx site"
cp "$APP_DIR/deploy/cloudflare-realip.conf" /etc/nginx/conf.d/cloudflare-realip.conf
cp "$APP_DIR/deploy/$DOMAIN" /etc/nginx/sites-available/$DOMAIN
ln -sf /etc/nginx/sites-available/$DOMAIN /etc/nginx/sites-enabled/$DOMAIN
rm -f /etc/nginx/sites-enabled/default
nginx -t
systemctl enable --now nginx
systemctl reload nginx

echo "==> Requesting Let's Encrypt certificate (Cloudflare Full/strict compatible)"
certbot --nginx -d "$DOMAIN" --redirect -m "$ADMIN_EMAIL" --agree-tos -n

echo "==> Installing nightly DB backup timer"
chmod +x "$APP_DIR/deploy/backup.sh"
cp "$APP_DIR/deploy/classyfm-backup.service" /etc/systemd/system/classyfm-backup.service
cp "$APP_DIR/deploy/classyfm-backup.timer" /etc/systemd/system/classyfm-backup.timer
systemctl daemon-reload
systemctl enable --now classyfm-backup.timer

echo "==> Done."
systemctl --no-pager status classyfm nginx | head -40
