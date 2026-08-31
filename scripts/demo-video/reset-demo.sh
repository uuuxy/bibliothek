#!/bin/bash
set -euo pipefail
S="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
docker compose -f $S/docker-compose.demo.yml down -v >/dev/null 2>&1
docker compose -f $S/docker-compose.demo.yml up -d >/dev/null 2>&1
for i in $(seq 1 60); do curl -sf http://localhost:8085/health >/dev/null && break; sleep 1; done
sleep 2
docker exec -i bibliothek-db-demo psql -U postgres -d bibliothek -v ON_ERROR_STOP=1 -q < $S/seed_video.sql | tail -1
docker exec -i bibliothek-db-demo psql -U postgres -d bibliothek -q -c "update system_einstellungen set wert='70' where schluessel='bestellbedarf_schwelle'" -c "update buecher_titel set last_counted = current_date - (abs(hashtext(titel)) % 120)"
docker exec bibliothek-backend-demo sh -c 'touch /app/backups/backup_2026-08-29_0300.sql.gz.enc'
cd $S && node api.mjs POST /api/admin/sync-covers
for i in $(seq 1 60); do n=$(docker exec -i bibliothek-db-demo psql -U postgres -d bibliothek -tA -c "select count(*) from buecher_titel where cover_status='PENDING'"); [ "$n" = "0" ] && break; sleep 1; done
echo "Cover: $(docker exec -i bibliothek-db-demo psql -U postgres -d bibliothek -tA -c "select cover_status||'='||count(*) from buecher_titel group by 1" | tr '\n' ' ')"
