#!/usr/bin/env bash
# npm audit über die Abhängigkeiten der Oberfläche, mit der Regel dieses Repositorys. Der
# Sicherheits-Prüflauf (.github/workflows/security-scan.yml) und der Hook vor dem Push
# (scripts/git-hooks/pre-push) rufen beide dieses Skript: Was die CI rot macht, soll am
# Arbeitsplatz auffallen, und was sie bewusst nicht wertet, soll dort nicht warnen.
#
# Rot ist eine Schwachstelle HIGH oder CRITICAL im ausgelieferten Programm
# (--audit-level=high --omit=dev). Niedrigere Stufen und Pakete, die nur Bau und Prüfung
# brauchen, blockieren in einer Schule nicht; die Ausgabe nennt, was npm im ausgelieferten Teil
# findet.
#
# Ein Ausfall der Registry ist kein Befund: Der Audit-Endpunkt ist manchmal kurz weg und
# antwortet dann mit „Service Unavailable". Ein Gate, das bei einem fremden Ausfall rot wird,
# erzieht zum Wegsehen. Deshalb VERSUCHE Anläufe (Vorgabe 1, die CI nimmt 3) im Abstand von
# NPM_AUDIT_PAUSE Sekunden (Vorgabe 20), danach Exit 2 statt 1.
#
# Aufruf: ./scripts/npm-audit-gate.sh   (aus jedem Ordner; das Skript wechselt nach frontend/)
# Exit:   0 kein Befund · 1 Befund HIGH oder CRITICAL · 2 Audit-Endpunkt nicht erreichbar
set -uo pipefail
cd "$(dirname "$0")/../frontend" || exit 1

versuche="${VERSUCHE:-1}"
pause="${NPM_AUDIT_PAUSE:-20}"
for ((versuch = 1; versuch <= versuche; versuch++)); do
	if ausgabe=$(npm audit --audit-level=high --omit=dev 2>&1); then
		echo "$ausgabe"
		exit 0
	fi
	echo "$ausgabe"
	if ! grep -qiE "audit endpoint returned an error|Service Unavailable|ENOTFOUND|ETIMEDOUT|ECONNRESET" <<<"$ausgabe"; then
		exit 1
	fi
	if ((versuch < versuche)); then
		echo "Versuch $versuch: Audit-Endpunkt nicht erreichbar, neuer Versuch in ${pause} s …"
		sleep "$pause"
	fi
done
exit 2
