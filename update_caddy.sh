#!/bin/bash
cat /root/caddy/Caddyfile | sed '/flasch3.herzog-dupont.de {/,/}/d' > /root/caddy/Caddyfile.new
cat << 'INNER_EOF' >> /root/caddy/Caddyfile.new
flasch3.herzog-dupont.de {

    handle /* {
        reverse_proxy bibliothek-backend:8083
    }
}
INNER_EOF
mv /root/caddy/Caddyfile.new /root/caddy/Caddyfile
docker restart caddy
