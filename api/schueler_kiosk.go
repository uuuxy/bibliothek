package api

import "bibliothek/repository"

// SchuelerKiosk ist die Theken-Sicht auf einen Schüler: genau die Felder, die
// Ausleihe und Suche an der Scanner-Station brauchen — Identität, Klasse und
// die Sperr-Flags. Wohnanschrift, Eltern-Mail und Geburtsdatum fehlen hier
// BEWUSST und dürfen nie dazukommen: /api/action und /api/search hängen am
// Recht perform_actions, das auch die Helfer-Rolle trägt, und der Seed
// verspricht dieser Rolle ausdrücklich keinen Zugriff auf Schülerprofile.
// Bis 18.08.2026 lief hier der volle repository.Student-Datensatz durch —
// Sicherheitsbefund bewertung/sicherheitsbefund-kiosk-suche.md. Die vollen
// Profile gibt es weiterhin, aber nur hinter view_students (/api/schueler…).
type SchuelerKiosk struct {
	ID        string `json:"id"`
	BarcodeID string `json:"barcode_id"`
	Vorname   string `json:"vorname"`
	Nachname  string `json:"nachname"`
	Klasse    string `json:"klasse"`
	// Operative Flags, keine Personendaten: Die Theke muss sehen, dass eine
	// Ausleihe blockiert ist. Der Grund (block_reason) bleibt draußen — er
	// kann sensiblen Freitext tragen und kommt bei Bedarf als Fehlermeldung
	// der jeweiligen Aktion an.
	IstGesperrt       bool `json:"ist_gesperrt"`
	IsManuallyBlocked bool `json:"is_manually_blocked"`
}

// zumKioskSchueler reduziert einen vollen Schülerdatensatz auf die Theken-Sicht.
func zumKioskSchueler(s *repository.Student) *SchuelerKiosk {
	if s == nil {
		return nil
	}
	return &SchuelerKiosk{
		ID:                s.ID,
		BarcodeID:         s.BarcodeID,
		Vorname:           s.Vorname,
		Nachname:          s.Nachname,
		Klasse:            s.Klasse,
		IstGesperrt:       s.IstGesperrt,
		IsManuallyBlocked: s.IsManuallyBlocked,
	}
}

// zuKioskSchuelern reduziert eine Trefferliste (Suche) auf die Theken-Sicht.
func zuKioskSchuelern(students []repository.Student) []SchuelerKiosk {
	kiosk := make([]SchuelerKiosk, 0, len(students))
	for i := range students {
		kiosk = append(kiosk, *zumKioskSchueler(&students[i]))
	}
	return kiosk
}
