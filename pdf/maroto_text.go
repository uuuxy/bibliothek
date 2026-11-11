package pdf

import (
	"bibliothek/pkg/pdfzeichen"

	"github.com/johnfercher/maroto/v2/pkg/components/text"
	"github.com/johnfercher/maroto/v2/pkg/core"
	"github.com/johnfercher/maroto/v2/pkg/props"
)

// marotoText ist der Textbaustein der Erzeuger mit maroto. maroto übersetzt jeden Text in einer
// Standardschrift mit dem cp1252-Übersetzer von gofpdf und macht aus „ş" einen Punkt; die
// Ersetzung aus pdfzeichen läuft deshalb vorher. Das Text-Paket von maroto bindet nur diese Datei
// ein (pdfzeichen_ratsche_test.go).
func marotoText(wert string, stil props.Text) core.Component {
	return text.New(pdfzeichen.Ersetze(wert), stil)
}
