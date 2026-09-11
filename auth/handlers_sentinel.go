package auth

import "errors"

// ErrAnmeldedienstGestoert heißt: Die Anmeldung selbst ist kaputt (Datenbank nicht erreichbar,
// Pool erschöpft, Frist gerissen) — im Unterschied zu „falsches Passwort" (401) und zum
// Mailserver-Ausfall (ErrMailserverNichtErreichbar). Der Handler antwortet 503 und zählt
// KEINEN Fehlversuch: Bis zum 31.08.2026 sperrte sich sonst eine Lehrkraft mit korrektem
// Passwort während eines DB-Aussetzers selbst für 15 Minuten.
var ErrAnmeldedienstGestoert = errors.New("anmeldedienst gestört — Anmeldung derzeit nicht möglich, bitte später erneut versuchen")

// ErrPruefungGestoert heißt: Die Sitzung ließ sich nicht prüfen, weil die Datenbank nicht
// geantwortet hat (Sperrliste oder Kontostatus). Die Anfrage wird abgelehnt, aber mit 503
// statt 401 — eine 401 meldet den Arbeitsplatz im Client ab, und ein kurzer Aussetzer der
// Datenbank hätte alle Arbeitsplätze abgemeldet.
var ErrPruefungGestoert = errors.New("sitzung konnte nicht geprüft werden, bitte erneut versuchen")
