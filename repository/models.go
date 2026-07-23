// Package repository stellt den Datenzugriff und die Tabellenmodelle der PostgreSQL-Datenbank bereit.
package repository

import (
	"time"
)

// User repräsentiert einen Systemadministrator, Lehrer oder ein Bibliotheks-Teammitglied.
type User struct {
	// ID ist der eindeutige Primärschlüssel (UUID) des Benutzers.
	ID string `json:"id"`
	// BarcodeID ist die gescannte Kennung auf dem Mitarbeiterausweis.
	BarcodeID string `json:"barcode_id"`
	// Vorname ist der Vorname des Benutzers.
	Vorname string `json:"vorname"`
	// Nachname ist der Nachname des Benutzers.
	Nachname string `json:"nachname"`
	// Rolle definiert die Rechteklasse (z. B. "ADMIN", "LEHRER", "HELFER").
	Rolle string `json:"rolle"`
	// Email ist die E-Mail-Adresse für Systembenachrichtigungen.
	Email string `json:"email"`
	// Aktiv zeigt an, ob das Benutzerkonto aktiv ist und sich anmelden darf.
	Aktiv bool `json:"aktiv"`
	// ErstelltAm ist der Zeitstempel der Benutzerregistrierung.
	ErstelltAm time.Time `json:"erstellt_am"`
	// Permissions enthält eine Liste spezifischer Berechtigungsschlüssel des Benutzers.
	Permissions []string `json:"permissions"`
}

// Student repräsentiert einen Schüler in der Datenbank (Tabelle `schueler`).
type Student struct {
	// ID ist der eindeutige Primärschlüssel (UUID) des Schülers.
	ID string `json:"id"`
	// BarcodeID ist der eindeutige Barcode des Schülerausweises.
	BarcodeID string `json:"barcode_id"`
	// Vorname ist der Vorname des Schülers.
	Vorname string `json:"vorname"`
	// Nachname ist der Nachname des Schülers.
	Nachname string `json:"nachname"`
	// Klasse ist die aktuelle Klasse bzw. Kursstufe des Schülers (z. B. "07B").
	Klasse string `json:"klasse"`
	// AbgaengerJahr ist das Jahr, in dem der Schüler voraussichtlich die Schule verlässt.
	AbgaengerJahr int `json:"abgaenger_jahr"`
	// IstGesperrt sperrt die Ausleihberechtigung des Schülers bei Verlusten oder offenen Gebühren.
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
	// IsManuallyBlocked zeigt an, ob der Schüler manuell (Hard-Block) gesperrt wurde.
	IsManuallyBlocked bool `json:"is_manually_blocked"`
	// BlockReason enthält die Begründung für die manuelle Sperre.
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
}

// BookTitle repräsentiert die beschreibenden Metadaten eines Buchtitels oder Werks (Tabelle `buecher_titel`).
type BookTitle struct {
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
	// ZielJahrgang definiert, bis zu welcher Klasse ein Exemplar dieses Titels bei Schülern bleibt (Default 0 = 1 Jahr).
	ZielJahrgang int `json:"ziel_jahrgang"`
	// Beschreibung enthält eine Inhaltsangabe oder Notizen zum Buch.
	Beschreibung string `json:"beschreibung,omitempty"`
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
}

// BookCopy repräsentiert ein physisches Einzelexemplar eines Buchs (Tabelle `buecher_exemplare`).
// Dieses Struct enthält zur Vereinfachung direkt gejointe Daten aus dem zugehörigen Buchtitel.
type BookCopy struct {
	// ID ist die UUID des konkreten Exemplars.
	ID string `json:"id"`
	// TitelID verweist auf die Metadaten des Buchtitels.
	TitelID string `json:"titel_id"`
	// BarcodeID ist die physische Inventar- oder Barcode-Nummer des Exemplars.
	BarcodeID string `json:"barcode_id"`
	// ZustandNotiz dokumentiert eventuelle Beschädigungen (z. B. "Wasserschaden") oder Reservierungen.
	ZustandNotiz string `json:"zustand_notiz"`
	// ErworbenAm ist das Kaufdatum oder Zugangsdatum des Exemplars.
	ErworbenAm time.Time `json:"erworben_am"`
	// IstAusleihbar gibt an, ob das Buch verliehen werden darf.
	IstAusleihbar bool `json:"ist_ausleihbar"`
	// IstAusgesondert markiert verloren gegangene, beschädigte oder ausgemusterte Bücher.
	IstAusgesondert bool `json:"ist_ausgesondert"`
	// ErstelltAm ist das System-Erfassungsdatum.
	ErstelltAm time.Time `json:"erstellt_am"`
	// AktualisiertAm ist das letzte Änderungsdatum.
	AktualisiertAm time.Time `json:"aktualisiert_am"`

	// Gejointe Felder aus der Tabelle buecher_titel:

	// Titel ist der Haupttitel des Werks.
	Titel string `json:"titel"`
	// Autor ist der Autor des Werks.
	Autor string `json:"autor"`
	// Verlag ist der Verlag des Werks.
	Verlag string `json:"verlag"`
	// ISBN ist die ISBN des Werks.
	ISBN string `json:"isbn"`
	// CoverURL ist das Coverbild des Werks.
	CoverURL string `json:"cover_url,omitempty"`
	// Medientyp ist die Medienart (z. B. "Buch").
	Medientyp string `json:"medientyp,omitempty"`
	// Signatur speichert die Bibliothekssignatur.
	Signatur string `json:"signatur,omitempty"`
	// ZielJahrgang definiert die Zielklasse für die Fristberechnung.
	ZielJahrgang int `json:"ziel_jahrgang"`
	// ErweiterteEigenschaften speichert zusätzliche dynamische Metadaten als JSON-Map.
	ErweiterteEigenschaften map[string]any `json:"erweiterteEigenschaften,omitempty"`
}

