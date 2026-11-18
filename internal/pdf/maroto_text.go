package pdf

import (
	"fmt"

	"bibliothek/internal/pkg/pdfzeichen"

	"github.com/johnfercher/maroto/v2/pkg/components/text"
	"github.com/johnfercher/maroto/v2/pkg/config"
	"github.com/johnfercher/maroto/v2/pkg/consts/fontstyle"
	"github.com/johnfercher/maroto/v2/pkg/core"
	"github.com/johnfercher/maroto/v2/pkg/fontrepository"
	"github.com/johnfercher/maroto/v2/pkg/props"
)

// marotoText ist der Textbaustein der Erzeuger mit maroto. Der Text geht vorher durch
// pdfzeichen.Ersetze: maroto bricht ihn mit gofpdf um, und ein Emoji bräche das Dokument ab. Das
// Text-Paket von maroto bindet nur diese Datei ein (pdfzeichen_ratsche_test.go).
func marotoText(wert string, stil props.Text) core.Component {
	return text.New(pdfzeichen.Ersetze(wert), stil)
}

// marotoKonfig ist der Anfang jeder Konfiguration von maroto: die Schriften der Ausdrucke, Arimo
// als Grundschrift. In einer Standardschrift übersetzte maroto jeden Text nach cp1252 und machte
// aus „ş" einen Punkt; in einer eingebetteten Schrift geht er unverändert aufs Papier.
func marotoKonfig() (config.Builder, error) {
	schnitte, err := pdfzeichen.Schnitte()
	if err != nil {
		return nil, err
	}
	ablage := fontrepository.New()
	for _, s := range schnitte {
		ablage.AddUTF8FontFromBytes(s.Familie, fontstyle.Type(s.Stil), s.Daten)
	}
	schriften, err := ablage.Load()
	if err != nil {
		return nil, fmt.Errorf("schriften für maroto: %w", err)
	}
	return config.NewBuilder().WithCustomFonts(schriften).WithDefaultFont(&props.Font{Family: pdfzeichen.Schrift}), nil
}
