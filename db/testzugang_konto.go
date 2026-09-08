package db

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5"
)

// insertTestzugangKontoSQL legt das Konto des außerordentlichen Testzugangs an.
//
// Ohne barcode_id: Der Zugang ist zum Ansehen des Systems da, nicht zum Ausleihen; ein
// Ausweis, der auf niemanden ausgestellt ist, hätte in der Theke nichts zu suchen. Die
// Spalte ist UNIQUE und nullable, NULL kollidiert also auch mit keinem echten Ausweis.
//
// Die Namensfelder sind bewusst sprechend statt neutral: In der Benutzerverwaltung, im
// Protokoll und in jeder Liste, in der Bearbeiter auftauchen, soll auf einen Blick zu
// sehen sein, dass hier kein Mensch der Schule gehandelt hat.
const insertTestzugangKontoSQL = `
	INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv)
	VALUES ('Testzugang', '(extern)', $1, 'admin', true)
	RETURNING id
`

// SichereTestzugangKonto stellt sicher, dass die in TESTZUGANG_EMAIL genannte Adresse
// einen Benutzereintrag hat — sonst prüft der Login das Sonderpasswort erfolgreich und
// scheitert danach am fehlenden Konto, was für den Tester wie ein falsches Passwort
// aussieht (die Klasse „Feature hängt an einer Einstellung, die nichts anlegt").
//
// Angelegt wird NUR, wenn die Zeile fehlt. Ein bereits vorhandenes Konto wird NICHT
// angefasst — weder auf aktiv noch auf admin gesetzt. Sonst wäre der Weg hierher eine
// Rechteerhöhung per Umgebungsvariable: Wer TESTZUGANG_EMAIL auf die Adresse einer
// Lehrkraft setzte, machte deren Konto beim nächsten Neustart still zum Administrator.
// Stimmt etwas mit dem vorhandenen Konto nicht, sagt der Start das, ändert aber nichts.
func (db *Database) SichereTestzugangKonto(ctx context.Context, email string) error {
	if email == "" {
		return nil
	}

	var id, rolle string
	var aktiv bool
	err := db.Pool.QueryRow(ctx,
		`SELECT id, rolle::text, aktiv FROM benutzer WHERE LOWER(email) = LOWER($1) LIMIT 1`,
		email).Scan(&id, &rolle, &aktiv)

	switch {
	case err == nil:
		// Vorhanden: unangetastet lassen, aber melden, wenn der Zugang damit nicht das
		// tut, was der Einrichtende erwartet.
		if !aktiv {
			log.Printf("Warnung: Testzugang-Konto %s ist INAKTIV — die Anmeldung wird abgelehnt. Unter Benutzer & Rechte freischalten.", email)
		}
		if rolle != "admin" {
			log.Printf("Warnung: Testzugang-Konto %s hat die Rolle %q, nicht admin — der Tester sieht nur die Rechte dieser Rolle. Bewusst nicht automatisch geändert.", email, rolle)
		}
		return nil
	case errors.Is(err, pgx.ErrNoRows):
		var neuID string
		if insErr := db.Pool.QueryRow(ctx, insertTestzugangKontoSQL, email).Scan(&neuID); insErr != nil {
			return fmt.Errorf("testzugang-konto anlegen: %w", insErr)
		}
		log.Printf("Testzugang-Konto %s wurde angelegt (Rolle admin, aktiv).", email)
		return nil
	default:
		return fmt.Errorf("testzugang-konto pruefen: %w", err)
	}
}
