// Package auskunft ist die Auskunft nach Art. 15 DSGVO über einen Leser: die Typen der Antwort,
// die Pflichtangaben aus den eingestellten Fristen und das Blatt, das aus derselben Antwort
// gedruckt wird. Die Zeilen liest repository/dsgvo_*.go; gesammelt und ausgeliefert wird in
// api/dsgvo_auskunft.go und api/dsgvo_pdf.go.
package auskunft

import (
	"encoding/json"
	"time"

	"bibliothek/repository"
)

// DsgvoStammdaten umfasst sämtliche im Leserdatensatz gespeicherten
// Stammdaten — bewusst inklusive Soft-Delete-Zeitpunkt und Sperrgrund,
// denn die Auskunft nach Art. 15 DSGVO deckt alles ab, was gespeichert ist.
// „Sämtliche" ist gemessen: TestDsgvoAuskunft_KenntJedeLeserSpalte hält jede Spalte von leser
// gegen repository.DsgvoStammdatenSQL. Die Felder stehen in derselben Reihenfolge wie in
// repository.DsgvoStammdatenZeile, sonst greift die Typumwandlung nicht.
type DsgvoStammdaten struct {
	ID                string     `json:"id"`
	BarcodeID         string     `json:"barcode_id"`
	Vorname           string     `json:"vorname"`
	Nachname          string     `json:"nachname"`
	Klasse            string     `json:"klasse"`
	Geburtsdatum      *string    `json:"geburtsdatum"`
	AbgaengerJahr     int        `json:"abgaenger_jahr"`
	IstGesperrt       bool       `json:"ist_gesperrt"`
	IstAbgaenger      bool       `json:"ist_abgaenger"`
	LusdID            *string    `json:"lusd_id"`
	Strasse           string     `json:"strasse"`
	Hausnummer        string     `json:"hausnummer"`
	Plz               string     `json:"plz"`
	Ort               string     `json:"ort"`
	ElternEmail       string     `json:"eltern_email"`
	IsManuallyBlocked bool       `json:"manuell_gesperrt"`
	BlockReason       *string    `json:"sperrgrund"`
	ErstelltAm        time.Time  `json:"erfasst_am"`
	AktualisiertAm    time.Time  `json:"zuletzt_aktualisiert_am"`
	GeloeschtAm       *time.Time `json:"geloescht_am"`
	// Seit Migration 084/094 (nachgetragen 02.09.2026 — die Auskunft war um vier Spalten
	// unvollständig; Gate: TestDsgvoAuskunft_KenntJedeLeserSpalte).
	SchulEintrittAm *string    `json:"schul_eintritt_am"`
	AbgaengerSeit   *time.Time `json:"abgaenger_seit"`
	// Migration 137: der Zeitpunkt des letzten abgeschlossenen Vorgangs — die zweite Uhr
	// der Karenz neben abgaenger_seit. Er gehört in die Auskunft, weil er ein über diese
	// Person gespeicherter Zeitpunkt ist und weil er mitbestimmt, wann ihre Daten
	// anonymisiert werden. WAS ausgeliehen war, sagt er nicht.
	LetzterVorgangAm *time.Time `json:"letzter_vorgang_am"`
	LusdBestaetigtAm *time.Time `json:"lusd_bestaetigt_am"`
	AnonymisiertAm   *time.Time `json:"anonymisiert_am"`
	// Migration 123: Die Tabelle führt alle Leser. Die Art gehört in die Auskunft, weil
	// sie über die Person etwas aussagt — und weil sie entscheidet, welche Felder
	// überhaupt gefüllt sind (ein Kollege hat keine Klasse und kein Abgängerjahr).
	Art string `json:"art"`
	// Zeigt ein Zugangskonto auf diesen Leser? Die Anmeldedaten selbst (E-Mail, Rolle)
	// stehen nicht hier, sondern im eigenen Teil der Auskunft (DsgvoAuskunftResponse.
	// Zugangskonto): Sie gehören zum Konto, nicht zum Leser.
	HatZugangskonto bool `json:"hat_zugangskonto"`
}

// DsgvoFoto beschreibt das (verschlüsselt gespeicherte) Ausweisfoto.
type DsgvoFoto struct {
	Vorhanden      bool       `json:"vorhanden"`
	AktualisiertAm *time.Time `json:"aktualisiert_am"`
	Hinweis        string     `json:"hinweis"`
}

