package repository

// Die Verwaltung der Geräte: anlegen, auflisten mit dem, der das Gerät gerade hat, und
// Stammdaten und Status ändern. Die Ausleihe der Geräte steht in geraete_ausleihe.go.

import (
	"context"
	"errors"
	"time"

	"bibliothek/db"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Geraet ist ein Gerät der Schule, etwa ein Laptop oder ein iPad (Tabelle geraete).
type Geraet struct {
	// ID ist die UUID des Geräts.
	ID string `json:"id"`
	// Modellname ist der Name des Gerätemodells (z. B. "Lenovo ThinkPad L13").
	Modellname string `json:"modellname"`
	// Seriennummer ist die Hardware-Hersteller-Seriennummer zur eindeutigen Geräteidentifikation.
	Seriennummer *string `json:"seriennummer,omitempty"`
	// BarcodeID ist der Barcode-Aufkleber auf dem Gehäuse.
	BarcodeID string `json:"barcode_id"`
	// Zubehoer listet mitgelieferte Zubehörteile auf, die bei Ausleihe geprüft werden müssen (z. B. "Ladekabel, Stift").
	Zubehoer string `json:"zubehoer"`
	// IstAusleihbar gibt an, ob das Gerät verliehen werden darf (Falsch bei Defekt).
	IstAusleihbar bool `json:"ist_ausleihbar"`
	// IstAusgesondert kennzeichnet Geräte, die dauerhaft aus dem Bestand entfernt wurden.
	IstAusgesondert bool `json:"ist_ausgesondert"`
	// ZustandNotiz beschreibt Vorschäden oder Gebrauchsspuren.
	ZustandNotiz *string `json:"zustand_notiz,omitempty"`
	// ErstelltAm ist das Datum der Geräteerfassung.
	ErstelltAm time.Time `json:"erstellt_am"`
	// AktualisiertAm ist der letzte Aktualisierungszeitpunkt.
	AktualisiertAm time.Time `json:"aktualisiert_am"`
}

// ErrGeraetBarcodeVergeben meldet, dass der Barcode schon auf einem anderen Gerät klebt. Der
// Text ist die Meldung an der Oberfläche (409) und deshalb als Satz geschrieben.
//
//nolint:staticcheck // ST1005: bewusst großgeschrieben, Endnutzer-Meldung
var ErrGeraetBarcodeVergeben = errors.New("Barcode ist bereits an ein anderes Gerät vergeben")

// ErrGeraetSeriennummerVergeben meldet, dass die Seriennummer schon an einem anderen Gerät steht.
//
//nolint:staticcheck // ST1005: bewusst großgeschrieben, Endnutzer-Meldung
var ErrGeraetSeriennummerVergeben = errors.New("Seriennummer ist bereits an einem anderen Gerät eingetragen")

// geraetEindeutigkeit übersetzt eine verletzte Eindeutigkeit in den Fehler, der das Feld
// nennt. Die Tabelle hat zwei: Barcode und Seriennummer.
func geraetEindeutigkeit(err error) error {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "23505" {
		return err
	}
	if pgErr.ConstraintName == "geraete_seriennummer_key" {
		return ErrGeraetSeriennummerVergeben
	}
	return ErrGeraetBarcodeVergeben
}

// GeraetMitStatus ist eine Zeile der Geräte-Verwaltung: die Stammdaten und dazu, wer das
// Gerät gerade hat (nil heißt: im Schrank).
type GeraetMitStatus struct {
	Geraet
	AusgeliehenAn *string `json:"ausgeliehen_an,omitempty"`
}

// GeraeteAbfragen liest und schreibt die Geräte für ihre Verwaltung.
type GeraeteAbfragen interface {
	LadeAlle(ctx context.Context) ([]GeraetMitStatus, error)
	LegeAn(ctx context.Context, modellname string, seriennummer *string, barcode, zubehoer, zustandNotiz string) (string, error)
	// Aendere ändert Stammdaten und ob das Gerät ausleihbar ist (nicht ausleihbar heißt defekt
	// oder gesperrt).
	Aendere(ctx context.Context, id string, aenderung GeraetAenderung) error
}

// GeraetAenderung nennt, was an einem Gerät geändert wird. Ein Feld ohne Wert (nil) ist nicht
// genannt und bleibt, wie es ist: Was ein anderer Platz inzwischen daran gespeichert hat,
// überschreibt die Änderung nicht. Ein leerer Wert heißt bei Zubehör, Zustandsnotiz und
// Seriennummer „keins".
type GeraetAenderung struct {
	Modellname    *string
	Zubehoer      *string
	ZustandNotiz  *string
	Seriennummer  *string
	IstAusleihbar *bool
}

// IstLeer sagt, ob die Änderung kein Feld nennt.
func (a GeraetAenderung) IstLeer() bool {
	return a.Modellname == nil && a.Zubehoer == nil && a.ZustandNotiz == nil &&
		a.Seriennummer == nil && a.IstAusleihbar == nil
}

type pgGeraeteAbfragen struct {
	db db.Pool
}

// NewGeraeteAbfragen liefert die Abfragen der Geräte am übergebenen Pool.
func NewGeraeteAbfragen(pool db.Pool) GeraeteAbfragen {
	return &pgGeraeteAbfragen{db: pool}
}

// LadeAlle liefert alle Geräte, die nicht ausgesondert sind, das neueste zuerst, mit dem
// Namen dessen, der das Gerät gerade hat; check_loan_borrower lässt höchstens einen zu.
func (a *pgGeraeteAbfragen) LadeAlle(ctx context.Context) ([]GeraetMitStatus, error) {
	rows, err := a.db.Query(ctx, `
		SELECT g.id, g.modellname, g.seriennummer, g.barcode_id, g.zubehoer,
		       g.ist_ausleihbar, g.ist_ausgesondert, g.zustand_notiz,
		       g.erstellt_am, g.aktualisiert_am,
		       btrim(l.vorname || ' ' || l.nachname ||
		             coalesce(' (' || nullif(l.klasse, '') || ')', '')) AS ausgeliehen_an
		FROM geraete g
		LEFT JOIN ausleihen a ON a.geraet_id = g.id AND a.rueckgabe_am IS NULL
		-- leser, nicht die Sicht schueler: Sonst stünde bei einem Gerät in der Hand
		-- eines Kollegen kein Name.
		LEFT JOIN leser l ON l.id = a.schueler_id
		WHERE g.ist_ausgesondert = false
		ORDER BY g.erstellt_am DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	geraete := []GeraetMitStatus{}
	for rows.Next() {
		var g GeraetMitStatus
		if err := rows.Scan(
			&g.ID, &g.Modellname, &g.Seriennummer, &g.BarcodeID, &g.Zubehoer,
			&g.IstAusleihbar, &g.IstAusgesondert, &g.ZustandNotiz,
			&g.ErstelltAm, &g.AktualisiertAm, &g.AusgeliehenAn,
		); err != nil {
			return nil, err
		}
		geraete = append(geraete, g)
	}
	return geraete, rows.Err()
}

// LegeAn legt ein Gerät an und liefert seine ID. Eine leere Seriennummer oder Zustandsnotiz
// wird NULL.
func (a *pgGeraeteAbfragen) LegeAn(ctx context.Context, modellname string, seriennummer *string, barcode, zubehoer, zustandNotiz string) (string, error) {
	var id string
	err := a.db.QueryRow(ctx, `
		INSERT INTO geraete (modellname, seriennummer, barcode_id, zubehoer, zustand_notiz)
		VALUES ($1, NULLIF($2, ''), $3, $4, NULLIF($5, ''))
		RETURNING id
	`, modellname, seriennummer, barcode, zubehoer, zustandNotiz).Scan(&id)
	if err != nil {
		return "", geraetEindeutigkeit(err)
	}
	return id, nil
}

// Aendere schreibt, was die Änderung nennt. Der Dialog zum Bearbeiten nennt das Kennzeichen
// „defekt" nie (es liegt auf einem eigenen Knopf), der Knopf nennt nur das Kennzeichen. Eine
// leere Seriennummer oder Zustandsnotiz wird NULL: Als leerer Text stieße das zweite Gerät
// ohne Seriennummer an die Eindeutigkeit. Nennt die Änderung nichts, wird nichts geschrieben;
// ein unbekanntes Gerät bleibt pgx.ErrNoRows.
func (a *pgGeraeteAbfragen) Aendere(ctx context.Context, id string, aenderung GeraetAenderung) error {
	if aenderung.IstLeer() {
		var vorhanden bool
		if err := a.db.QueryRow(ctx,
			`SELECT EXISTS (SELECT 1 FROM geraete WHERE id = $1 AND ist_ausgesondert = false)`, id).Scan(&vorhanden); err != nil {
			return err
		}
		if !vorhanden {
			return pgx.ErrNoRows
		}
		return nil
	}
	tag, err := a.db.Exec(ctx, `
		UPDATE geraete
		SET modellname = COALESCE($1, modellname), zubehoer = COALESCE($2, zubehoer),
		    zustand_notiz = CASE WHEN $3::text IS NULL THEN zustand_notiz ELSE NULLIF($3, '') END,
		    seriennummer = CASE WHEN $4::text IS NULL THEN seriennummer ELSE NULLIF($4, '') END,
		    ist_ausleihbar = COALESCE($5, ist_ausleihbar),
		    aktualisiert_am = CURRENT_TIMESTAMP
		WHERE id = $6 AND ist_ausgesondert = false
	`, aenderung.Modellname, aenderung.Zubehoer, aenderung.ZustandNotiz, aenderung.Seriennummer, aenderung.IstAusleihbar, id)
	if err != nil {
		return geraetEindeutigkeit(err)
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}
