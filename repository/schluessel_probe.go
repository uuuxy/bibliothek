package repository

import (
	"context"
	"errors"
	"fmt"

	"bibliothek/db"
	"bibliothek/internal/crypto"

	"github.com/jackc/pgx/v5"
)

// VerschluesselteSpalte beschreibt eine Spalte, deren Inhalt mit APP_ENCRYPTION_KEY
// verschlüsselt ist.
type VerschluesselteSpalte struct {
	Beschreibung string
	Tabelle      string
	IDSpalte     string
	DatenSpalte  string
}

// VerschluesselteSpalten ist die EINE Liste dieser Spalten. Zwei Leser: der
// Schlüsselwechsel (cmd/rotate-encryption-key) und die Probe darunter. Bis zum 21.09.2026
// stand die Liste nur im Werkzeug; eine zweite für die Probe wäre die Stelle gewesen, an
// der eine neue Spalte vergessen wird. Dass die Liste vollständig ist, prüft
// TestRotiere_JedeVerschluesselteSpalteStehtInDerListe gegen die Datenbank.
//
// Tabellen- und Spaltennamen gehen in SQL-Text ein — sie stammen ausschließlich aus
// dieser Konstante, nie aus einer Eingabe.
var VerschluesselteSpalten = []VerschluesselteSpalte{
	{"Schülerfotos", "schueler_fotos", "schueler_id", "foto_encrypted"},
	{"SMTP-Passwort", "mail_settings_config", "id", "smtp_password_encrypted"},
}

// SchluesselProbe ist das Ergebnis der Frage: Passt APP_ENCRYPTION_KEY zum Bestand?
type SchluesselProbe struct {
	// Geprueft: wie viele Spalten einen Wert zum Probieren hatten. 0 heißt: Es ist noch
	// nichts verschlüsselt abgelegt — dann kann der Schlüssel auch nichts verfehlen.
	Geprueft int
	// NichtLesbar: Beschreibungen der Spalten, deren Probe sich nicht entschlüsseln ließ.
	NichtLesbar []string
}

// PruefeSchluesselGegenBestand entschlüsselt je verschlüsselter Spalte EINEN Wert.
//
// Anlass (21.09.2026): Nichts im System merkte, wenn der Schlüssel in der .env nicht zu
// den Daten passt — nach einer Wiederherstellung auf einem neuen Server mit neuer .env
// etwa. Sichtbar wurde es erst dort, wo es weh tut: Ein Foto bleibt leer, die Mahnmail
// geht nicht raus, weil das SMTP-Passwort nicht lesbar ist.
//
// GRENZE: eine Stichprobe je Spalte, kein Durchlauf. Sie findet den falschen Schlüssel,
// nicht einen Bestand mit zwei Schlüsseln — den verhindert der Schlüsselwechsel selbst
// (ein nicht lesbarer Datensatz bricht ihn ab).
func PruefeSchluesselGegenBestand(ctx context.Context, pool db.PgxPoolIface) (SchluesselProbe, error) {
	var probe SchluesselProbe
	for _, s := range VerschluesselteSpalten {
		// #nosec G201 -- Namen aus der Konstante oben, nie aus einer Eingabe.
		abfrage := fmt.Sprintf(`SELECT %s FROM %s WHERE %s IS NOT NULL AND length(%s) > 0 LIMIT 1`,
			s.DatenSpalte, s.Tabelle, s.DatenSpalte, s.DatenSpalte)
		var wert []byte
		err := pool.QueryRow(ctx, abfrage).Scan(&wert)
		if errors.Is(err, pgx.ErrNoRows) {
			continue
		}
		if err != nil {
			return SchluesselProbe{}, fmt.Errorf("%s lesen: %w", s.Beschreibung, err)
		}
		probe.Geprueft++
		if _, err := crypto.Decrypt(wert); err != nil {
			probe.NichtLesbar = append(probe.NichtLesbar, s.Beschreibung)
		}
	}
	return probe, nil
}
