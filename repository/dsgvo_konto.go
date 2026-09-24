package repository

// dsgvo_konto.go — der Teil der Auskunft nach Art. 15 DSGVO, der am ZUGANGSKONTO hängt: das
// Konto selbst, die Klassenleitungen, die eigenen Anfragen im Kollegiums-Portal, die
// Einträge des Verwaltungsprotokolls über das Konto und die Vorgänge, die die Person selbst
// bearbeitet hat (dsgvo_konto_vorgaenge.go). Bis zum 24.09.2026 endete die Auskunft
// eines Kollegen mit „nicht gefunden" (Stammdaten aus der Sicht `schueler`); entschieden ist
// seither: Die Auskunft gibt es für jeden Leser (OFFEN.md 5.19).
//
// Hier und nicht in api/dsgvo_auskunft.go, weil ein Handler kein neues SQL formuliert
// (api/schichtung_test.go).

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// DsgvoZugangskonto ist das Konto, mit dem sich die Person anmeldet, samt allem, was über
// dieses Konto gespeichert ist. Nil in der Auskunft heißt: Auf diesen Leser zeigt kein Konto.
type DsgvoZugangskonto struct {
	ID                string     `json:"id"`
	Email             string     `json:"email"`
	Vorname           string     `json:"vorname"`
	Nachname          string     `json:"nachname"`
	Rolle             string     `json:"rolle"`
	Aktiv             bool       `json:"aktiv"`
	ZugangBeantragtAm *time.Time `json:"zugang_beantragt_am"`
	ErstelltAm        time.Time  `json:"erstellt_am"`
	AktualisiertAm    time.Time  `json:"aktualisiert_am"`
	// Die Klassen, deren Klassenleitung diese Adresse ist (klassen_lehrer_mapping): Dorthin
	// geht die Liste der überfälligen Medien einer Klasse.
	Klassenleitungen []string             `json:"klassenleitungen"`
	Anfragen         []DsgvoAnfrage       `json:"anfragen"`
	Ereignisse       []DsgvoKontoEreignis `json:"ereignisse_im_verwaltungsprotokoll"`
	// Was die Person mit diesem Konto selbst bearbeitet hat — ohne die Daten Dritter
	// (dsgvo_konto_vorgaenge.go).
	EigeneVorgaenge []DsgvoEigenerVorgang `json:"selbst_bearbeitete_vorgaenge"`
}

// DsgvoAnfrage ist ein Wunsch, eine Meldung oder eine Klassensatz-Reservierung, die die
// Person im Kollegiums-Portal gestellt hat.
type DsgvoAnfrage struct {
	Art           string     `json:"art"` // wunsch | meldung | klassensatz
	Titel         string     `json:"titel"`
	ISBN          string     `json:"isbn"`
	Klasse        string     `json:"klasse"`
	Anzahl        int        `json:"anzahl"` // nur beim Klassensatz, sonst 0
	Kommentar     string     `json:"kommentar"`
	ErstelltAm    time.Time  `json:"erstellt_am"`
	Erledigt      bool       `json:"erledigt"`
	ErledigtAm    *time.Time `json:"erledigt_am"` // beim Klassensatz vor Migration 089 leer
	ErledigtNotiz string     `json:"erledigt_notiz"`
}

// DsgvoKontoEreignis ist ein Eintrag des Verwaltungsprotokolls ÜBER das Konto: angelegt,
// geändert, selbst beantragt. Wer ihn ausgelöst hat (admin_id, IP-Adresse), ist eine andere
// Person und steht deshalb nicht darin.
type DsgvoKontoEreignis struct {
	Aktion    string          `json:"aktion"`
	Zeitpunkt time.Time       `json:"zeitpunkt"`
	Details   json.RawMessage `json:"details" swaggertype:"object"`
}

// DsgvoKontoSQL ist die eine Spaltenliste des Kontos in der Auskunft. Das Spalten-Gate
// (dsgvo_konto_spalten_pg_test.go) hält sie gegen jede Spalte von benutzer.
const DsgvoKontoSQL = `
	SELECT id, email, vorname, nachname, rolle::text, aktiv, zugang_beantragt_am,
	       erstellt_am, aktualisiert_am
	FROM benutzer
	WHERE leser_id = $1`

