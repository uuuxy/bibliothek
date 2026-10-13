package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

// Titel repräsentiert die beschreibenden Metadaten eines Buchtitels oder Werks (Tabelle `buecher_titel`).
type Titel struct {
	// ID ist die UUID des Buchtitels.
	ID string `json:"id"`
	// Titel ist der Haupttitel des Werks.
	Titel string `json:"titel"`
	// Untertitel enthält optionale Zusatzangaben zum Titel.
	Untertitel string `json:"untertitel,omitempty"`
	// Autor ist der Name des Autors oder der Autoren.
	Autor string `json:"autor,omitempty"`
	// ISBN ist die Internationale Standardbuchnummer (ISBN-10 oder ISBN-13).
	ISBN string `json:"isbn,omitempty"`
	// Verlag ist der herausgebende Buchverlag.
	Verlag string `json:"verlag,omitempty"`
	// Erscheinungsjahr ist das Publikationsjahr.
	Erscheinungsjahr int `json:"erscheinungsjahr,omitempty"`
	// Signatur speichert die Bibliothekssignatur (z. B. Standort/Regal).
	Signatur string `json:"signatur,omitempty"`
	// Auflage ist die Auflagenbezeichnung („4. Aufl. 2023"). Eine neue Auflage trägt eine
	// eigene ISBN und ist damit ein eigener Titel; dieses Feld sagt, welche von zwei gleich
	// heißenden Zeilen man vor sich hat.
	Auflage string `json:"auflage,omitempty"`
	// Mehrjahresband: Das Buch bleibt über die Spanne JahrgangVon..JahrgangBis beim Kind; die
	// Frist rechnet bis zum Stichtag des Schuljahres, in dem das Kind JahrgangBis beendet. Die
	// Jahreszahl ist JahrgangBis, eine zweite gibt es nicht.
	Mehrjahresband bool `json:"mehrjahresband"`
	// IstLernmittel: Schulbuch der Lernmittelfreiheit. Die Regeln —
	// Frist, Limit, öffentlicher Katalog, Löschfrist — lesen dieses Feld, nicht den Text.
	IstLernmittel bool `json:"ist_lernmittel"`
	// Fach, JahrgangVon, JahrgangBis kommen nur aus dem Katalogisat-Import (Zerlegung
	// der Littera-Signatur „LMF Bio 7", Schlagwörter, Zielgruppe) und gehen nur in den
	// Upsert; kein Leser füllt sie. 0 bzw. "" heißt „unbekannt, Bestand nicht anfassen".
	Fach        string `json:"-"`
	JahrgangVon int    `json:"-"`
	JahrgangBis int    `json:"-"`
	// Schlagworte kommen ebenso nur aus dem Katalogisat (MAB 710) und gehen nur in den Upsert:
	// an Titel, die noch keine tragen.
	Schlagworte []string `json:"-"`
	// CoverURL verweist auf das Bild des Buchumschlags.
	CoverURL string `json:"cover_url,omitempty"`
	// Medientyp klassifiziert die Art des Mediums (z. B. "Buch", "CD", "DVD").
	Medientyp string `json:"medientyp,omitempty"`
	// ErstelltAm ist der Erstellungszeitpunkt.
	ErstelltAm time.Time `json:"erstellt_am"`
	// AktualisiertAm ist der letzte Änderungszeitpunkt.
	AktualisiertAm time.Time `json:"aktualisiert_am"`
	// ErweiterteEigenschaften speichert zusätzliche dynamische Metadaten als JSON-Map.
	ErweiterteEigenschaften map[string]any `json:"erweiterteEigenschaften,omitempty"`

	// Bestand und Verfuegbar füllen nur die Suchabfragen (FindeTitel, FindeTitelUnscharf): Die
	// Trefferliste sagt damit, wie viele Exemplare ein Titel hat und wie viele davon frei sind.
	//
	// Zeiger, weil „nicht mitgeliefert" und „null Exemplare" zwei verschiedene Dinge sind: Jede
	// andere Abfrage, die einen Titel liefert (Katalog, Import, Bestellwesen), lässt die Felder
	// nil, und die Oberfläche schreibt dann gar nichts statt „0 Exemplare" über einen Titel,
	// dessen Bestand niemand gezählt hat.
	Bestand    *int `json:"bestand,omitempty"`
	Verfuegbar *int `json:"verfuegbar,omitempty"`
	// ImZulauf zählt die bestellten, noch nicht eingetroffenen Exemplare
	// (SQLBestandImZulauf) — gleiche Zeiger-Regel wie Bestand.
	ImZulauf *int `json:"im_zulauf,omitempty"`
}

// leseTitelMitZusatz liest die feste Spaltenliste eines Titels und danach beliebige
// Zusatzspalten, etwa count(*) OVER () für die Gesamtzahl der Treffer. So bleibt die
// Spaltenliste an einer Stelle.
func leseTitelMitZusatz(row Scanner, zusatz ...any) (*Titel, error) {
	var t Titel
	ziele := []any{
		&t.ID, &t.Titel, &t.Untertitel, &t.Autor, &t.ISBN, &t.Verlag, &t.Erscheinungsjahr, &t.CoverURL, &t.Medientyp, &t.Signatur, &t.Auflage, &t.Mehrjahresband, &t.IstLernmittel, &t.ErstelltAm, &t.AktualisiertAm, &t.ErweiterteEigenschaften,
	}
	if err := row.Scan(append(ziele, zusatz...)...); err != nil {
		return nil, err
	}
	return &t, nil
}

// LadeTitel sucht einen Buchtitel anhand seiner UUID, optimiert für Transaktionen.
func (a *pgKatalogAbfragen) LadeTitel(ctx context.Context, tx pgx.Tx, id string) (*Titel, error) {
	query := "SELECT id, coalesce(titel, ''), coalesce(autor, ''), coalesce(isbn, ''), coalesce(verlag, ''), coalesce(signatur, '') FROM buecher_titel WHERE id = $1"

	var t Titel
	err := tx.QueryRow(ctx, query, id).Scan(&t.ID, &t.Titel, &t.Autor, &t.ISBN, &t.Verlag, &t.Signatur)
	if err != nil {
		return nil, err
	}
	return &t, nil
}