// DsgvoAusleihe ist ein Eintrag der vollständigen Ausleihhistorie.
type DsgvoAusleihe struct {
	Gegenstand     string     `json:"gegenstand"`
	Barcode        string     `json:"barcode"`
	AusgeliehenAm  time.Time  `json:"ausgeliehen_am"`
	RueckgabeFrist time.Time  `json:"rueckgabe_frist"`
	RueckgabeAm    *time.Time `json:"rueckgabe_am"`
	IstHandapparat bool       `json:"ist_handapparat"`
}

// DsgvoSchadensfall ist ein gemeldeter Schadens-/Verlustfall des Schülers.
type DsgvoSchadensfall struct {
	Beschreibung      string     `json:"beschreibung"`
	Betrag            string     `json:"betrag_eur"`
	IstBezahlt        bool       `json:"ist_bezahlt"`
	ErstelltAm        time.Time  `json:"erstellt_am"`
	StorniertAm       *time.Time `json:"storniert_am"`
	Stornierungsgrund *string    `json:"stornierungsgrund"`
}

// DsgvoVormerkung ist eine Vormerkung auf einen Buchtitel.
type DsgvoVormerkung struct {
	Titel      string    `json:"titel"`
	Status     string    `json:"status"`
	Notiz      *string   `json:"notiz"`
	ErstelltAm time.Time `json:"erstellt_am"`
}

// DsgvoAuditEintrag ist ein Eintrag der Datensatz-Historie (audit_log), der den Leser nennt.
type DsgvoAuditEintrag struct {
	// Tabelle sagt, woran der Eintrag hängt (ausleihen, schueler, schadensfaelle …). Mit der
	// Aktion ergibt sie den Vorgang, den das Blatt in Worten nennt (dsgvoVorgang).
	Tabelle   string    `json:"tabelle"`
	Aktion    string    `json:"aktion"`
	Akteur    string    `json:"akteur"`
	Zeitpunkt time.Time `json:"zeitpunkt"`
	Kontext   *string   `json:"kontext"`
	// Gegenstand und Barcode: Titel und Nummer des Buchs oder Geräts einer Ausleihe oder
	// Rückgabe; der Eintrag selbst trägt nur die Kennung des Exemplars. Leer, wenn es das
	// Exemplar nicht mehr gibt oder der Eintrag kein Buch betrifft.
	Gegenstand string `json:"gegenstand"`
	Barcode    string `json:"barcode"`
	// swaggertype: json.RawMessage ist ein []byte-Alias aus der Standardbibliothek, das
	// swag ohne --parseDependency nicht auflösen kann. Ohne diesen Hinweis bricht die
	// Generierung für DIESEN Endpunkt still ab — die DSGVO-Auskunft fehlte deshalb
	// komplett in der Swagger-Datei, obwohl sie annotiert war (gefunden 05.08.2026).
	Details json.RawMessage `json:"details" swaggertype:"object"`
}

// DsgvoVerwaltungsEintrag ist ein Eintrag des Verwaltungsprotokolls (audit_logs):
// Admin-Eingriffe wie DELETE/RESTORE/PURGE_STUDENT oder LUSD_ID_NACHGETRAGEN, die den
// Schüler über details->>'schueler_id' referenzieren. admin_id und ip_adresse sind
// Daten des BEARBEITERS, nicht des Schülers — sie gehören nicht in dessen Auskunft.
type DsgvoVerwaltungsEintrag struct {
	Aktion    string          `json:"aktion"`
	Zeitpunkt time.Time       `json:"zeitpunkt"`
	Details   json.RawMessage `json:"details" swaggertype:"object"`
}

// DsgvoBescheid ist ein Schadensersatz-Bescheid, der an diese Person gerichtet war
// (Migration 110). Die Positionen selbst stehen als Schadensfälle im Abschnitt darüber;
// hier steht der BRIEF: Nummer, Datum, Frist, Summe, Zustand.
type DsgvoBescheid struct {
	Referenznummer string    `json:"referenznummer"`
	BriefDatum     time.Time `json:"brief_datum"`
	FristBis       time.Time `json:"frist_bis"`
	Gesamtbetrag   string    `json:"gesamtbetrag"`
	Status         string    `json:"status"`
}

