package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"bibliothek/db"

	"github.com/jackc/pgx/v5"
)

// Konto ist der Zugang einer Person zum Programm: Lehrkraft, Verwaltung oder Helfer (Tabelle
// benutzer).
type Konto struct {
	// ID ist die UUID des Kontos.
	ID string `json:"id"`
	// BarcodeID ist die Ausweisnummer aus der Leserzeile des Kontos.
	BarcodeID string `json:"barcode_id"`
	Vorname   string `json:"vorname"`
	Nachname  string `json:"nachname"`
	// Rolle ist die Rechteklasse des Kontos (Aufzählung benutzer_rolle).
	Rolle string `json:"rolle"`
	// Email ist die Adresse, mit der sich die Person anmeldet.
	Email string `json:"email"`
	// Aktiv sagt, ob sich die Person anmelden darf.
	Aktiv      bool      `json:"aktiv"`
	ErstelltAm time.Time `json:"erstellt_am"`
	// ZugangBeantragtAm setzt die Selbstanmeldung, das Freischalten leert es; nil heißt kein
	// offener Antrag.
	ZugangBeantragtAm *time.Time `json:"zugang_beantragt_am"`
	// LeserID ist die Leserzeile des Kontos: Dort stehen Ausweis und Ausleihen. Jedes Konto
	// hat eine; dafür sorgt ein Trigger und nicht der Rufer.
	LeserID string `json:"leser_id"`
}

// KontoAbfragen liest und schreibt die Konten: die Zugänge von Lehrkräften, Verwaltung und
// Helfern zum Programm (Tabelle benutzer).
type KontoAbfragen interface {
	// LadeAlle lädt alle Konten, sortiert nach Nachname und Vorname.
	LadeAlle(ctx context.Context) ([]Konto, error)

	// IstEmailVergeben meldet, ob eine Adresse schon zu einem Konto gehört. ausserID nimmt
	// das Konto aus, das gerade bearbeitet wird.
	IstEmailVergeben(ctx context.Context, email string, ausserID string) (bool, error)

	// IstAusweisnummerVergeben meldet, ob eine Ausweisnummer schon vergeben ist. ausserID ist
	// die ID des Kontos, das gerade bearbeitet wird; ausgenommen ist seine Leserzeile.
	IstAusweisnummerVergeben(ctx context.Context, ausweisnummer string, ausserID string) (bool, error)

	// HatLeserzeileOhneKonto meldet, ob ein Kollege dieses Namens schon eine Leserzeile ohne
	// Konto hat. Ein neues Konto bekäme vom Trigger eine zweite, leere Zeile; der Weg zur
	// vorhandenen ist die Schul-E-Mail in der Akte.
	HatLeserzeileOhneKonto(ctx context.Context, vorname, nachname string) (bool, error)

	// LegeAn legt ein Konto an und liefert seine ID. Die Leserzeile entsteht dabei von selbst
	// (Trigger trg_benutzer_hat_leserzeile); ausweisnummer nil oder "" lässt sie ohne Nummer.
	LegeAn(ctx context.Context, ausweisnummer *string, vorname, nachname, email, rolle string) (string, error)

	// Aendere ändert die genannten Felder eines Kontos und liefert seinen Stand danach.
	Aendere(ctx context.Context, p KontoAenderung) (KontoStand, error)

	// LadeRolle liefert die gespeicherte Rolle eines Kontos in Großschreibung; ohne Treffer
	// ("", nil). Die Rolle aus dem Rumpf der Anfrage taugt dafür nicht: Wer das Konto eines
	// Admins bearbeitet, schickt selbst mit, welche Rolle daraus werden soll. Ob das Ziel
	// jetzt ein Admin ist und die Bearbeitung damit Admin-Rechte verlangt, steht nur in der
	// Datenbank.
	LadeRolle(ctx context.Context, id string) (string, error)
}

