// Package auskunft ist die Auskunft nach Art. 15 DSGVO über einen Leser: die Typen der Antwort,
// die Pflichtangaben aus den eingestellten Fristen und das Blatt, das aus derselben Antwort
// gedruckt wird. Die Zeilen liest repository/dsgvo_*.go; gesammelt und ausgeliefert wird in
// api/leser/dsgvo_auskunft.go und api/leser/dsgvo_pdf.go.
package auskunft

import (
	"encoding/json"
	"time"

	"bibliothek/repository"
)

// Stammdaten sind alle im Leserdatensatz gespeicherten Stammdaten, auch der Zeitpunkt des
// Löschens und der Sperrgrund: Die Auskunft nennt alles, was gespeichert ist
// (TestDSGVOAuskunft_KenntJedeLeserSpalte in repository hält jede Spalte von leser dagegen). Die Felder
// stehen in der Reihenfolge von repository.DSGVOStammdatenZeile, sonst greift die Typumwandlung
// der Tür nicht.
type Stammdaten struct {
	ID                 string     `json:"id"`
	BarcodeID          string     `json:"barcode_id"`
	Vorname            string     `json:"vorname"`
	Nachname           string     `json:"nachname"`
	Klasse             string     `json:"klasse"`
	Geburtsdatum       *string    `json:"geburtsdatum"`
	AbgaengerJahr      int        `json:"abgaenger_jahr"`
	IstGesperrt        bool       `json:"ist_gesperrt"`
	IstAbgaenger       bool       `json:"ist_abgaenger"`
	LUSDID             *string    `json:"lusd_id"`
	Strasse            string     `json:"strasse"`
	Hausnummer         string     `json:"hausnummer"`
	Plz                string     `json:"plz"`
	Ort                string     `json:"ort"`
	ElternEmail        string     `json:"eltern_email"`
	IstVonHandGesperrt bool       `json:"manuell_gesperrt"`
	Sperrgrund         *string    `json:"sperrgrund"`
	ErstelltAm         time.Time  `json:"erfasst_am"`
	AktualisiertAm     time.Time  `json:"zuletzt_aktualisiert_am"`
	GeloeschtAm        *time.Time `json:"geloescht_am"`
	SchulEintrittAm    *string    `json:"schul_eintritt_am"`
	AbgaengerSeit      *time.Time `json:"abgaenger_seit"`
	// LetzterVorgangAm ist die zweite Uhr der Karenz neben AbgaengerSeit: Der Zeitpunkt bestimmt
	// mit, wann die Daten der Person anonymisiert werden. Was ausgeliehen war, sagt er nicht.
	LetzterVorgangAm *time.Time `json:"letzter_vorgang_am"`
	LUSDBestaetigtAm *time.Time `json:"lusd_bestaetigt_am"`
	AnonymisiertAm   *time.Time `json:"anonymisiert_am"`
	// Art entscheidet, welche Felder gefüllt sind: Ein Kollege hat keine Klasse und kein
	// Abgangsjahr.
	Art string `json:"art"`
	// HatZugangskonto sagt, ob ein Zugangskonto auf diesen Leser zeigt. Die Anmeldedaten stehen
	// in Antwort.Zugangskonto: Sie gehören zum Konto, nicht zum Leser.
	HatZugangskonto bool `json:"hat_zugangskonto"`
}

// Foto beschreibt das (verschlüsselt gespeicherte) Ausweisfoto.
type Foto struct {
	Vorhanden      bool       `json:"vorhanden"`
	AktualisiertAm *time.Time `json:"aktualisiert_am"`
	Hinweis        string     `json:"hinweis"`
}

// Ausleihe ist ein Eintrag der vollständigen Ausleihhistorie.
type Ausleihe struct {
	Gegenstand     string     `json:"gegenstand"`
	Barcode        string     `json:"barcode"`
	AusgeliehenAm  time.Time  `json:"ausgeliehen_am"`
	RueckgabeFrist time.Time  `json:"rueckgabe_frist"`
	RueckgabeAm    *time.Time `json:"rueckgabe_am"`
	IstHandapparat bool       `json:"ist_handapparat"`
}

// Schadensfall ist ein gemeldeter Schadens- oder Verlustfall des Lesers.
type Schadensfall struct {
	Beschreibung      string     `json:"beschreibung"`
	Betrag            string     `json:"betrag_eur"`
	IstBezahlt        bool       `json:"ist_bezahlt"`
	ErstelltAm        time.Time  `json:"erstellt_am"`
	StorniertAm       *time.Time `json:"storniert_am"`
	Stornierungsgrund *string    `json:"stornierungsgrund"`
}

// Vormerkung ist eine Vormerkung auf einen Buchtitel.
type Vormerkung struct {
	Titel      string    `json:"titel"`
	Status     string    `json:"status"`
	Notiz      *string   `json:"notiz"`
	ErstelltAm time.Time `json:"erstellt_am"`
}

