package service

import (
	"time"

	"bibliothek/db"
	"bibliothek/pkg/schulzeit"
	"bibliothek/repository"
)

// AusleihErgebnis ist das Ergebnis einer Ausleihe oder Rückgabe eines Buchs.
type AusleihErgebnis struct {
	// Art ist "ausleihe" oder "rueckgabe".
	Art string
	// Exemplar ist das gebuchte Exemplar.
	Exemplar *repository.Exemplar
	// Leser ist der Leser, dessen Konto sich geändert hat: der Ausleiher, bei einer
	// Fremdrückgabe am Scan der Vorbesitzer.
	Leser *repository.Leser
	// Frist ist die Rückgabefrist einer Ausleihe.
	Frist *time.Time
	// AusleiheID ist die Kennung der gebuchten Ausleihe.
	AusleiheID *string
	// Fremdrueckgabe sagt, ob jemand anderes als der Ausleiher das Buch zurückgebracht hat.
	Fremdrueckgabe bool
	// Vorbesitzer ist bei einer Fremdrückgabe der Leser, der das Buch ausgeliehen hatte.
	Vorbesitzer *repository.Leser
	// HatVormerkung sagt, dass das zurückgegebene Buch jetzt für einen wartenden Schüler
	// bereitliegt.
	HatVormerkung bool
	// VormerkungTitel ist der Titel des vorgemerkten Buchs.
	VormerkungTitel string
	// VorgemerktFuer nennt Name und Klasse des Schülers, für den das Buch bereitliegt.
	VorgemerktFuer string
	// RegalfreigabeBarcode ist gesetzt, wenn für den Schüler ein anderes Exemplar desselben
	// Titels im Abholfach lag und er ein Exemplar aus dem Regal genommen hat. Das Exemplar im
	// Fach muss zurück ins Regal; sonst läge es dort, obwohl es frei ausleihbar ist.
	RegalfreigabeBarcode string
	// AuflagenHinweis ist gesetzt, wenn Kinder derselben Klasse eine andere Auflage des
	// ausgeliehenen Schulbuchs haben. Er ist nur ein Hinweis, die Ausleihe ist gebucht.
	AuflagenHinweis *repository.AuflagenMischung
}

// AusleihDienst bucht Ausleihe und Rückgabe eines Buchs an der Theke, je in einer Transaktion,
// und für NachbuchDienst nach: Der Scan am Netz und das Nachbuchen buchen mit denselben Schranken
// und Schreibern.
type AusleihDienst struct {
	pool              db.Pool
	leserAbfragen     leserErmittlung
	katalogAbfragen   *repository.KatalogAbfragen
	ausleihAbfragen   ausleihPflege
	protokollAbfragen buchungsProtokoll
	// jetzt ist die Uhr der Fristen; nil heißt time.Now. Tests setzen einen festen Tag, um die
	// Frist am Tag vor, am und nach einem Rückgabetermin zu prüfen.
	jetzt func() time.Time
}

// heute liefert den Zeitpunkt der Uhr in der Schulzeitzone.
func (d *AusleihDienst) heute() time.Time {
	if d.jetzt != nil {
		return d.jetzt().In(schulzeit.Zone())
	}
	return time.Now().In(schulzeit.Zone())
}

// NewAusleihDienst baut den Dienst für Ausleihe und Rückgabe.
func NewAusleihDienst(pool db.Pool, leserAbfragen leserErmittlung, katalogAbfragen *repository.KatalogAbfragen, ausleihAbfragen ausleihPflege, protokollAbfragen buchungsProtokoll) *AusleihDienst {
	return &AusleihDienst{
		pool:              pool,
		leserAbfragen:     leserAbfragen,
		katalogAbfragen:   katalogAbfragen,
		ausleihAbfragen:   ausleihAbfragen,
		protokollAbfragen: protokollAbfragen,
	}
}
