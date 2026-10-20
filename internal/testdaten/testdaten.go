// Package testdaten legt Zeilen in der Testdatenbank an und leert sie: Titel, Exemplare, Leser,
// Konten, Ausleihen, Bestellungen, Forderungen. Was es anlegt, schreibt es unmittelbar in die
// Tabelle, nicht über den Weg des Programms: Ein Test stellt damit eine Lage her und prüft dann
// die Tür. Was über den Weg des Programms geht, steht in internal/testweg. Pakete des Projekts
// bindet testdaten nicht ein, außer internal/pgtest und zwei aus pkg/ ohne eigene Abhängigkeiten:
// So können auch die Tests von repository, db, inventur und internal/service es einbinden.
//
// Wie internal/pgtest ist es ein gewöhnliches Paket, das nur Tests einbinden: Eine Testdatei
// lässt sich nicht aus einem anderen Paket einbinden, und die Tests der Türen stehen in mehr
// als einem. Die Liste des toten Codes (scripts/deadcode_baseline.txt) führt seine Funktionen
// deshalb mit Begründung.
package testdaten

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"bibliothek/pkg/schulzeit"
)

// LeereBestandsdaten leert Bestands-, Bestell- und Personendaten zwischen Tests. klassen und
// ausweisnummern_ausgeschieden gehören dazu: Der Trigger der Klassen-Schreibweise und der
// Generator der Ausweisnummern lesen sie, und ein Rest aus einem früheren Test änderte, was der
// nächste speichert.
func LeereBestandsdaten(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `
		TRUNCATE buecher_exemplare, buecher_titel, ausleihen, leser, benutzer, klassen,
		         ausweisnummern_ausgeschieden
		RESTART IDENTITY CASCADE
	`)
	if err != nil {
		t.Fatalf("Reset fehlgeschlagen: %v", err)
	}
}

// Datum liefert einen Kalendertag in UTC als Zeiger, wie ihn die Datumsfelder eines Lesers tragen.
func Datum(j int, m time.Month, t int) *time.Time {
	d := time.Date(j, m, t, 0, 0, 0, 0, time.UTC)
	return &d
}

// SaisonUhr ist eine feste Uhr in der Saison der Abgänger. Ohne sie hinge ein Test am Kalender
// des Rechners.
func SaisonUhr() time.Time { return time.Date(2026, time.June, 15, 10, 0, 0, 0, schulzeit.Zone()) }

// WinterUhr ist eine feste Uhr außerhalb der Saison der Abgänger.
func WinterUhr() time.Time { return time.Date(2026, time.October, 15, 10, 0, 0, 0, schulzeit.Zone()) }

// ZeigerAuf liefert einen Zeiger auf den Text.
func ZeigerAuf(s string) *string { return &s }