type pgKontoAbfragen struct {
	pool db.PgxPoolIface
}

// NewKontoAbfragen liefert die Abfragen der Konten an Postgres.
func NewKontoAbfragen(pool db.PgxPoolIface) KontoAbfragen {
	return &pgKontoAbfragen{pool: pool}
}

// Wer ausleihen darf, entscheidet die Leserzeile und nicht das Konto: Die Theke sucht einen
// aktiven Leser, kein Konto.

// LadeAlle lädt alle Konten mit der Ausweisnummer ihrer Leserzeile.
func (a *pgKontoAbfragen) LadeAlle(ctx context.Context) ([]Konto, error) {
	query := `
		SELECT b.id, coalesce(l.barcode_id, ''), b.vorname, b.nachname, b.email, b.rolle, b.aktiv,
		       b.erstellt_am, b.zugang_beantragt_am, coalesce(b.leser_id::text, '')
		FROM benutzer b
		LEFT JOIN leser l ON l.id = b.leser_id
		ORDER BY b.nachname, b.vorname
	`
	rows, err := a.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	konten := []Konto{}
	for rows.Next() {
		var k Konto
		err := rows.Scan(&k.ID, &k.BarcodeID, &k.Vorname, &k.Nachname, &k.Email, &k.Rolle, &k.Aktiv, &k.ErstelltAm,
			&k.ZugangBeantragtAm, &k.LeserID)
		if err != nil {
			return nil, err
		}
		konten = append(konten, k)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return konten, nil
}

// IstEmailVergeben vergleicht in der Normalform der Anmeldung, ohne Rücksicht auf Groß- und
// Kleinschreibung (LadeAnmeldeKonto). Bei einem genauen Vergleich gingen zwei Konten durch,
// die sich nur in der Schreibweise unterscheiden, und welches die Anmeldung öffnet, entschiede
// die Reihenfolge in der Tabelle. Der Index uniq_benutzer_email_lower hält dasselbe in der
// Datenbank.
func (a *pgKontoAbfragen) IstEmailVergeben(ctx context.Context, email string, ausserID string) (bool, error) {
	var vergeben bool
	var err error
	if ausserID == "" {
		err = a.pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM benutzer WHERE LOWER(email) = LOWER($1))", email).Scan(&vergeben)
	} else {
		err = a.pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM benutzer WHERE LOWER(email) = LOWER($1) AND id != $2)", email, ausserID).Scan(&vergeben)
	}
	return vergeben, err
}

// IstAusweisnummerVergeben sucht in der Lesertabelle: Dort steht jede Ausweisnummer, und der
// Index uniq_schueler_barcode_active hält sie eindeutig. Gelöschte Leser geben ihre Nummer
// frei.
//
// ausserID ist die ID des Kontos; ausgenommen wird seine Leserzeile. Der Rufer muss die ID
// des Lesers dafür nicht kennen.
func (a *pgKontoAbfragen) IstAusweisnummerVergeben(ctx context.Context, ausweisnummer string, ausserID string) (bool, error) {
	var vergeben bool
	err := a.pool.QueryRow(ctx, `
		SELECT EXISTS(
		    SELECT 1 FROM leser
		     WHERE barcode_id = $1
		       AND deleted_at IS NULL
		       AND id IS DISTINCT FROM (SELECT leser_id FROM benutzer WHERE id = NULLIF($2, '')::uuid))`,
		ausweisnummer, ausserID).Scan(&vergeben)
	return vergeben, err
}

