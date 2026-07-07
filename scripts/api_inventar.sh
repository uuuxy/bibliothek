#!/bin/sh
# Erzeugt dokumentation/api_inventar.md: alle registrierten Go-Routen,
# alle /api/-Aufrufer im Frontend und den Abgleich in beide Richtungen
# (tote Handler / Geister-Aufrufe). Radar-Referenz aus dem Master-Fahrplan.
#
# Aufruf: ./scripts/api_inventar.sh   (vom Repo-Root)
set -eu

OUT=dokumentation/api_inventar.md
TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT

# 1. Go-Routen (alle Registrierungsorte, inkl. Subtree-Mounts ohne Methode)
grep -rhE '\.Handle(Func)?\(' api/ inventur/ --include="*.go" \
  | grep -oE '"(GET |POST |PUT |PATCH |DELETE )?/[^"]*"' \
  | tr -d '"' | sed 's/^ *//' | sort -u > "$TMP/go_routes.txt"

# 2. Frontend-Aufrufe (Literale und Template-Strings)
grep -rhoE '(/api/[A-Za-z0-9_/${}.?=&-]*)' frontend/src --include="*.js" --include="*.svelte" \
  | sed 's/[?].*$//' | sort -u > "$TMP/fe_calls.txt"

# 3. Abgleich
python3 - "$TMP" <<'EOF' > "$TMP/abgleich.txt"
import sys, os
tmp = sys.argv[1]

def norm(path):
    return ['*' if ('{' in s or '$' in s) else s for s in path.strip('/').split('/')]

go = []
for line in open(os.path.join(tmp, 'go_routes.txt')):
    line = line.strip()
    if not line: continue
    parts = line.split(' ', 1)
    method, path = (parts[0], parts[1]) if len(parts) == 2 else ('*', parts[0])
    # Subtree-Mounts (Pfad endet auf /) matchen als Präfix
    go.append((method, path, norm(path), path.endswith('/')))

fe = [(p.strip(), norm(p.strip())) for p in open(os.path.join(tmp, 'fe_calls.txt')) if p.strip()]

def matches(gsegs, fsegs, prefix):
    if prefix:
        if len(fsegs) < len(gsegs): return False
        gcmp, fcmp = gsegs, fsegs[:len(gsegs)]
    else:
        if len(gsegs) != len(fsegs): return False
        gcmp, fcmp = gsegs, fsegs
    return all(g == '*' or f == '*' or g == f for g, f in zip(gcmp, fcmp))

print("## Go-Routen ohne Frontend-Aufrufer\n")
print("(SSE `/events`, Dashboards, Public-Endpoints und Swagger können legitim ohne SPA-Aufrufer sein — vor dem Löschen prüfen!)\n")
for method, path, gsegs, _ in go:
    if not path.startswith('/api/'): continue
    if not any(matches(gsegs, fsegs, False) for _, fsegs in fe):
        print(f"- `{method} {path}`")

print("\n## Frontend-Aufrufe ohne Go-Route (Geister-Aufrufe = Bugs!)\n")
for p, fsegs in fe:
    if not any(matches(gsegs, fsegs, pre) for _, _, gsegs, pre in go):
        print(f"- `{p}`")
EOF

{
  echo "# API-Inventar (generiert)"
  echo
  echo "> Generiert von \`scripts/api_inventar.sh\` am $(date +%Y-%m-%d). Nicht von Hand editieren."
  echo
  cat "$TMP/abgleich.txt"
  echo
  echo "## Alle registrierten Routen ($(wc -l < "$TMP/go_routes.txt" | tr -d ' '))"
  echo
  sed 's/^/- `/;s/$/`/' "$TMP/go_routes.txt"
} > "$OUT"

echo "geschrieben: $OUT"