// Loan repräsentiert einen aktiven oder historischen Ausleiheintrag in der Datenbank (Tabelle `ausleihen`).
type Loan struct {
	// ID ist die eindeutige ID (UUID) des Ausleihvorgangs.
	ID string `json:"id"`
	// ExemplarID verweist auf das ausgeliehene Buch (null bei Geräteausleihe).
	ExemplarID *string `json:"exemplar_id,omitempty"`
	// GeraetID verweist auf das ausgeliehene Hardware-Gerät (null bei Buchausleihe).
	GeraetID *string `json:"geraet_id,omitempty"`
	// SchuelerID verweist auf den ausleihenden Schüler (null bei Ausleihe an Lehrkraft).
	SchuelerID *string `json:"schueler_id,omitempty"`
	// AusleiherBenutzerID verweist auf die ausleihende Lehrkraft (null bei Ausleihe an Schüler).
	AusleiherBenutzerID *string `json:"ausleiher_benutzer_id,omitempty"`
	// AusgeliehenAm ist der genaue Zeitpunkt der Ausleihe.
	AusgeliehenAm time.Time `json:"ausgeliehen_am"`
	// RueckgabeFrist definiert den spätesten Abgabetermin.
	RueckgabeFrist time.Time `json:"rueckgabe_frist"`
	// RueckgabeAm ist der tatsächliche Rückgabezeitpunkt (null bei laufenden Ausleihen).
	RueckgabeAm *time.Time `json:"rueckgabe_am,omitempty"`
	// BearbeiterID verweist auf den Bibliotheksmitarbeiter, der die Ausleihe erfasst hat.
	// Nullbar: die DB-Spalte ist ON DELETE SET NULL, und der GDPR-Anonymisierungs-Job
	// setzt sie auf NULL. Als nicht-nullbarer string scheiterte der Scan mit
	// "cannot scan NULL into *string" → HTTP 500 im Kiosk (Scan/Rückgabe).
	BearbeiterID *string `json:"bearbeiter_id"`
	// RueckgabeBearbeiterID verweist auf den Mitarbeiter, der die Rückgabe erfasst hat (null bei laufenden Ausleihen).
	RueckgabeBearbeiterID *string `json:"rueckgabe_bearbeiter_id,omitempty"`
	// IstFremdrueckgabe zeigt an, ob das Buch von einer anderen Person zurückgebracht wurde.
	IstFremdrueckgabe bool `json:"ist_fremdrueckgabe"`
	// IstHandapparat markiert Dauerleihen (meist an Lehrer) zur Präsenznutzung im Unterricht.
	IstHandapparat bool `json:"ist_handapparat"`
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

// SystematikKategorie repräsentiert eine bibliothekarische Sachgruppe (Tabelle `systematik_kategorien`).
type SystematikKategorie struct {
	// ID ist die UUID der Sachgruppe.
	ID string `json:"id"`
	// Kuerzel ist das systematische Kürzel der Gruppe (z. B. "Nat" für Naturwissenschaften).
	Kuerzel string `json:"kuerzel"`
	// Bezeichnung ist der ausgeschriebene Name der Sachgruppe (z. B. "Naturwissenschaften").
	Bezeichnung string `json:"bezeichnung"`
	// ErstelltAm ist das Datum der Erstellung.
	ErstelltAm time.Time `json:"erstellt_am"`
	// AktualisiertAm ist der letzte Aktualisierungszeitpunkt.
	AktualisiertAm time.Time `json:"aktualisiert_am"`
}

// Lesergruppe repräsentiert eine logische Gruppierung von Lesern (Tabelle `lesergruppen`).
type Lesergruppe struct {
	// ID ist die UUID der Lesergruppe.
	ID string `json:"id"`
	// Kuerzel ist das Kürzel (z. B. "S" für Schüler).
	Kuerzel string `json:"kuerzel"`
	// Bezeichnung ist der ausgeschriebene Name (z. B. "Schülerschaft").
	Bezeichnung string `json:"bezeichnung"`
	// ErstelltAm ist das Datum der Erstellung.
	ErstelltAm time.Time `json:"erstellt_am"`
	// AktualisiertAm ist der letzte Aktualisierungszeitpunkt.
	AktualisiertAm time.Time `json:"aktualisiert_am"`
}

// MailVorlage repräsentiert eine HTML- oder Text-E-Mail-Vorlage für Systemnachrichten (Tabelle `mail_vorlagen`).
type MailVorlage struct {
	// ID ist der Primärschlüssel der Vorlage.
	ID string `json:"id"`
	// Typ definiert den Verwendungszweck (z. B. "mahnung_1", "vormerkung_bereit").
	Typ string `json:"typ"`
	// Betreff ist die Betreffzeile der E-Mail.
	Betreff string `json:"betreff"`
	// TextBody ist der Inhaltstext der E-Mail (kann Variablenplatzhalter enthalten).
	TextBody string `json:"text_body"`
	// UpdatedAt speichert den Zeitpunkt der letzten Bearbeitung.
	UpdatedAt time.Time `json:"updated_at"`
}
