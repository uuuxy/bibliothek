package repository

import (
	"time"

	"bibliothek/db"
	"bibliothek/pkg/schulzeit"
)

// UeberfaelligesMedium ist ein ausgeliehenes Exemplar, dessen Frist abgelaufen ist.
type UeberfaelligesMedium struct {
	// AusleiheID ist die UUID des Ausleihvorgangs.
	AusleiheID string `json:"ausleihe_id"`
	// Titel ist der Haupttitel des überfälligen Mediums.
	Titel string `json:"titel"`
	// Autor ist der Autor des Werks.
	Autor string `json:"autor"`
	// ISBN ist die ISBN des Buchs.
	ISBN string `json:"isbn"`
	// Barcode ist der Exemplar-Barcode (zum Scannen bei der Rückgabe; steht auf den Zetteln).
	Barcode string `json:"barcode"`
	// CoverURL verweist optional auf das Coverbild des Buchs.
	CoverURL string `json:"cover_url,omitempty"`
	// FaelligAm ist der Tag der Frist als Text (TT.MM.JJJJ).
	FaelligAm string `json:"faellig_am"`
	// TageUeberfaellig speichert die Anzahl der Tage, die das Medium bereits überfällig ist.
	TageUeberfaellig int `json:"tage_ueberfaellig"`
	// Mahnstufe zählt die gedruckten Mahnbriefe zu dieser Ausleihe.
	Mahnstufe int `json:"mahnstufe"`
	// LetztesMahndatum ist der Kalendertag der Schule, an dem zuletzt gemahnt wurde
	// (JJJJ-MM-TT). Leer ohne Mahnung und bei einer aus Littera übernommenen Mahnstufe.
	LetztesMahndatum string `json:"letztes_mahndatum,omitempty"`
}

// mahndatumText nennt den Kalendertag der Schule zu einem Mahndatum, leer ohne Datum.
func mahndatumText(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.In(schulzeit.Zone()).Format("2006-01-02")
}

// UeberfaelligerSchueler fasst alle überfälligen Medien eines konkreten Schülers zusammen.
type UeberfaelligerSchueler struct {
	// SchuelerID ist die UUID des betroffenen Schülers.
	SchuelerID string `json:"schueler_id"`
	// Name ist der vollständige Name des Schülers.
	Name string `json:"name"`
	// Klasse ist die Klasse des Schülers; die Mahnliste nennt bei einem Ehemaligen
	// stattdessen den Namen seiner Gruppe.
	Klasse string `json:"klasse"`
	// Medien listet alle überfälligen Buchexemplare auf, die auf diesen Schüler entfallen.
	Medien []UeberfaelligesMedium `json:"medien"`
}

// GruppeEhemalige ist der Name, unter dem die Mahnliste führt, wer die Schule verlassen hat.
const GruppeEhemalige = "Ehemalige"

// MahnwesenKlasse gruppiert überfällige Schüler und Ausleihen nach ihren Schulklassen für die Mahnwesen-Übersicht.
type MahnwesenKlasse struct {
	// Klasse ist das Klassenkürzel (z. B. "09A").
	Klasse string `json:"klasse"`
	// Ehemalige ist gesetzt, wenn die Gruppe keine Klasse ist, sondern die Ehemaligen
	// sammelt. Sie hat keine Klassenleitung und geht in keinen Versand an eine.
	Ehemalige bool `json:"ehemalige"`
	// LehrerEmail ist die E-Mail-Adresse der Klassenleitung (für automatische Benachrichtigungen).
	LehrerEmail string `json:"lehrer_email"`
	// Schueler enthält die Liste aller Schüler dieser Klasse mit überfälligen Büchern.
	Schueler []UeberfaelligerSchueler `json:"schueler"`
}

// MahnwesenAbfragen liest die überfälligen Ausleihen für Mahnliste und Mahnbriefe und erhöht
// die Mahnstufe.
type MahnwesenAbfragen struct {
	db db.Pool
}

// NewMahnwesenAbfragen liefert die Abfragen des Mahnwesens an Postgres.
func NewMahnwesenAbfragen(pool db.Pool) *MahnwesenAbfragen {
	return &MahnwesenAbfragen{db: pool}
}
