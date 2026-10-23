package auth

import (
	"errors"
	"fmt"
)

// errAnmeldedienstGestoert heißt: Die Anmeldung selbst ist gestört (Datenbank nicht erreichbar,
// Pool erschöpft, Frist abgelaufen), im Unterschied zum falschen Passwort (401) und zum Ausfall
// des Mailservers (errMailserverNichtErreichbar). Die Tür antwortet mit 503 und zählt keinen
// Fehlversuch, sonst sperrte sich eine Lehrkraft mit richtigem Passwort während eines Aussetzers
// der Datenbank für 15 Minuten selbst aus.
var errAnmeldedienstGestoert = errors.New("anmeldedienst gestört — Anmeldung derzeit nicht möglich, bitte später erneut versuchen")

// ErrPruefungGestoert heißt: Die Sitzung ließ sich nicht prüfen, weil die Datenbank nicht
// geantwortet hat (Sperrliste oder Stand des Kontos). Die Anfrage wird abgelehnt, aber mit 503
// statt 401: Eine 401 meldet den Arbeitsplatz in der Oberfläche ab, und ein kurzer Aussetzer der
// Datenbank meldete alle Arbeitsplätze ab.
var ErrPruefungGestoert = errors.New("sitzung konnte nicht geprüft werden, bitte erneut versuchen")

// ErrWiderrufGestoert heißt: Die Abmeldung konnte die Sitzung nicht widerrufen, weil die
// Datenbank nicht antwortete, beim Eintrag in die Sperrliste oder schon bei der Prüfung davor.
// Das Cookie zum Löschen geht trotzdem hinaus, denn diese Hälfte hat stattgefunden; die Antwort
// meldet aber keinen Erfolg. Das Token bleibt bis zu seinem natürlichen Ablauf gültig, und wer
// das nicht erfährt, hält eine Sitzung für beendet, die es nicht ist.
var ErrWiderrufGestoert = errors.New("abmeldung unvollständig: die sitzung ließ sich nicht widerrufen, bitte erneut versuchen")

// ErrSitzungGesperrt heißt: Die Anmeldung gilt, ist aber gesperrt, bis das Passwort eingegeben
// ist. Der Server antwortet mit 423; die Oberfläche zeigt den Sperrbildschirm und meldet nicht ab.
var ErrSitzungGesperrt = errors.New("anmeldung gesperrt, bitte passwort eingeben")

// ErrHoechstdauerErreicht heißt: Seit der letzten Passworteingabe ist die Höchstdauer vergangen
// (hoechstdauerNachPasswort). Die Anmeldung ist gesperrt wie nach Inaktivität, errors.Is mit
// ErrSitzungGesperrt trifft auch sie; nur die Meldung nennt den anderen Grund.
var ErrHoechstdauerErreicht = fmt.Errorf("%w: höchstdauer nach der passworteingabe erreicht", ErrSitzungGesperrt)
