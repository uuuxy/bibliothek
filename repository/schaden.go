package repository

import (
	"context"
	"errors"
	"time"

	"bibliothek/db"
)

// SchadensArt ist die Fallgruppe einer Forderung, eines der beiden Kästchen im Formular des
// Bescheids (chk_schaden_art): Sie entscheidet, welches Kästchen der Brief ankreuzt und ob er
// die Rückgabe verlangt. Ein eigener Typ, damit sie in der Parameterliste von Melde nicht mit
// der Beschreibung vertauscht wird.
type SchadensArt string

const (
	// SchadensArtNichtZurueck heißt: verloren oder nicht ordnungsgemäß zurückgegeben.
	SchadensArtNichtZurueck SchadensArt = "nicht_zurueckgegeben"
	// SchadensArtBeschaedigt heißt: so stark beschädigt zurückgegeben, dass es unbrauchbar ist.
	SchadensArtBeschaedigt SchadensArt = "beschaedigt"
)

// IstGueltig meldet, ob a einer der beiden Werte ist.
func (a SchadensArt) IstGueltig() bool {
	return a == SchadensArtNichtZurueck || a == SchadensArtBeschaedigt
}

// SchadensAbfragen bucht einen Verlust oder Schaden an einer Ausleihe und liest die
// Schadensfälle eines Lesers.
type SchadensAbfragen interface {
	// Melde bucht einen Verlust oder Schaden an einer Ausleihe. Einen Schuldner nimmt
	// die Tür nicht entgegen: Er steht an der Ausleihe.
	Melde(ctx context.Context, exemplarID, ausleiheID, bearbeiterID, beschreibung string, art SchadensArt, betrag float64) (string, error)
	// LadeZuLeser liefert alle Schadensfälle eines Lesers, die neuesten zuerst, für die Liste
	// der Forderungen in der Akte.
	LadeZuLeser(ctx context.Context, leserID string) ([]Schadensfall, error)
}

// Schadensfall ist eine Zeile der Liste der Forderungen in der Akte eines Lesers. Titel und
// BarcodeID sind Zeiger, weil exemplar_id bei einem Geräteschaden leer ist und sich NULL nicht
// in einen Text lesen lässt.
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
	// BescheidID nennt den Schadensersatz-Bescheid, auf dem die Forderung steht. Die Akte
	// bietet „Bescheid erstellen" nur an, solange eine offene Forderung ohne Brief da ist.
	BescheidID *string `json:"bescheid_id"`
}

type pgSchadensAbfragen struct {
	db db.Pool
}

// NewSchadensAbfragen liefert die Abfragen der Schadensfälle an Postgres.
func NewSchadensAbfragen(db db.Pool) SchadensAbfragen {
	return &pgSchadensAbfragen{db: db}
}

// LadeZuLeser liefert alle Schadensfälle eines Lesers, die neuesten zuerst. Mit LEFT JOIN, weil
// exemplar_id bei einem Geräteschaden leer ist; Titel und Barcode bleiben dann leer.
func (a *pgSchadensAbfragen) LadeZuLeser(ctx context.Context, leserID string) ([]Schadensfall, error) {
	rows, err := a.db.Query(ctx, `
		SELECT sf.id, sf.beschreibung, sf.betrag, sf.ist_bezahlt, sf.erstellt_am,
		       sf.storniert_am, sf.stornierungsgrund, t.titel, e.barcode_id, sf.bescheid_id
		FROM schadensfaelle sf
		LEFT JOIN buecher_exemplare e ON sf.exemplar_id = e.id
		LEFT JOIN buecher_titel t ON e.titel_id = t.id
		WHERE sf.schueler_id = $1
		ORDER BY sf.erstellt_am DESC
	`, leserID)
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

// ErrExemplarNeuVerliehen meldet, dass das Exemplar an jemand anderen ausgeliehen wurde,
// während das Formular der Schadensmeldung offen stand. Die Tür antwortet mit 409.
var ErrExemplarNeuVerliehen = errors.New("exemplar inzwischen neu ausgeliehen")

// Melde bucht einen Verlust oder Schaden in eigener Transaktion, für „Verlust/Schaden melden"
// in der Akte. Den Rumpf (meldeSchaden) ruft auch der Bescheid, in seiner Transaktion.
func (a *pgSchadensAbfragen) Melde(ctx context.Context, exemplarID, ausleiheID, bearbeiterID, beschreibung string, art SchadensArt, betrag float64) (string, error) {
	tx, err := a.db.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer db.Verwerfe(ctx, tx)

	schadensID, err := meldeSchaden(ctx, tx, schadensMeldung{
		exemplarID:   exemplarID,
		ausleiheID:   ausleiheID,
		bearbeiterID: bearbeiterID,
		beschreibung: beschreibung,
		art:          art,
		betrag:       betrag,
	})
	if err != nil {
		return "", err
	}
	if err := tx.Commit(ctx); err != nil {
		return "", err
	}
	return schadensID, nil
}
