package theke

import "bibliothek/repository"

// thekenLeser ist die Theken-Sicht auf einen Leser: genau die Felder, die Ausleihe und Suche
// an der Scanner-Station brauchen, also Identität, Klasse und die Angaben zur Sperre.
// Wohnanschrift, Eltern-Mail und Geburtsdatum fehlen hier bewusst und dürfen nie dazukommen:
// /api/action und /api/search hängen am Recht perform_actions, das auch die Helfer-Rolle
// trägt, und der Seed verspricht dieser Rolle ausdrücklich keinen Zugriff auf Schülerprofile.
// Der volle repository.Leser-Datensatz an dieser Stelle war ein Sicherheitsbefund
// (bewertung/sicherheitsbefund-kiosk-suche.md). Die vollen Profile gibt es nur hinter
// view_students (/api/schueler…).
type thekenLeser struct {
	ID string `json:"id"`
	// Art: eine aus pkg/leserart (Migration 125 und 153). Die Theke zeigt sie am
	// Treffer an — sonst stünde ein Kollege ohne Klasse da wie ein Schüler mit fehlender
	// Angabe. Keine Personendaten: Sie sagt nichts, was der Ausweis nicht schon sagt.
	Art       string `json:"art,omitempty"`
	BarcodeID string `json:"barcode_id"`
	Vorname   string `json:"vorname"`
	Nachname  string `json:"nachname"`
	Klasse    string `json:"klasse"`
	// Angaben für den Betrieb, keine Personendaten: Die Theke muss sehen, dass eine Ausleihe
	// blockiert ist. Der Grund (block_reason) bleibt draußen; er kann sensiblen Freitext tragen
	// und kommt bei Bedarf als Fehlermeldung der jeweiligen Aktion an.
	IstGesperrt        bool `json:"ist_gesperrt"`
	IstVonHandGesperrt bool `json:"is_manually_blocked"`
}

// zumThekenLeser reduziert einen vollen Leserdatensatz auf die Theken-Sicht.
func zumThekenLeser(leser *repository.Leser) *thekenLeser {
	if leser == nil {
		return nil
	}
	return &thekenLeser{
		ID:                 leser.ID,
		Art:                leser.Art,
		BarcodeID:          leser.BarcodeID,
		Vorname:            leser.Vorname,
		Nachname:           leser.Nachname,
		Klasse:             leser.Klasse,
		IstGesperrt:        leser.IstGesperrt,
		IstVonHandGesperrt: leser.IstVonHandGesperrt,
	}
}

// zuThekenLesern reduziert eine Trefferliste (Suche) auf die Theken-Sicht.
func zuThekenLesern(leser []repository.Leser) []thekenLeser {
	sicht := make([]thekenLeser, 0, len(leser))
	for i := range leser {
		sicht = append(sicht, *zumThekenLeser(&leser[i]))
	}
	return sicht
}
