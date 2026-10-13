// Package repository stellt den Datenzugriff und die Tabellenmodelle der PostgreSQL-Datenbank bereit.
package repository

import (
	"time"
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

// Geraet repräsentiert ein physisches Hardware-Gerät (z. B. Laptop, iPad) der Schule (Tabelle `geraete`).
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
