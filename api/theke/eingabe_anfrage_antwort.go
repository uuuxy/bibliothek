package theke

import (
	"encoding/json"
	"time"

	"bibliothek/repository"
)

// EingabeAnfrage holds the parameters for the Omnibox dispatcher.
type EingabeAnfrage struct {
	Text string `json:"query" validate:"required"`
	// AktiverLeserID ist die Person, die gerade an der Theke steht. Bis Migration 125
	// gab es active_student_id UND active_teacher_id; welche gefüllt war, entschied,
	// in welche Spalte die Ausleihe ging.
	AktiverLeserID       *string `json:"active_leser_id,omitempty" validate:"omitempty,uuid_oder_leer"`
	ChecklisteBestaetigt bool    `json:"confirmed_checklist,omitempty"`
	SperreUebergehen     bool    `json:"override_block,omitempty"`
	IdempotenzSchluessel string  `json:"idempotency_key,omitempty"`
}

// eingabeAntwort is the polymorphic output payload returned by the Omnibox.
type eingabeAntwort struct {
	Art             string               `json:"type"`                       // "student", "ausleihe", "rueckgabe", "search_results", "info"
	Meldung         string               `json:"message,omitempty"`          // Informational message for the frontend
	Leser           *thekenLeser         `json:"student,omitempty"`          // Der aktive Leser oder der Vorbesitzer (Theken-Sicht, siehe thekenLeser)
	Exemplar        *repository.Exemplar `json:"book,omitempty"`             // Book copy details if applicable
	Geraet          *repository.Geraet   `json:"geraet,omitempty"`           // Hardware details if applicable
	Frist           *time.Time           `json:"due_date,omitempty"`         // Return deadline for check-outs
	AusleiheID      *string              `json:"loan_id,omitempty"`          // Loan UUID (for Undo support on returns)
	Fremdrueckgabe  bool                 `json:"fremdrueckgabe,omitempty"`   // Flag for returns from another reader
	Vorbesitzer     *thekenLeser         `json:"vorbesitzer,omitempty"`      // Original borrower if foreign return (Theken-Sicht)
	Treffer         []repository.Titel   `json:"search_results,omitempty"`   // Full-text search list
	HatVormerkung   bool                 `json:"has_vormerkung,omitempty"`   // True if returned book has a pending reservation
	VormerkungTitel string               `json:"vormerkung_titel,omitempty"` // Title name of the reserved book
	VorgemerktFuer  string               `json:"vormerkung_user,omitempty"`  // Reserved for: student name & class
	// RegalfreigabeBarcode: reserved copy in the hold shelf to be returned to the
	// regular shelf because the student took a different copy of the same title.
	RegalfreigabeBarcode string `json:"regalfreigabe_barcode,omitempty"`
	// AufsichtInformieren: Das zurückgebrachte Buch steht auf einem bereits übergebenen
	// Schadensersatz-Bescheid — die Schulaufsicht ist unverzüglich zu informieren (#597).
	// Getrennt von Message, weil es eine Aufgabe ist und keine Erfolgsmeldung.
	AufsichtInformieren string `json:"aufsicht_informieren,omitempty"`
	// Abholbereit: Abholfach-Hinweis beim Schüler-Scan (Betreiber-Entscheidung
	// 01.09.2026) — die Mitarbeiterin sieht sofort, dass für den gescannten
	// Schüler ein vorgemerktes Buch im Abholfach liegt. Titel + Frist, keine IDs;
	// PII-Matrix: Stufe-1-Inhalt wie die Warteliste (Leseinteresse zu Name).
	Abholbereit []abholbereiteVormerkung `json:"abholbereit,omitempty"`
	// AuflagenHinweis: Das eben ausgeliehene Schulbuch ist eine andere Auflage als die, die
	// Kinder derselben Klasse schon haben (docs/OFFEN.md 4.18, Stufe 5) — Klasse, diese
	// Auflage und die anderen mit der Zahl der Kinder, keine Namen. PII-Matrix: Stufe 1 wie
	// die übrige Theken-Antwort (die Klasse des Kindes steht dort ohnehin).
	AuflagenHinweis *repository.AuflagenMischung `json:"auflagen_hinweis,omitempty"`
}

// abholbereiteVormerkung ist ein Eintrag des Abholfach-Hinweises.
type abholbereiteVormerkung struct {
	Titel             string     `json:"titel"`
	BereitgestelltBis *time.Time `json:"bereitgestellt_bis,omitempty"`
}

// thekenEreignis represents the data broadcasted to SSE clients on updates.
type thekenEreignis struct {
	Art        string `json:"event"` // "ausleihe", "rueckgabe", "fremdrueckgabe"
	LeserID    string `json:"student_id,omitempty"`
	ExemplarID string `json:"copy_id,omitempty"`
	BarcodeID  string `json:"barcode_id"`
	Titel      string `json:"titel"`
	Zeitpunkt  int64  `json:"timestamp"`
}

// sendeThekenEreignis broadcasts the action event as JSON via SSE stream.
func (t *Tueren) sendeThekenEreignis(antwort eingabeAntwort) {
	if antwort.Exemplar == nil {
		return
	}
	var leserID string
	if antwort.Leser != nil {
		leserID = antwort.Leser.ID
	}
	ereignis := thekenEreignis{
		Art:        antwort.Art,
		LeserID:    leserID,
		ExemplarID: antwort.Exemplar.ID,
		BarcodeID:  antwort.Exemplar.BarcodeID,
		Titel:      antwort.Exemplar.Titel,
		Zeitpunkt:  time.Now().Unix(),
	}
	if antwort.Fremdrueckgabe {
		ereignis.Art = "fremdrueckgabe"
	}

	daten, err := json.Marshal(ereignis)
	if err == nil {
		t.Broker.Broadcast("action", string(daten))
	}
}