// HatLeserzeileOhneKonto meldet, ob für den Namen schon ein Kollege ohne Konto in der
// Lesertabelle steht. So bleibt seine Zeile nach dem Löschen des Kontos zurück, weil die
// Ausleihen daran hängen, und so kommt sie aus der Übernahme aus Littera.
//
// Nur das Kollegium zählt (ein Schüler gleichen Namens ist eine andere Person) und nur Zeilen
// ohne Konto: Hat der Namensvetter schon eines, ist der Neue ein zweiter Mensch. Verglichen
// wird in suchnorm wie bei der Namensdublette beim Anlegen eines Lesers.
func (a *pgKontoAbfragen) HatLeserzeileOhneKonto(ctx context.Context, vorname, nachname string) (bool, error) {
	var vorhanden bool
	err := a.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM leser l
			WHERE l.art <> 'schueler' AND l.deleted_at IS NULL
			  AND suchnorm(l.vorname) = suchnorm($1) AND suchnorm(l.nachname) = suchnorm($2)
			  AND NOT EXISTS (SELECT 1 FROM benutzer b WHERE b.leser_id = l.id))`,
		vorname, nachname).Scan(&vorhanden)
	return vorhanden, err
}

// LegeAn legt ein Konto an und trägt die Ausweisnummer in seine Leserzeile ein, beides in
// einer Transaktion: Ein Konto ohne Leserzeile fände die Theke nicht, eine Leserzeile ohne
// Konto stünde namenlos in der Lesertabelle. Die Zeile legt der Trigger an
// (trg_benutzer_hat_leserzeile); die Ausweisnummer kennt er nicht.
func (a *pgKontoAbfragen) LegeAn(ctx context.Context, ausweisnummer *string, vorname, nachname, email, rolle string) (string, error) {
	tx, err := a.pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer db.SafeRollback(ctx, tx)

	var kontoID, leserID string
	err = tx.QueryRow(ctx, `
		INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv)
		VALUES ($1, $2, $3, $4::benutzer_rolle, true)
		RETURNING id, leser_id::text
	`, vorname, nachname, email, rolle).Scan(&kontoID, &leserID)
	if err != nil {
		return "", err
	}

	if ausweisnummer != nil && *ausweisnummer != "" {
		tag, err := tx.Exec(ctx, `UPDATE leser SET barcode_id = $1 WHERE id = $2`, *ausweisnummer, leserID)
		if err != nil {
			return "", err
		}
		// Keine Zeile getroffen hieße: Das Konto hat keine Leserzeile. Der Trigger sichert sie
		// zu; ohne diese Prüfung entstünde sonst ein Konto, dessen Ausweisnummer nirgends
		// steht und das die Theke nicht findet.
		if tag.RowsAffected() == 0 {
			return "", fmt.Errorf("die Leserzeile des neuen Kontos fehlt (id %s)", leserID)
		}
	}
	return kontoID, tx.Commit(ctx)
}

// KontoAenderung nennt, was an einem Konto geändert wird. Ein Feld ohne Wert (nil) ist nicht
// genannt und bleibt, wie es ist: Was ein anderer Platz inzwischen daran gespeichert hat,
// überschreibt die Änderung nicht.
type KontoAenderung struct {
	ID       string
	Vorname  *string
	Nachname *string
	Email    *string
	Rolle    *string
	Aktiv    *bool
	// BarcodeGenannt: Die Änderung nennt die Ausweisnummer. Barcode nil oder leer heißt dann
	// „kein Ausweis"; ohne BarcodeGenannt bleibt die Nummer der Leserzeile stehen.
	BarcodeGenannt bool
	Barcode        *string
}

// KontoStand ist ein Konto nach dem Ändern: was der Protokolleintrag darüber festhält.
type KontoStand struct {
	Email string
	Rolle string
	Aktiv bool
}

// ErrKontoNichtGefunden meldet eine unbekannte ID beim Ändern oder Löschen. Keine getroffene
// Zeile ist hier ein Fehler: Die Tür schriebe sonst einen Protokolleintrag über eine Änderung,
// die es nicht gab.
var ErrKontoNichtGefunden = errors.New("benutzer nicht gefunden")

// Aendere ändert ein Konto und führt die Leserzeile mit, in einer Transaktion.
//
// Name und Ausweis stehen für die Theke in der Leserzeile, für die Anmeldung am Konto. Liefen
// sie auseinander, hieße dieselbe Person an der Theke anders als in der Verwaltung der Konten,
// und der Ausweis, den jemand hier einträgt, bliebe wirkungslos.
func (a *pgKontoAbfragen) Aendere(ctx context.Context, p KontoAenderung) (KontoStand, error) {
	var stand KontoStand
	tx, err := a.pool.Begin(ctx)
	if err != nil {
		return stand, err
	}
	defer db.SafeRollback(ctx, tx)

	// Ein nicht genanntes Feld kommt als NULL an und behält seinen Wert; leeren lässt sich
	// über diese Tür keine der Spalten.
	var leserID *string
	err = tx.QueryRow(ctx, `
		UPDATE benutzer
		SET vorname = COALESCE($1, vorname), nachname = COALESCE($2, nachname),
		    email = COALESCE($3, email), rolle = COALESCE($4::benutzer_rolle, rolle),
		    aktiv = COALESCE($5, aktiv),
		    -- Freischalten erledigt den Antrag; ein späteres Deaktivieren soll nicht
		    -- wieder wie ein Antrag aussehen.
		    zugang_beantragt_am = CASE WHEN $5 IS TRUE THEN NULL ELSE zugang_beantragt_am END,
		    aktualisiert_am = CURRENT_TIMESTAMP
		WHERE id = $6
		RETURNING leser_id::text, email, rolle::text, aktiv
	`, p.Vorname, p.Nachname, p.Email, p.Rolle, p.Aktiv, p.ID).Scan(&leserID, &stand.Email, &stand.Rolle, &stand.Aktiv)
	if errors.Is(err, pgx.ErrNoRows) {
		return stand, ErrKontoNichtGefunden
	}
	if err != nil {
		return stand, err
	}

	// Die Leserzeile nur anfassen, wenn die Änderung eines ihrer Felder nennt.
	if leserID != nil && (p.Vorname != nil || p.Nachname != nil || p.BarcodeGenannt) {
		tag, err := tx.Exec(ctx, `
			UPDATE leser
			SET vorname = COALESCE($1, vorname), nachname = COALESCE($2, nachname),
			    barcode_id = CASE WHEN $3::boolean THEN NULLIF($4::text, '') ELSE barcode_id END
			WHERE id = $5
		`, p.Vorname, p.Nachname, p.BarcodeGenannt, barcodeText(p.Barcode), *leserID)
		if err != nil {
			return stand, err
		}
		// Keine Zeile getroffen hieße: Der Name ist am Konto geändert und an der Leserzeile
		// nicht, dieselbe Person hieße an der Theke anders als in der Verwaltung.
		if tag.RowsAffected() == 0 {
			return stand, fmt.Errorf("die Leserzeile %s zu Konto %s fehlt", *leserID, p.ID)
		}
	}
	return stand, tx.Commit(ctx)
}

// barcodeText macht aus dem Ausweis, der fehlen darf, einen Text: nil und "" heißen beide
// „kein Ausweis" und werden in der Leserzeile zu NULL. Ein leerer Text wäre dort eine vergebene
// Nummer und verletzte die Eindeutigkeit gegen den nächsten Leser ohne Ausweis.
func barcodeText(b *string) string {
	if b == nil {
		return ""
	}
	return *b
}

// LadeRolle liest die gespeicherte Rolle eines Kontos. rolle::text, weil die Spalte die
// Aufzählung benutzer_rolle ist.
func (a *pgKontoAbfragen) LadeRolle(ctx context.Context, id string) (string, error) {
	var rolle string
	err := a.pool.QueryRow(ctx, `SELECT UPPER(rolle::text) FROM benutzer WHERE id = $1`, id).Scan(&rolle)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("konnte die Rolle des Benutzers nicht lesen: %w", err)
	}
	return rolle, nil
}
