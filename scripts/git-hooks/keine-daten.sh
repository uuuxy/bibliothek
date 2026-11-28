#!/bin/sh
# Liest Dateipfade von stdin und bricht ab, wenn einer ein Datenformat hat. Exporte aus Littera
# und Drucke daraus tragen Namen, Ausweisnummern und Passfotos von Kindern, und das Repository
# ist öffentlich. Aufgerufen von pre-commit (vorgemerkte Dateien) und pre-push (Dateien der
# Commits, die hinausgehen); .gitignore deckt dieselben Formate, hält aber `git add -f` nicht auf.
verboten=$(grep -iE '\.(mdb|accdb|bak|abf|csv|xls|xlsx|pdf)$|(^|/)item\.data$' || true)
if [ -n "$verboten" ]; then
	echo "✗ Daten-Dateien gehören nicht ins Repository (öffentlich auf GitHub):" >&2
	echo "$verboten" | sed 's/^/    /' >&2
	exit 1
fi
exit 0
