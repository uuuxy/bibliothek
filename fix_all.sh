#!/bin/bash
sed -i 's|path /api/\* /login /swagger/\* /uploads/\* /health|path /api/* /login /swagger/* /uploads/* /health /events|' /root/caddy/Caddyfile
docker restart caddy

cd /root/bibliothek
./update.sh
