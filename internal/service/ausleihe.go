package service

import (
	"context"
	"time"

	"bibliothek/db"
	"bibliothek/pkg/schulzeit"
	"bibliothek/repository"
)

// AusleihErgebnis beschreibt das Ergebnis einer Ausleih- oder Rückgabeoperation eines Buches.
type AusleihErgebnis struct {
	// Art spezifiziert die Art des Ergebnisses (z. B. "ausleihe", "rueckgabe", "info").
	Art string
	// Exemplar enthält die Daten des betroffenen Buchexemplars.
	Exemplar *repository.Exemplar
	// Leser ist das Schülerprofil, falls die Aktion für einen Schüler durchgeführt wurde.
	Leser *repository.Leser
	// Frist gibt das berechnete Rückgabedatum an (nur bei erfolgreicher Ausleihe).
	Frist *time.Time
	// AusleiheID ist die eindeutige ID des Ausleihvorgangs.
	AusleiheID *string
	// Fremdrueckgabe gibt an, ob das Buch von jemand anderem als dem Entleiher zurückgegeben wurde.
	Fremdrueckgabe bool
	// Vorbesitzer ist der LESER, der das Buch zuvor ausgeliehen hatte (bei Fremdrückgabe).
	// Ein zweites Feld für Lehrkräfte gab es bis Migration 125; es gibt nur noch Leser.
	Vorbesitzer *repository.Leser
	// HatVormerkung ist wahr, wenn für das Buch eine Vormerkung vorliegt und es nun für den nächsten Schüler bereitgestellt wurde.
	HatVormerkung bool
	// VormerkungTitel ist der Titel des vorgemerkten Buchs.
	VormerkungTitel string
	// VorgemerktFuer ist der Name (und ggf. Klasse) des Schülers, für den das Buch reserviert wurde.
	VorgemerktFuer string
	// RegalfreigabeBarcode ist gesetzt, wenn für diesen Schüler ein ANDERES Exemplar
	// desselben Titels im Reservierungsfach lag, er sich aber ein Freihand-Exemplar
	// genommen hat. Das reservierte Exemplar muss zurück ins normale Regal — sonst
	// bleibt es als "Geisterbuch" im Fach liegen, obwohl es laut DB verfügbar ist.
	RegalfreigabeBarcode string
	// AuflagenHinweis ist gesetzt, wenn das ausgeliehene Schulbuch eine Auflage eines Buchs
	// ist, von dem Kinder derselben Klasse gerade eine ANDERE Auflage haben (docs/OFFEN.md
	// 4.18, Stufe 5: „Die Ausgabe warnt, wenn eine Klasse gemischte Auflagen bekommt"). Nur
	// ein Hinweis — die Ausleihe ist gebucht.
	AuflagenHinweis *repository.AuflagenMischung
}

// AusleihDienst steuert die Geschäftsregeln und Transaktionen rund um das Ausleihen und Zurückgeben von Büchern.
type AusleihDienst interface {
	// BucheAmLeser wickelt die Ausleihe eines Buchexemplars an den aktiven Leser ab.
	// Falls das Exemplar bereits von jemand anderem ausgeliehen war, wird dieses zuerst automatisch zurückgegeben
	// (Fremdrückgabe) und danach für den neuen Ausleiher verbucht.
	BucheAmLeser(ctx context.Context, exemplar *repository.Exemplar, aktiverLeserID *string, bearbeiterID string, uebergehen bool) (*AusleihErgebnis, error)

	// NimmZurueck wickelt die direkte Rückgabe eines Buchexemplars ab (ohne dass ein neuer Ausleiher aktiv ist).
	// Ein freies Exemplar ist hier ein Fehler: Ausgeliehen wird über den Ausweis, nicht
	// dadurch, dass ein angemeldetes Konto ein Buch in die Hand nimmt.
	NimmZurueck(ctx context.Context, exemplar *repository.Exemplar, bearbeiterID string) (*AusleihErgebnis, error)
}

// ausleihDienst implementiert den AusleihDienst unter Verwendung von Repositories.
type ausleihDienst struct {
	pool              db.PgxPoolIface
	leserAbfragen     repository.LeserAbfragen
	katalogAbfragen   repository.KatalogAbfragen
	ausleihAbfragen   repository.AusleihAbfragen
	protokollAbfragen repository.ProtokollAbfragen
	// jetzt ist die Uhr der Fristberechnung; nil heißt time.Now. Tests setzen einen festen
	// Tag, um die Frist am Tag vor, am und nach dem Rückgabetermin zu prüfen (Bugklasse
	// „Frist am Tag des Ereignisses", docs/sweeps.md).
	jetzt func() time.Time
}

// heute liefert den Zeitpunkt der Uhr in der Schulzeitzone.
func (d *ausleihDienst) heute() time.Time {
	if d.jetzt != nil {
		return d.jetzt().In(schulzeit.Zone())
	}
	return time.Now().In(schulzeit.Zone())
}

// NewAusleihDienst erzeugt eine neue Instanz des standardmäßigen AusleihDienst.
func NewAusleihDienst(pool db.PgxPoolIface, leserAbfragen repository.LeserAbfragen, katalogAbfragen repository.KatalogAbfragen, ausleihAbfragen repository.AusleihAbfragen, protokollAbfragen repository.ProtokollAbfragen) AusleihDienst {
	return &ausleihDienst{
		pool:              pool,
		leserAbfragen:     leserAbfragen,
		katalogAbfragen:   katalogAbfragen,
		ausleihAbfragen:   ausleihAbfragen,
		protokollAbfragen: protokollAbfragen,
	}
}
