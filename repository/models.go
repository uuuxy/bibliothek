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

// Student repräsentiert einen LESER in der Datenbank (Tabelle `leser`).
//
// Der Typ heißt weiter Student, weil die allermeisten Leser Schüler sind und die Listen,
// Mahnungen und Klassenansichten genau die meinen; sie lesen die Sicht `schueler`. Die
// Theke dagegen fragt nach einem LESER (GetLeserByBarcode, GetLeserByID) und bekommt auch
// Kollegen — unterscheidbar an `Art`.
type Student struct {
	// ID ist der eindeutige Primärschlüssel (UUID) des Lesers.
	ID string `json:"id"`
	// Art: chk_leser_art (Migration 123 und 153). Sie entscheidet keine Rechte,
	// sondern nur, wen der LUSD-Abgleich und der Löschjob erfassen — und was die Theke
	// am Treffer anzeigt. Leer, wo die Abfrage die Sicht `schueler` liest: dort ist
	// jede Zeile ein Schüler.
	Art string `json:"art,omitempty"`
	// BarcodeID ist der eindeutige Barcode des Ausweises.
	BarcodeID string `json:"barcode_id"`
	// Vorname ist der Vorname des Schülers.
	Vorname string `json:"vorname"`
	// Nachname ist der Nachname des Schülers.
	Nachname string `json:"nachname"`
	// Klasse ist die aktuelle Klasse bzw. Kursstufe des Schülers (z. B. "07B").
	Klasse string `json:"klasse"`
	// AbgaengerJahr ist das Jahr, in dem der Schüler voraussichtlich die Schule verlässt.
	// Steuert die DSGVO-Löschung und die Stapel-Archivierung — NICHT die Gültigkeit des
	// Ausweises. Beide Zahlen fallen auseinander: Ein Gymnasialschüler bleibt bis
	// Jahrgang 13, sein Ausweis gilt aber nur bis zum Ende der Mittelstufe.
	AbgaengerJahr int `json:"abgaenger_jahr"`
	// AusweisGueltigBis ist das Jahr, bis zu dem der Schülerausweis gilt (31.07.), aus
	// dem Bildungsgang der Klasse gerechnet (internal/ausweis). Kein Datenbankfeld,
	// sondern beim Lesen gesetzt. nil heißt: aus dieser Klassenbezeichnung nicht
	// ableitbar — dann fragt der Druckdialog nach, statt ein Datum zu erfinden.
	AusweisGueltigBis *int `json:"ausweis_gueltig_bis,omitempty"`
	// IstGesperrt ist die Sperre, die das Programm setzt: Ehemalige (Versetzung, LUSD-Import),
	// Papierkorb, Anonymisierung. Die Theke lässt dann nur die Rückgabe zu, außer beim
	// Lernmittel (service.pruefeAusleihSperren). Offene Forderungen und überfällige Medien
	// setzen sie NICHT — die zählt die Theke bei jeder Ausleihe selbst.
	IstGesperrt bool `json:"ist_gesperrt"`
	// LusdID ist die Schüler-ID aus dem hessischen LUSD-System für automatisierte Abgleiche.
	LusdID *string `json:"lusd_id,omitempty"`
	// IstAbgaenger markiert Schüler, die die Schule bereits verlassen haben.
	IstAbgaenger bool `json:"ist_abgaenger"`
	// Geburtsdatum speichert das Geburtsdatum (Format: DATE in PostgreSQL, als String im Code).
	Geburtsdatum *string `json:"geburtsdatum,omitempty"`
	// ErstelltAm ist der Erstellungszeitpunkt des Schülerdatensatzes.
	ErstelltAm time.Time `json:"erstellt_am"`
	// AktualisiertAm ist der Zeitpunkt der letzten Aktualisierung.
	AktualisiertAm time.Time `json:"aktualisiert_am"`
	// IsManuallyBlocked ist die Sperre von Hand (PATCH /api/admin/students/{id}/lock). Sie gilt
	// auch beim Lernmittel; aufgehoben wird sie in der Akte.
	IsManuallyBlocked bool `json:"is_manually_blocked"`
	// BlockReason ist der Grund der Sperre — von Hand oder vom Programm, eine Spalte für beide.
	BlockReason *string `json:"block_reason,omitempty"`
	// Strasse ist der Straßenname der Postanschrift (optionale Stammdaten).
	Strasse string `json:"strasse"`
	// Hausnummer ergänzt die Straße der Postanschrift.
	Hausnummer string `json:"hausnummer"`
	// Plz ist die Postleitzahl der Postanschrift.
	Plz string `json:"plz"`
	// Ort ist der Wohnort der Postanschrift.
	Ort string `json:"ort"`
	// ElternEmail ist die Kontakt-E-Mail der Erziehungsberechtigten.
	ElternEmail string `json:"eltern_email"`
	// IstAnonymisiert: anonymized_at ist gesetzt, die Zeile ist keine Person mehr. Die Theke
	// gibt ihr nichts aus, auch kein Lernmittel (service.pruefeAusleihSperren) — die Namenssuche
	// findet sie noch („Abgänger Anonymisiert-…"). Gefüllt nur über die Spaltenliste der
	// Leser-Abfragen (repository.spaltenLeser); nicht in der API.
	IstAnonymisiert bool `json:"-"`
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