// dsgvoAnfragenSQL liest beide Tabellen des Portals, in denen die Person als Anfragende
// steht. Der Titel einer Reservierung kommt aus dem Katalog; der eines Wunsches ist der
// Freitext der Lehrkraft.
const dsgvoAnfragenSQL = `
	SELECT art, titel_text, isbn, klasse, 0, kommentar, erstellt_am,
	       erledigt_am IS NOT NULL, erledigt_am, erledigt_notiz
	FROM lehrer_anliegen
	WHERE angefordert_von = $1
	UNION ALL
	SELECT 'klassensatz', t.titel, COALESCE(t.isbn, ''), k.klasse, k.anzahl,
	       COALESCE(k.notiz, ''), k.erstellt_am, k.erledigt, k.erledigt_am, k.erledigt_notiz
	FROM klassensatz_reservierungen k
	JOIN buecher_titel t ON t.id = k.titel_id
	WHERE k.angefordert_von = $1
	ORDER BY 7 DESC`

// dsgvoKontoEreignisseSQL findet die Einträge über das Konto. Eine Änderung trägt das Konto
// als ziel_id, die Selbstanmeldung als admin_id (das neue Konto meldet sich selbst an). Die
// Anlage durch die Verwaltung trug bis zum 24.09.2026 nur die Adresse — deshalb für diese
// Altzeilen der Vergleich über die Adresse, und nur ab der Anlage dieses Kontos: Eine
// frühere Zeile mit derselben Adresse gehörte zu einem anderen, inzwischen gelöschten Konto.
const dsgvoKontoEreignisseSQL = `
	SELECT aktion, zeitstempel, details
	FROM audit_logs
	WHERE details->>'ziel_id' = $1::text
	   OR (aktion = 'SELBSTANMELDUNG' AND admin_id = $1::uuid)
	   OR (aktion = 'USER_CREATE' AND NOT details ? 'ziel_id'
	       AND lower(details->>'email') = lower($2) AND zeitstempel >= $3)
	ORDER BY zeitstempel DESC`

// LeseDsgvoZugangskonto liest das Konto, das auf diesen Leser zeigt, samt allem, was daran
// hängt. Ohne Konto: nil, kein Fehler.
func LeseDsgvoZugangskonto(ctx context.Context, q DBQueryer, leserID string) (*DsgvoZugangskonto, error) {
	var k DsgvoZugangskonto
	err := q.QueryRow(ctx, DsgvoKontoSQL, leserID).Scan(
		&k.ID, &k.Email, &k.Vorname, &k.Nachname, &k.Rolle, &k.Aktiv, &k.ZugangBeantragtAm,
		&k.ErstelltAm, &k.AktualisiertAm)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("zugangskonto: %w", err)
	}

	if k.Klassenleitungen, err = sammle(ctx, q, `
		SELECT klasse FROM klassen_lehrer_mapping
		WHERE lower(lehrer_email) = lower($1)
		ORDER BY klasse`, func(r pgx.Rows) (string, error) {
		var klasse string
		return klasse, r.Scan(&klasse)
	}, k.Email); err != nil {
		return nil, fmt.Errorf("klassenleitungen: %w", err)
	}
	if k.Anfragen, err = sammle(ctx, q, dsgvoAnfragenSQL, func(r pgx.Rows) (DsgvoAnfrage, error) {
		var a DsgvoAnfrage
		return a, r.Scan(&a.Art, &a.Titel, &a.ISBN, &a.Klasse, &a.Anzahl, &a.Kommentar,
			&a.ErstelltAm, &a.Erledigt, &a.ErledigtAm, &a.ErledigtNotiz)
	}, k.ID); err != nil {
		return nil, fmt.Errorf("anfragen: %w", err)
	}
	if k.Ereignisse, err = sammle(ctx, q, dsgvoKontoEreignisseSQL, func(r pgx.Rows) (DsgvoKontoEreignis, error) {
		var e DsgvoKontoEreignis
		return e, r.Scan(&e.Aktion, &e.Zeitpunkt, &e.Details)
	}, k.ID, k.Email, k.ErstelltAm); err != nil {
		return nil, fmt.Errorf("kontoereignisse: %w", err)
	}
	if k.EigeneVorgaenge, err = leseDsgvoEigeneVorgaenge(ctx, q, k.ID); err != nil {
		return nil, err
	}
	return &k, nil
}

// sammle liest alle Zeilen einer Abfrage in eine Liste — leer statt nil, damit die
// abgerufene Auskunft „keine" sagt und nicht „unbekannt".
func sammle[T any](ctx context.Context, q DBQueryer, sql string, lies func(pgx.Rows) (T, error), args ...any) ([]T, error) {
	rows, err := q.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []T{}
	for rows.Next() {
		v, err := lies(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
