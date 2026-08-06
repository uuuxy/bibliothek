package api

// LabelFormat defines the dimensions and grid layout for a specific physical label sheet.
type LabelFormat struct {
	FormatID    string
	Name        string
	Cols        int
	Rows        int
	LabelWidth  float64 // width of a single label in mm
	LabelHeight float64 // height of a single label in mm
	MarginTop   float64 // top margin of the page in mm
	MarginLeft  float64 // left margin of the page in mm
	GapX        float64 // horizontal gap between labels in mm
	GapY        float64 // vertical gap between labels in mm
}

var labelFormats = map[string]LabelFormat{
	"zweckform_l4760": {
		FormatID:    "zweckform_l4760",
		Name:        "Zweckform L4760 (3x7, 21 Etiketten)",
		Cols:        3,
		Rows:        7,
		LabelWidth:  63.5, // Avery/Zweckform standard 63.5 x 38.1
		LabelHeight: 38.1,
		MarginTop:   15.1,
		MarginLeft:  7.2,
		GapX:        2.5,
		GapY:        0.0,
	},
	"avery_3475": {
		FormatID:    "avery_3475",
		Name:        "Avery 3475 (3x8, 24 Etiketten)",
		Cols:        3,
		Rows:        8,
		LabelWidth:  70.0,
		LabelHeight: 37.0, // Or 36mm depending on specific sheet, using 37mm
		MarginTop:   0.5,
		MarginLeft:  0.0,
		GapX:        0.0,
		GapY:        0.0,
	},
	"standard_52": {
		FormatID:    "standard_52",
		Name:        "Standard 52 (4x13, 52 Etiketten)",
		Cols:        4,
		Rows:        13,
		LabelWidth:  48.3, // user specified 48.3mm x 21.2mm
		LabelHeight: 21.2,
		MarginTop:   10.7, // Estimate to center
		MarginLeft:  3.4,  // Estimate to center
		GapX:        0.0,
		GapY:        0.0,
	},
}

// StandardLabelFormat ist die Vorgabe, wenn nichts (Brauchbares) gewählt wurde.
const StandardLabelFormat = "zweckform_l4760"

// labelFormatReihenfolge legt fest, in welcher Reihenfolge die Formate angeboten werden.
//
// Nötig, weil eine Go-Map keine stabile Reihenfolge hat: Ohne diese Liste stünden die
// Formate in jedem Aufruf anders in der Auswahl, und der Lieferant müsste jedes Mal neu
// suchen, was er beim letzten Mal genommen hat.
var labelFormatReihenfolge = []string{"zweckform_l4760", "avery_3475", "standard_52"}

// GetLabelFormat retrieves the layout parameters for a given format ID.
// If not found, it returns the default "zweckform_l4760" format.
func GetLabelFormat(id string) (LabelFormat, bool) {
	fmt, ok := labelFormats[id]
	if !ok {
		return labelFormats[StandardLabelFormat], false
	}
	return fmt, true
}

// istBekanntesEtikettFormat prüft eine OPTIONALE Formatangabe aus einem Request.
//
// Leer ist gültig und heißt „nicht angegeben" — die Angabe ist überall freiwillig.
// Ein gesetzter, aber unbekannter Wert wird abgewiesen statt auf die Vorgabe gedreht:
// GetLabelFormat liefert bei Unbekanntem stillschweigend zweckform_l4760 zurück, und
// genau dieses stille Umbiegen soll an der Eingangstür nicht passieren.
func istBekanntesEtikettFormat(id string) bool {
	if id == "" {
		return true
	}
	_, bekannt := GetLabelFormat(id)
	return bekannt
}

// EtikettFormatAuswahl ist ein Format, wie es in einer Auswahlliste erscheint.
type EtikettFormatAuswahl struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	ProSeite int    `json:"pro_seite"`
	Spalten  int    `json:"spalten"`
	Zeilen   int    `json:"zeilen"`
}

// LabelFormatAuswahl liefert die anbietbaren Formate in fester Reihenfolge.
//
// Es gibt sie, damit die Liste EINE Quelle hat. Vorher stand sie zusätzlich als
// hartcodiertes Array im Frontend (LabelLayoutOptionen.svelte) — zwei Listen über
// dieselben Etikettenbögen, und die zweite erfährt es nicht, wenn hier ein Format
// dazukommt oder sich ein Name ändert.
func LabelFormatAuswahl() []EtikettFormatAuswahl {
	auswahl := make([]EtikettFormatAuswahl, 0, len(labelFormatReihenfolge))
	for _, id := range labelFormatReihenfolge {
		f, ok := labelFormats[id]
		if !ok {
			continue
		}
		auswahl = append(auswahl, EtikettFormatAuswahl{
			ID:       f.FormatID,
			Name:     f.Name,
			ProSeite: f.Cols * f.Rows,
			Spalten:  f.Cols,
			Zeilen:   f.Rows,
		})
	}
	return auswahl
}
