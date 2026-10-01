#!/bin/sh
set -eu
umask 077

project_dir=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "$project_dir"

if [ ! -f .env ]; then
    printf 'Missing .env; the database and encryption keys must be backed up together.\n' >&2
    exit 1
fi

backup_root=${1:-deploy/backups}
mkdir -p "$backup_root"
timestamp=$(date -u +%Y%m%dT%H%M%SZ)
staging_dir=$(mktemp -d "$backup_root/.inforadar-$timestamp.XXXXXX")
trap 'rm -rf -- "$staging_dir"' EXIT HUP INT TERM

# The password is expanded inside the database container, never by the host shell.
docker compose exec -T mysql sh -c '
    exec mysqldump --user=root --password="$MYSQL_ROOT_PASSWORD" \
        --single-transaction --quick --routines --events --triggers \
        --no-tablespaces --set-gtid-purged=OFF --databases inforadar
' > "$staging_dir/database.sql"
cp .env "$staging_dir/deployment.env"
chmod 600 "$staging_dir/database.sql" "$staging_dir/deployment.env"
git rev-parse HEAD > "$staging_dir/revision.txt"
docker compose images > "$staging_dir/images.txt"

backup_dir="$backup_root/inforadar-$timestamp-$$"
mv "$staging_dir" "$backup_dir"
trap - EXIT HUP INT TERM
printf 'Backup saved to %s (contains private credentials; copy it off this server).\n' "$backup_dir"
