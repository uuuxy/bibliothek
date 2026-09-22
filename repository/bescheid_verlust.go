package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"bibliothek/pkg/schulzeit"

	"github.com/jackc/pgx/v5"
)

// Stufe 2 des Mahnverfahrens (15.09.2026, OFFEN.md 4.18 und 5.13): Der Bescheid entsteht
// direkt aus den überfälligen Büchern. Der Dialog zeigt sie vorgewählt, und beim
// Erstellen bucht der Server je gewähltem Buch den Verlust — Ausleihe endet, Exemplar
// VERLUST, Forderung „nicht zurückgegeben" — und dann Nummer und Brief, alles in EINER
// Transaktion. Papier == Datenbank: Es gibt keinen Brief über ein Buch, dessen Ausleihe
// weiterläuft, und keine beendete Ausleihe ohne Brief. Kommt das Buch zurück, storniert
// der Theke-Scan die Forderung (VerbucheRueckkehr).

// UeberfaelligeAusleihe ist ein überfälliges Buch eines Kindes, das noch keine Forderung
// trägt — ein Kandidat für den Brief.
type UeberfaelligeAusleihe struct {
	AusleiheID string
	ExemplarID string
	Titel      string
	ISBN       string
	Kaufpreis  float64
	// Listenpreis und ZustandAbschlag seit Migration 127 — dieselben Größen wie in
	// OffeneForderung, damit beide Wege denselben Betrag vorschlagen.
	Listenpreis     float64
	ZustandAbschlag int
	IstLernmittel   bool
	FaelligSeit     time.Time
	// Dieselben beiden Größen wie bei OffeneForderung, für die Staffel.
	SchuljahreMitAusleihe int
	SchuljahreImBestand   int
}

// BescheidVerlustEingabe ist ein Buch, das mit dem Brief als Verlust gebucht wird.
type BescheidVerlustEingabe struct {
	AusleiheID string
	Betrag     float64
}

// Die Abweisungen sind Bedienfälle (409), keine Serverfehler: Der Dialog stand offen,
// während sich die Lage änderte — neu öffnen genügt. Nutzer-sichtbar, daher ganze Sätze.
var (
	//nolint:staticcheck // ST1005: ganzer Satz — die Meldung steht so vor der Bibliothekskraft.
	ErrAusleiheInzwischenZurueck = errors.New("Ein gewähltes Buch ist inzwischen zurückgegeben — bitte den Dialog neu öffnen.")
	//nolint:staticcheck // ST1005: ganzer Satz.
	ErrAusleiheSchonGemeldet = errors.New("Für ein gewähltes Buch ist schon eine Forderung erfasst — bitte den Dialog neu öffnen.")
	//nolint:staticcheck // ST1005: ganzer Satz.
	ErrAusleiheFremd = errors.New("Ein gewähltes Buch ist nicht an dieses Kind ausgeliehen.")
)

