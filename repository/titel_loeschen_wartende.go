package repository

// titel_loeschen_wartende.go — wer auf einen Titel wartet, wenn er gelöscht wird.
//
// Frage 12 „Gegenrichtung Schema" (06.09.2026): Die DDL sagt mehr als der Code. An
// `buecher_titel` hängen VIER Kinder mit ON DELETE CASCADE — Exemplare, `class_books`,
// `klassensatz_reservierungen` und `vormerkungen`. Der Löschpfad kannte davon nur die
// Exemplare (und über RESTRICT indirekt Ausleihen und Forderungen, die er seit dem
// 23.08. bzw. dem Rasterdurchgang desselben Tages protokolliert). Die beiden
// Warteschlangen fielen lautlos.
//
// Dahinter stehen Menschen, die etwas getan haben und darauf warten:
//
//   - Eine Vormerkung ist ein angestellter Schüler. Steht er auf Platz 1, ist der Titel
//     weg und mit ihm sein Platz — keine Mail, keine Notiz, und an der Theke ist später
//     nicht nachvollziehbar, dass er je gewartet hat.
//   - Eine Klassensatz-Reservierung ist die Anforderung einer Lehrkraft („Reservieren =
//     Anstellen"). Auch sie verschwindet ohne Bereit-Mail und ohne Absage.
//   - `class_books` ist die von Hand gepflegte Zuordnung eines Titels zu Klassensätzen.
//     Der Satz schrumpft, ohne dass irgendwo steht, warum.
//
// Dieselbe Antwort wie bei den offenen Ausleihen und Forderungen: Was verschwindet,
// steht vorher im Protokoll. Ein spurlos verschwundener Wartender ist ein verlorener
// Wartender.
//
// Warum hier und nicht im inventur-Paket: Es gibt ZWEI Türen zum Titel-Löschen — die
// Massenaktion der Bestandstabelle (DELETE /api/books, inventur/db_books_delete.go) und
// der Knopf der Buch-Akte (DELETE /api/buecher/titel/{id}, repository/audit_books.go).
// Beide fahren über denselben CASCADE. Zwei Fassungen derselben Regel wären genau die
// Bugklasse, gegen die Frage 3 gebaut ist; `inventur` importiert `repository`, also
// steht sie hier.

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// WartenderBezug ist eine Zeile, die mit dem Titel per CASCADE fällt.
type WartenderBezug struct {
	Tabelle   string // die Tabelle, in der die Zeile stand — sie steht so im Audit-Log
	ID        string
	TitelID   string
	Titel     string
	Wer       string // Schülername, Klasse oder Klassensatz-Name
	Status    string
	Seit      string
	SchuelerD *string // gesetzt bei Vormerkungen: Schlüssel für die Lesehistorie-Befristung
	Kontext   string
}

// wartendeAbfrager ist das, was zum Lesen genügt — ein Pool oder eine laufende
// Transaktion. Die eine Tür liest vor der Transaktion, die andere in ihr.
type wartendeAbfrager interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

