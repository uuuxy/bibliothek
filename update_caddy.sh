#!/bin/bash
cat /root/caddy/Caddyfile | sed '/flasch3.herzog-dupont.de {/,/}/d' > /root/caddy/Caddyfile.new
cat << 'INNER_EOF' >> /root/caddy/Caddyfile.new
flasch3.herzog-dupont.de {
    tls /etc/caddy/certs/flasch3.crt /etc/caddy/certs/flasch3.key
    
    @backend {
        path /api/* /login /swagger/* /uploads/* /health
    }
    handle @backend {
        reverse_proxy bibliothek-backend:8083
    }
    
    handle /* {
        reverse_proxy bibliothek-frontend:3000
    }
}
INNER_EOF
mv /root/caddy/Caddyfile.new /root/caddy/Caddyfile
docker restart caddy
