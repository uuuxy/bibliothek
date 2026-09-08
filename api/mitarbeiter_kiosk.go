package api

import "bibliothek/repository"

// MitarbeiterKiosk ist die Theken-Sicht auf eine Lehrkraft bzw. ein Personal-Konto:
// wer das Buch hat, nicht mehr. Bis zum 07.09.2026 lief hier das volle repository.User
// durch — mit E-Mail, Rolle, Aktiv-Flag, Anlagedatum und Antragsdatum — an jeden mit
// perform_actions, also auch an die Helfer-Rolle, während die Schülerseite seit dem
// 18.08.2026 bewusst auf SchuelerKiosk reduziert ist (Sicherheits-Audit 07.09.2026:
// ein Helfer sammelte Kollegiums-Adressen und Rollen durch Scannen von Handapparat-
// Büchern ein). Dieselbe Regel für beide Seiten: Identität ja, Konto nein.
type MitarbeiterKiosk struct {
	ID        string `json:"id"`
	BarcodeID string `json:"barcode_id"`
	Vorname   string `json:"vorname"`
	Nachname  string `json:"nachname"`
}

// zumKioskMitarbeiter reduziert ein volles Konto auf die Theken-Sicht.
func zumKioskMitarbeiter(u *repository.User) *MitarbeiterKiosk {
	if u == nil {
		return nil
	}
	return &MitarbeiterKiosk{ID: u.ID, BarcodeID: u.BarcodeID, Vorname: u.Vorname, Nachname: u.Nachname}
}