// LeseWartendeBezuege sammelt Vormerkungen, Klassensatz-Reservierungen und
// Klassensatz-Zuordnungen der Titel — VOR dem Löschen, denn danach sind sie weg.
func LeseWartendeBezuege(ctx context.Context, q wartendeAbfrager, ids []string) ([]WartenderBezug, error) {
	var alle []WartenderBezug

	// Nur offene Vormerkungen: Eine erledigte Zeile beschreibt einen abgeschlossenen
	// Vorgang, sie kostet niemanden seinen Platz.
	rows, err := q.Query(ctx, `
		SELECT v.id::text, t.id::text, t.titel,
		       COALESCE(s.vorname || ' ' || s.nachname, 'unbekannt'),
		       v.status, to_char(v.erstellt_am, 'YYYY-MM-DD"T"HH24:MI:SSOF'), s.id::text
		FROM vormerkungen v
		JOIN buecher_titel t ON t.id = v.titel_id
		LEFT JOIN schueler s ON s.id = v.schueler_id
		WHERE v.titel_id = ANY($1::uuid[]) AND v.status <> 'erledigt'`, ids)
	if err != nil {
		return nil, fmt.Errorf("offene vormerkungen konnten nicht gelesen werden: %w", err)
	}
	for rows.Next() {
		var b WartenderBezug
		if err := rows.Scan(&b.ID, &b.TitelID, &b.Titel, &b.Wer, &b.Status, &b.Seit, &b.SchuelerD); err != nil {
			rows.Close()
			return nil, fmt.Errorf("offene vormerkungen konnten nicht gelesen werden: %w", err)
		}
		b.Tabelle = "vormerkungen"
		b.Kontext = "Titel gelöscht, eine Vormerkung stand noch offen"
		alle = append(alle, b)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("offene vormerkungen konnten nicht gelesen werden: %w", err)
	}

	rows, err = q.Query(ctx, `
		SELECT r.id::text, t.id::text, t.titel, r.klasse, 'offen',
		       to_char(r.erstellt_am, 'YYYY-MM-DD"T"HH24:MI:SSOF')
		FROM klassensatz_reservierungen r
		JOIN buecher_titel t ON t.id = r.titel_id
		WHERE r.titel_id = ANY($1::uuid[]) AND r.erledigt = false`, ids)
	if err != nil {
		return nil, fmt.Errorf("offene klassensatz-reservierungen konnten nicht gelesen werden: %w", err)
	}
	for rows.Next() {
		var b WartenderBezug
		if err := rows.Scan(&b.ID, &b.TitelID, &b.Titel, &b.Wer, &b.Status, &b.Seit); err != nil {
			rows.Close()
			return nil, fmt.Errorf("offene klassensatz-reservierungen konnten nicht gelesen werden: %w", err)
		}
		b.Tabelle = "klassensatz_reservierungen"
		b.Kontext = "Titel gelöscht, eine Klassensatz-Reservierung stand noch offen"
		alle = append(alle, b)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("offene klassensatz-reservierungen konnten nicht gelesen werden: %w", err)
	}

	rows, err = q.Query(ctx, `
		SELECT t.id::text, t.id::text, t.titel, c.class_name
		FROM class_books c
		JOIN buecher_titel t ON t.id = c.book_id
		WHERE c.book_id = ANY($1::uuid[])`, ids)
	if err != nil {
		return nil, fmt.Errorf("klassensatz-zuordnungen konnten nicht gelesen werden: %w", err)
	}
	for rows.Next() {
		var b WartenderBezug
		if err := rows.Scan(&b.ID, &b.TitelID, &b.Titel, &b.Wer); err != nil {
			rows.Close()
			return nil, fmt.Errorf("klassensatz-zuordnungen konnten nicht gelesen werden: %w", err)
		}
		b.Tabelle = "class_books"
		b.Kontext = "Titel gelöscht, er gehörte zu einem Klassensatz"
		alle = append(alle, b)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("klassensatz-zuordnungen konnten nicht gelesen werden: %w", err)
	}
	return alle, nil
}

// ProtokolliereWartendeBezuege hält jede dieser Zeilen fest, bevor der CASCADE sie
// nimmt — in derselben Transaktion wie die Löschung: entweder beides oder keins.
func ProtokolliereWartendeBezuege(ctx context.Context, tx pgx.Tx, bezuege []WartenderBezug) error {
	for _, b := range bezuege {
		inhalt := map[string]any{
			"titel":       b.Titel,
			"betrifft":    b.Wer,
			"erstellt_am": b.Seit,
			"action":      "titel_geloescht_mit_offenem_bezug",
			"tabelle":     b.Tabelle,
		}
		if b.Status != "" {
			inhalt["status"] = b.Status
		}
		if b.SchuelerD != nil {
			inhalt["schueler_id"] = *b.SchuelerD
		}
		details, err := json.Marshal(inhalt)
		if err != nil {
			return fmt.Errorf("protokoll des offenen bezugs: %w", err)
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO audit_log (tabelle, aktion, datensatz_id, akteur, kontext, details)
			VALUES ($1, 'DELETE', $2, 'SYSTEM', $3, $4::jsonb)`,
			b.Tabelle, b.TitelID, b.Kontext, string(details)); err != nil {
			return fmt.Errorf("protokoll des offenen bezugs: %w", err)
		}
	}
	return nil
}
