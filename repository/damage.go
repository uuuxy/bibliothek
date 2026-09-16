package repository

import (
	"context"
	"errors"
	"time"

	"bibliothek/db"
)

// SchadensArt ist die Fallgruppe einer Forderung — genau die beiden Kästchen des
// Bescheid-Formulars (chk_schaden_art, Migration 110): Sie entscheidet, welches Kästchen
// der Brief ankreuzt und ob er die Rückgabe verlangt. Ein eigener Typ, damit sie in der
// Parameterliste von ReportDamage nicht stumm mit der Beschreibung vertauscht wird.
type SchadensArt string

const (
	// SchadensArtNichtZurueck heißt: verloren bzw. nicht ordnungsgemäß zurückgegeben.
	SchadensArtNichtZurueck SchadensArt = "nicht_zurueckgegeben"
	// SchadensArtBeschaedigt heißt: so stark beschädigt zurückgegeben, dass es unbrauchbar ist.
	SchadensArtBeschaedigt SchadensArt = "beschaedigt"
)

// Gueltig sagt, ob a einer der beiden Werte ist.
func (a SchadensArt) Gueltig() bool {
	return a == SchadensArtNichtZurueck || a == SchadensArtBeschaedigt
}

// DamageRepository defines operations for managing book damages and related loan actions.
type DamageRepository interface {
	ReportDamage(ctx context.Context, copyID, loanID, schuelerID string, benutzerID string, beschreibung string, art SchadensArt, betrag float64) (string, error)
	// ListSchadensfaelleVonSchueler liefert alle Schadensfälle eines Schülers,
	// neueste zuerst — die Gebühren-Sektion der Schülerakte.
	ListSchadensfaelleVonSchueler(ctx context.Context, schuelerID string) ([]Schadensfall, error)
}

// Schadensfall ist eine Zeile der Gebühren-/Schadensliste eines Schülers.
// Titel/Barcode sind Zeiger: exemplar_id ist nullbar (Geräteschäden, anonymisierte
// Fälle) — ein nicht-nullbarer String würde beim Scan mit 500 abstürzen.
type Schadensfall struct {
	ID                string     `json:"id"`
	Beschreibung      string     `json:"beschreibung"`
	Betrag            float64    `json:"betrag"`
	IstBezahlt        bool       `json:"ist_bezahlt"`
	ErstelltAm        time.Time  `json:"erstellt_am"`
	StorniertAm       *time.Time `json:"storniert_am"`
	Stornierungsgrund *string    `json:"stornierungsgrund"`
	Titel             *string    `json:"titel"`
	BarcodeID         *string    `json:"barcode_id"`
	// BescheidID: steht die Forderung schon auf einem Schadensersatz-Bescheid? Die
	// Akte bietet „Bescheid erstellen" nur an, solange eine offene Forderung ohne
	// Brief da ist.
	BescheidID *string `json:"bescheid_id"`
}

type pgDamageRepository struct {
	db db.PgxPoolIface
}

// NewDamageRepository returns a new PostgreSQL implementation of DamageRepository.
func NewDamageRepository(db db.PgxPoolIface) DamageRepository {
	return &pgDamageRepository{db: db}
}

// ListSchadensfaelleVonSchueler liefert alle Schadensfälle eines Schülers, neueste zuerst.
// LEFT JOIN: exemplar_id kann NULL sein (Geräteschäden), dann bleiben Titel/Barcode leer.
func (r *pgDamageRepository) ListSchadensfaelleVonSchueler(ctx context.Context, schuelerID string) ([]Schadensfall, error) {
	rows, err := r.db.Query(ctx, `
		SELECT sf.id, sf.beschreibung, sf.betrag, sf.ist_bezahlt, sf.erstellt_am,
		       sf.storniert_am, sf.stornierungsgrund, t.titel, e.barcode_id, sf.bescheid_id
		FROM schadensfaelle sf
		LEFT JOIN buecher_exemplare e ON sf.exemplar_id = e.id
		LEFT JOIN buecher_titel t ON e.titel_id = t.id
		WHERE sf.schueler_id = $1
		ORDER BY sf.erstellt_am DESC
	`, schuelerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	faelle := []Schadensfall{}
	for rows.Next() {
		var f Schadensfall
		if err := rows.Scan(&f.ID, &f.Beschreibung, &f.Betrag, &f.IstBezahlt, &f.ErstelltAm,
			&f.StorniertAm, &f.Stornierungsgrund, &f.Titel, &f.BarcodeID, &f.BescheidID); err != nil {
			return nil, err
		}
		faelle = append(faelle, f)
	}
	return faelle, rows.Err()
}

// ErrExemplarNeuVerliehen signalisiert, dass das zu meldende Exemplar zwischenzeitlich
// (nach dem Öffnen des Schadensformulars) an jemand anderen ausgeliehen wurde. Der Text
// ist nutzer-sichtbar (wird als 409-Meldung ausgeliefert), daher deutsche Großschreibung.
//
//nolint:staticcheck // ST1005: bewusst großgeschrieben, Endnutzer-Meldung
var ErrExemplarNeuVerliehen = errors.New("Exemplar wurde zwischenzeitlich neu ausgeliehen — bitte den Vorgang neu laden")

// ReportDamage bucht einen Verlust oder Schaden in eigener Transaktion — der Weg aus der
// Schülerakte („Verlust/Schaden melden"). Der Rumpf ist meldeSchaden; der Bescheid ruft
// ihn in seiner eigenen Transaktion (Stufe 2 des Mahnverfahrens). schuelerID ist nur
// Anzeige: Der Schuldner steht an der Ausleihe.
func (r *pgDamageRepository) ReportDamage(ctx context.Context, copyID, loanID, _ string, benutzerID string, beschreibung string, art SchadensArt, betrag float64) (string, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer db.SafeRollback(ctx, tx)

	schadensID, err := meldeSchaden(ctx, tx, copyID, loanID, benutzerID, beschreibung, art, betrag)
	if err != nil {
		return "", err
	}
	if err := tx.Commit(ctx); err != nil {
		return "", err
	}
	return schadensID, nil
}