// UeberfaelligeAusleihen liest die überfälligen Bücher eines Kindes ohne offene
// Forderung. Dasselbe Prädikat wie die Mahnliste (rueckgabe_frist < jetzt); was schon
// eine Forderung trägt, steht bei OffeneForderungen.
func (r *pgBescheidRepository) UeberfaelligeAusleihen(ctx context.Context, schuelerID string) ([]UeberfaelligeAusleihe, error) {
	// EIN Zeitpunkt für beide Rechnungen dieser Funktion: Bis zum 17.09.2026 filterte das
	// SQL mit CURRENT_TIMESTAMP (Uhr der Datenbank), während die Staffel das Schuljahr aus
	// schulzeit.Jetzt() (Uhr des Servers) bestimmte. Zwei Uhren, eine Antwort — und am
	// Schuljahreswechsel konnte die eine schon im neuen Jahr sein, während die andere noch
	// im alten zählte: Das Buch stand in der Liste, die Staffel nannte eine Stufe daneben.
	// Der Unterschied ist heute klein (dieselbe Maschine), aber er ist nicht zugesichert.
	jetzt := schulzeit.Jetzt()
	rows, err := r.db.Query(ctx, `
		SELECT a.id, e.id, t.titel, coalesce(t.isbn, ''), coalesce(e.einkaufspreis, 0)::float8,
		       coalesce(t.listenpreis, 0)::float8, coalesce(e.zustand_abwertung_prozent, 0),
		       -- Zugang statt Bestelltag (Migration 129, Begründung an ersatzwert_groessen.go).
		       coalesce(t.ist_lernmittel, false), a.rueckgabe_frist, COALESCE(e.zugang_am, e.erworben_am)
		FROM ausleihen a
		JOIN buecher_exemplare e ON e.id = a.exemplar_id
		JOIN buecher_titel t ON t.id = e.titel_id
		WHERE a.schueler_id = $1
		  AND a.rueckgabe_am IS NULL
		  AND a.rueckgabe_frist < $2
		  -- Dauerleihen bleiben aussen vor: Sie werden nicht überfällig (dieselbe Regel wie
		  -- in der Sperr-Automatik und in der Leserliste), und eine Forderung „wegen
		  -- Überschreitung der Frist" gegen jemanden, der keine hat, wäre unbegründet.
		  -- Ob ein Kollege für ein VERLORENES Buch zahlen soll, ist davon unberührt — das
		  -- ist eine Betriebsfrage (docs/OFFEN.md 5.16 B) und keine Nebenwirkung der Frist.
		  AND a.ist_handapparat = false
		  AND NOT EXISTS (SELECT 1 FROM schadensfaelle f WHERE f.ausleihe_id = a.id AND f.storniert_am IS NULL)
		ORDER BY a.rueckgabe_frist, t.titel`, schuelerID, jetzt)
	if err != nil {
		return nil, fmt.Errorf("überfällige ausleihen lesen: %w", err)
	}
	defer rows.Close()

	heute := schuljahrVon(jetzt)
	out := []UeberfaelligeAusleihe{}
	exemplare := []string{}
	for rows.Next() {
		var a UeberfaelligeAusleihe
		var zugang *time.Time
		if err := rows.Scan(&a.AusleiheID, &a.ExemplarID, &a.Titel, &a.ISBN, &a.Kaufpreis,
			&a.Listenpreis, &a.ZustandAbschlag,
			&a.IstLernmittel, &a.FaelligSeit, &zugang); err != nil {
			return nil, err
		}
		if zugang != nil {
			a.SchuljahreImBestand = heute - schuljahrVon(*zugang)
		}
		out = append(out, a)
		exemplare = append(exemplare, a.ExemplarID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	schuljahre, err := r.schuljahreMitAusleihe(ctx, exemplare)
	if err != nil {
		return nil, err
	}
	for i := range out {
		out[i].SchuljahreMitAusleihe = schuljahre[out[i].ExemplarID]
	}
	return out, nil
}

// bucheVerluste bucht je gewähltem Buch den Verlust in der Transaktion des Briefs und
// liefert die entstandenen Forderungen als Positionen.
//
// Geprüft wird IN der Transaktion, an der gesperrten Ausleihe-Zeile: Sie gehört diesem
// Kind, läuft noch und trägt noch keine Forderung. Der Dialog kann minutenlang offen
// stehen, während das Buch an der Theke zurückkommt oder jemand in der Akte den Verlust
// meldet — dann entstünde ohne diese Prüfung ein Brief über ein Buch, das im Regal
// steht, oder eine zweite Forderung für dasselbe Buch.
func bucheVerluste(ctx context.Context, tx pgx.Tx, e BescheidEingabe, referenznummer string) ([]BescheidPositionEingabe, error) {
	beschreibung := "Nicht zurückgegeben, Schadensersatz-Bescheid " + referenznummer
	positionen := make([]BescheidPositionEingabe, 0, len(e.Verluste))
	for _, v := range e.Verluste {
		var exemplarID string
		var schuelerID *string
		var zurueck, gemeldet bool
		err := tx.QueryRow(ctx, `
			SELECT a.exemplar_id, a.schueler_id, a.rueckgabe_am IS NOT NULL,
			       EXISTS (SELECT 1 FROM schadensfaelle f WHERE f.ausleihe_id = a.id AND f.storniert_am IS NULL)
			FROM ausleihen a WHERE a.id = $1 FOR UPDATE`, v.AusleiheID,
		).Scan(&exemplarID, &schuelerID, &zurueck, &gemeldet)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAusleiheFremd
		}
		if err != nil {
			return nil, fmt.Errorf("ausleihe %s lesen: %w", v.AusleiheID, err)
		}
		switch {
		case schuelerID == nil || *schuelerID != e.SchuelerID:
			return nil, ErrAusleiheFremd
		case gemeldet:
			return nil, ErrAusleiheSchonGemeldet
		case zurueck:
			return nil, ErrAusleiheInzwischenZurueck
		}
		schadensfallID, err := meldeSchaden(ctx, tx, meldeSchadenParams{
			copyID:       exemplarID,
			loanID:       v.AusleiheID,
			benutzerID:   e.ErstelltVon,
			beschreibung: beschreibung,
			art:          SchadensArtNichtZurueck,
			betrag:       v.Betrag,
		})
		if errors.Is(err, ErrExemplarNeuVerliehen) {
			return nil, err // Bedienfall, unverpackt: der Satz steht so vor der Bibliothekskraft
		}
		if err != nil {
			return nil, fmt.Errorf("verlust für ausleihe %s buchen: %w", v.AusleiheID, err)
		}
		positionen = append(positionen, BescheidPositionEingabe{SchadensfallID: schadensfallID, Betrag: v.Betrag})
	}
	return positionen, nil
}
