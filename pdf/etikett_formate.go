package pdf

// EtikettFormat beschreibt einen Bogen mit Etiketten: Raster, Maße eines Etiketts und Ränder, alle Maße in mm.
type EtikettFormat struct {
	ID        string
	Name      string
	Spalten   int
	Zeilen    int
	Breite    float64 // eines Etiketts
	Hoehe     float64 // eines Etiketts
	RandOben  float64 // der Seite
	RandLinks float64 // der Seite
	AbstandX  float64 // zwischen zwei Etiketten nebeneinander
	AbstandY  float64 // zwischen zwei Etiketten untereinander
}

var etikettFormate = map[string]EtikettFormat{
	"zweckform_l4760": {
		ID:        "zweckform_l4760",
		Name:      "Zweckform L4760 (3x7, 21 Etiketten)",
		Spalten:   3,
		Zeilen:    7,
		Breite:    63.5, // Avery/Zweckform 63,5 x 38,1
		Hoehe:     38.1,
		RandOben:  15.1,
		RandLinks: 7.2,
		AbstandX:  2.5,
		AbstandY:  0.0,
	},
	"avery_3475": {
		ID:        "avery_3475",
		Name:      "Avery 3475 (3x8, 24 Etiketten)",
		Spalten:   3,
		Zeilen:    8,
		Breite:    70.0,
		Hoehe:     37.0, // manche Bögen haben 36 mm
		RandOben:  0.5,
		RandLinks: 0.0,
		AbstandX:  0.0,
		AbstandY:  0.0,
	},
	"standard_52": {
		ID:        "standard_52",
		Name:      "Standard 52 (4x13, 52 Etiketten)",
		Spalten:   4,
		Zeilen:    13,
		Breite:    48.3, // 48,3 x 21,2 mm nach Angabe der Schule
		Hoehe:     21.2,
		RandOben:  10.7, // geschätzt, damit das Raster mittig liegt
		RandLinks: 3.4,  // geschätzt, damit das Raster mittig liegt
		AbstandX:  0.0,
		AbstandY:  0.0,
	},
}

// EtikettFormatVorgabe gilt, wenn kein brauchbares Format gewählt ist.
const EtikettFormatVorgabe = "zweckform_l4760"

// etikettFormatReihenfolge legt fest, in welcher Reihenfolge die Formate angeboten werden. Eine
// Map in Go hat keine feste Reihenfolge; ohne diese Liste stünden die Formate in jeder Auswahl
// anders, und der Lieferant müsste jedes Mal neu suchen, was er beim letzten Mal genommen hat.
var etikettFormatReihenfolge = []string{"zweckform_l4760", "avery_3475", "standard_52"}

// FindeEtikettFormat liefert das Format zu id. Gibt es keines, liefert sie das Format der
// Vorgabe und false.
func FindeEtikettFormat(id string) (EtikettFormat, bool) {
	format, ok := etikettFormate[id]
	if !ok {
		return etikettFormate[EtikettFormatVorgabe], false
	}
	return format, true
}

// IstBekanntesEtikettFormat prüft eine freiwillige Angabe des Formats aus einer Anfrage. Leer
// heißt „nicht angegeben" und ist gültig. Ein unbekannter Wert wird abgewiesen statt still auf
// die Vorgabe gedreht, wie es FindeEtikettFormat täte.
func IstBekanntesEtikettFormat(id string) bool {
	if id == "" {
		return true
	}
	_, bekannt := FindeEtikettFormat(id)
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

// EtikettFormate liefert die Formate, die angeboten werden, in fester Reihenfolge. Die Liste hat
// hier ihre einzige Quelle; eine zweite in der Oberfläche erführe nicht, wenn ein Format
// dazukommt oder einen anderen Namen bekommt.
func EtikettFormate() []EtikettFormatAuswahl {
	auswahl := make([]EtikettFormatAuswahl, 0, len(etikettFormatReihenfolge))
	for _, id := range etikettFormatReihenfolge {
		f, ok := etikettFormate[id]
		if !ok {
			continue
		}
		auswahl = append(auswahl, EtikettFormatAuswahl{
			ID:       f.ID,
			Name:     f.Name,
			ProSeite: f.Spalten * f.Zeilen,
			Spalten:  f.Spalten,
			Zeilen:   f.Zeilen,
		})
	}
	return auswahl
}