// DsgvoVerarbeitungsangaben sind die Pflichtangaben nach Art. 15 Abs. 1 DSGVO: Zwecke
// (lit. a), Empfänger (lit. c), Speicherdauer (lit. d), Herkunft (lit. g) und die
// Betroffenenrechte samt Beschwerderecht (lit. e und f). Die Datenkategorien (lit. b)
// stehen als die Daten selbst in der Auskunft.
//
// Die Rechtsgrundlage ist KEINE Angabe des Art. 15 — sie gehört zur Information bei der
// Erhebung (Art. 13 Abs. 1 lit. c) und steht hier freiwillig mit, weil die Auskunft sonst
// den Grund der Verarbeitung nicht nennt. Maßgeblich bleibt das Verzeichnis von
// Verarbeitungstätigkeiten der Schule.
//
// Geprüft am 10.09.2026 gegen die Fundstellen: § 83 HSchG ist die Norm zur Erhebung und
// Verarbeitung personenbezogener Daten durch Schulen, § 153 HSchG die Lernmittelfreiheit
// (Schulbücher bleiben Eigentum des Landes und werden befristet überlassen). Bis dahin
// klammerte dieser Text beide Paragrafen unter „(Lernmittelfreiheit)" und nannte die LUSD
// „Landesschülerdatenbank" — zwei falsche Angaben in einem Dokument, das die betroffene
// Person in die Hand bekommt. VVT-Entwurf und SECURITY.md waren die ganze Zeit richtig.
type DsgvoVerarbeitungsangaben struct {
	Zwecke            []string `json:"zwecke"`
	Rechtsgrundlage   string   `json:"rechtsgrundlage"`
	Empfaenger        string   `json:"empfaenger"`
	Speicherdauer     string   `json:"speicherdauer"`
	Herkunft          string   `json:"herkunft_der_daten"`
	Betroffenenrechte string   `json:"betroffenenrechte"`
}

// DsgvoAuskunftResponse ist die vollständige Betroffenenauskunft nach Art. 15 DSGVO.
type DsgvoAuskunftResponse struct {
	Art               string                    `json:"art"`
	ErstelltAm        time.Time                 `json:"auskunft_erstellt_am"`
	Stammdaten        DsgvoStammdaten           `json:"stammdaten"`
	Foto              DsgvoFoto                 `json:"ausweisfoto"`
	Ausleihen         []DsgvoAusleihe           `json:"ausleihhistorie"`
	Schadensfaelle    []DsgvoSchadensfall       `json:"schadensfaelle"`
	Vormerkungen      []DsgvoVormerkung         `json:"vormerkungen"`
	Bescheide         []DsgvoBescheid           `json:"schadensersatz_bescheide"`
	NachbuchMeldungen []DsgvoNachbuchMeldung    `json:"nachbuch_meldungen"`
	AuditEintraege    []DsgvoAuditEintrag       `json:"protokolleintraege"`
	Verwaltung        []DsgvoVerwaltungsEintrag `json:"verwaltungsprotokolle"`
	// Das Konto, mit dem sich die Person anmeldet, samt Anfragen und Kontoereignissen;
	// null, wenn auf diesen Leser kein Konto zeigt (bei Schülern der Regelfall).
	Zugangskonto *repository.DsgvoZugangskonto `json:"zugangskonto"`
	// Gelöschte Konten, die auf diesen Leser zeigten, samt den Einträgen über sie (seit
	// 29.09.2026). Leer, wenn es keine gab.
	FruehereZugangskonten []repository.DsgvoFrueheresZugangskonto `json:"fruehere_zugangskonten"`
	Verarbeitungsangaben  DsgvoVerarbeitungsangaben               `json:"verarbeitungsangaben"`
}

// DsgvoNachbuchMeldung ist eine Nachbuch-Meldung der Theke (Migration 117), in der die
// Person als Ausleiher oder Vorbesitzer steht — mit Rolle, Barcode, Ergebnis und Zeitpunkt.
type DsgvoNachbuchMeldung struct {
	Rolle       string     `json:"rolle"` // ausleiher | vorbesitzer
	Barcode     string     `json:"barcode"`
	Ergebnis    string     `json:"ergebnis"`
	Grund       *string    `json:"grund"`
	GescanntAm  time.Time  `json:"gescannt_am"`
	QuittiertAm *time.Time `json:"quittiert_am"`
}