// ProtokollEintrag ist ein Eintrag der Datensatz-Historie (audit_log), der den Leser nennt.
type ProtokollEintrag struct {
	// Tabelle sagt, woran der Eintrag hängt (ausleihen, schueler, schadensfaelle …). Mit der
	// Aktion ergibt sie den Vorgang, den das Blatt in Worten nennt.
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
	// Details sind die Angaben des Eintrags, wie der Schreiber sie abgelegt hat.
	Details json.RawMessage `json:"details"`
}

// VerwaltungsEintrag ist ein Eintrag des Verwaltungsprotokolls (audit_logs), der den Leser
// über details->>'schueler_id' nennt, etwa DELETE_STUDENT oder LUSD_ID_NACHGETRAGEN. admin_id
// und ip_adresse fehlen: Sie sind Daten der bearbeitenden Person, nicht des Lesers.
type VerwaltungsEintrag struct {
	Aktion    string          `json:"aktion"`
	Zeitpunkt time.Time       `json:"zeitpunkt"`
	Details   json.RawMessage `json:"details"`
}

// Bescheid ist ein Schadensersatz-Bescheid an diese Person: der Brief mit Nummer, Datum,
// Frist, Summe und Zustand. Seine Positionen stehen als Schadensfälle in der Auskunft.
type Bescheid struct {
	Referenznummer string    `json:"referenznummer"`
	BriefDatum     time.Time `json:"brief_datum"`
	FristBis       time.Time `json:"frist_bis"`
	Gesamtbetrag   string    `json:"gesamtbetrag"`
	Status         string    `json:"status"`
}

// Verarbeitungsangaben sind die Pflichtangaben nach Art. 15 Abs. 1 DSGVO: Zwecke
// (lit. a), Empfänger (lit. c), Speicherdauer (lit. d), Herkunft (lit. g) und die
// Betroffenenrechte samt Beschwerderecht (lit. e und f). Die Datenkategorien (lit. b)
// stehen als die Daten selbst in der Auskunft.
//
// Die Rechtsgrundlage ist keine Angabe des Art. 15; sie gehört zur Information bei der
// Erhebung (Art. 13 Abs. 1 lit. c) und steht hier mit, weil die Auskunft sonst den Grund der
// Verarbeitung nicht nennt. § 83 HSchG ist die Norm zur Erhebung und Verarbeitung
// personenbezogener Daten durch Schulen, § 153 HSchG die Lernmittelfreiheit. Maßgeblich bleibt
// das Verzeichnis von Verarbeitungstätigkeiten der Schule.
type Verarbeitungsangaben struct {
	Zwecke            []string `json:"zwecke"`
	Rechtsgrundlage   string   `json:"rechtsgrundlage"`
	Empfaenger        string   `json:"empfaenger"`
	Speicherdauer     string   `json:"speicherdauer"`
	Herkunft          string   `json:"herkunft_der_daten"`
	Betroffenenrechte string   `json:"betroffenenrechte"`
}

// Antwort ist die vollständige Betroffenenauskunft nach Art. 15 DSGVO.
type Antwort struct {
	Art                string               `json:"art"`
	ErstelltAm         time.Time            `json:"auskunft_erstellt_am"`
	Stammdaten         Stammdaten           `json:"stammdaten"`
	Foto               Foto                 `json:"ausweisfoto"`
	Ausleihen          []Ausleihe           `json:"ausleihhistorie"`
	Schadensfaelle     []Schadensfall       `json:"schadensfaelle"`
	Vormerkungen       []Vormerkung         `json:"vormerkungen"`
	Bescheide          []Bescheid           `json:"schadensersatz_bescheide"`
	NachbuchMeldungen  []NachbuchMeldung    `json:"nachbuch_meldungen"`
	ProtokollEintraege []ProtokollEintrag   `json:"protokolleintraege"`
	Verwaltung         []VerwaltungsEintrag `json:"verwaltungsprotokolle"`
	// Das Konto, mit dem sich die Person anmeldet, samt Anfragen und Kontoereignissen;
	// null, wenn auf diesen Leser kein Konto zeigt (bei Schülern der Regelfall).
	Zugangskonto *repository.DSGVOZugangskonto `json:"zugangskonto"`
	// Gelöschte Konten, die auf diesen Leser zeigten, samt den Einträgen über sie. Leer, wenn
	// es keine gab.
	FruehereZugangskonten []repository.DSGVOFrueheresZugangskonto `json:"fruehere_zugangskonten"`
	Verarbeitungsangaben  Verarbeitungsangaben                    `json:"verarbeitungsangaben"`
}

// NachbuchMeldung ist eine Nachbuch-Meldung der Theke, in der die Person als Ausleiher
// oder Vorbesitzer steht.
type NachbuchMeldung struct {
	Rolle       string     `json:"rolle"` // ausleiher | vorbesitzer
	Barcode     string     `json:"barcode"`
	Ergebnis    string     `json:"ergebnis"`
	Grund       *string    `json:"grund"`
	GescanntAm  time.Time  `json:"gescannt_am"`
	QuittiertAm *time.Time `json:"quittiert_am"`
}
