package theke

import (
	"time"

	"bibliothek/repository"
)

// EingabeAnfrage ist ein Scan oder eine Suche an der Theke (POST /api/action).
type EingabeAnfrage struct {
	Text string `json:"query" validate:"required"`
	// AktiverLeserID ist die Person, die gerade an der Theke steht, gleich welcher Art:
	// Schüler und Kollegen stehen in einer Tabelle (Migration 125), und die Ausleihe hat eine
	// Spalte für beide.
	AktiverLeserID       *string `json:"active_leser_id,omitempty" validate:"omitempty,uuid_oder_leer"`
	ChecklisteBestaetigt bool    `json:"confirmed_checklist,omitempty"`
	SperreUebergehen     bool    `json:"override_block,omitempty"`
	IdempotenzSchluessel string  `json:"idempotency_key,omitempty"`
}

// eingabeAntwort ist die Antwort der Theke auf eine Eingabe: service.OmniboxErgebnis in der
// Form für JSON, mit der Theken-Sicht auf den Leser (thekenLeser) statt des vollen
// Datensatzes. Was die Felder bedeuten, steht dort.
type eingabeAntwort struct {
	Art             string               `json:"type"`
	Meldung         string               `json:"message,omitempty"`
	Leser           *thekenLeser         `json:"student,omitempty"`
	Exemplar        *repository.Exemplar `json:"book,omitempty"`
	Geraet          *repository.Geraet   `json:"geraet,omitempty"`
	Frist           *time.Time           `json:"due_date,omitempty"`
	AusleiheID      *string              `json:"loan_id,omitempty"`
	Fremdrueckgabe  bool                 `json:"fremdrueckgabe,omitempty"`
	Vorbesitzer     *thekenLeser         `json:"vorbesitzer,omitempty"`
	Treffer         []repository.Titel   `json:"search_results,omitempty"`
	HatVormerkung   bool                 `json:"has_vormerkung,omitempty"`
	VormerkungTitel string               `json:"vormerkung_titel,omitempty"`
	VorgemerktFuer  string               `json:"vormerkung_user,omitempty"`
	// RegalfreigabeBarcode: das vorgemerkte Exemplar im Abholfach, das zurück ins Regal muss,
	// weil der Leser ein anderes Exemplar desselben Titels genommen hat.
	RegalfreigabeBarcode string `json:"regalfreigabe_barcode,omitempty"`
	// AufsichtInformieren: Das zurückgebrachte Buch steht auf einem bereits übergebenen
	// Schadensersatz-Bescheid; die Schulaufsicht ist unverzüglich zu informieren. Getrennt von
	// Meldung, weil es eine Aufgabe ist und keine Erfolgsmeldung.
	AufsichtInformieren string `json:"aufsicht_informieren,omitempty"`
	// Abholbereit: Abholfach-Hinweis beim Scan eines Ausweises; die Kraft an der Theke sieht
	// sofort, dass für den gescannten Schüler ein vorgemerktes Buch im Abholfach liegt. Titel
	// und Frist, keine IDs; PII-Matrix: Stufe-1-Inhalt wie die Warteliste (Leseinteresse zu
	// Name).
	Abholbereit []abholbereiteVormerkung `json:"abholbereit,omitempty"`
	// AuflagenHinweis: Das eben ausgeliehene Schulbuch ist eine andere Auflage als die, die
	// Kinder derselben Klasse schon haben; Klasse, diese Auflage und die anderen mit der Zahl
	// der Kinder, keine Namen. PII-Matrix: Stufe 1 wie die übrige Theken-Antwort (die Klasse
	// des Kindes steht dort ohnehin).
	AuflagenHinweis *repository.AuflagenMischung `json:"auflagen_hinweis,omitempty"`
}

// abholbereiteVormerkung ist ein Eintrag des Abholfach-Hinweises.
type abholbereiteVormerkung struct {
	Titel             string     `json:"titel"`
	BereitgestelltBis *time.Time `json:"bereitgestellt_bis,omitempty"`
}

// sendeThekenEreignis schickt das Ereignis über die Live-Leitung; ohne Exemplar (Gerät)
// schickt es nichts.
func (t *Tueren) sendeThekenEreignis(antwort eingabeAntwort) {
	if antwort.Exemplar == nil {
		return
	}
	var leserID string
	if antwort.Leser != nil {
		leserID = antwort.Leser.ID
	}
	art := antwort.Art
	if antwort.Fremdrueckgabe {
		art = "fremdrueckgabe"
	}
	t.SendeThekenEreignis(art, leserID)
}
