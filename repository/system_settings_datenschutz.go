package repository

import "strconv"

// Datenschutz- und Sitzungs-Einstellungen (A1/A4 aus docs/datenschutz_offene_punkte.md).
//
// Alle vier Werte sind Zeiger mit der Semantik von OeffentlicheAdresse: nil heißt „diese
// Sektion kennt das Feld nicht" (gespeicherter Wert bleibt), ein gesetzter Zeiger heißt
// „so speichern" — auch 0. Denn 0 ist hier ein echter Wert (= aus), kein fehlender: Ein
// int ohne Zeiger könnte „nicht mitgeschickt" und „abgeschaltet" nicht unterscheiden, und
// jedes Speichern einer anderen Einstellungs-Sektion hätte die Befristung still
// abgeschaltet (Upsert-Blanking).

const (
	// StandardLesehistorieTage sind die Tage nach der Rückgabe, nach denen eine Schülerbücherei-
	// Ausleihe (Freihand, Medien, Geräte) vom Schüler getrennt wird. Das HBDI-Muster-VVT
	// „Schulbibliothek" verlangt Löschung, „sobald nicht mehr notwendig"; 90 Tage decken
	// Nachfragen zu einer Rückgabe (später bemerkter Schaden, Fremdrückgabe-Klärung).
	StandardLesehistorieTage = 90
	// StandardLesehistorieLernmittelTage ist die Lernmittel-Frist: Lernmittel bleiben länger zuordenbar, weil die
	// Bestandskartei Ausleihe UND Rücklauf nachweisen muss (HKM-Leitfaden LMF 11.3) und
	// Schadensersatz öffentlich-rechtlich über die Schulaufsicht läuft (12.3–12.7) —
	// das zieht sich über das Folgeschuljahr. Zwei Jahre nach Rückgabe.
	StandardLesehistorieLernmittelTage = 730
	// StandardThekeLeerenMinuten ist die Inaktivität, nach der die Theken-Ansicht den geladenen
	// Schüler fallen lässt — damit der nächste an der Theke nicht den vorigen sieht.
	StandardThekeLeerenMinuten = 5
	// StandardSperreMinuten ist die Inaktivität, nach der der Sperrbildschirm kommt.
	StandardSperreMinuten = 15
)

func zeiger(n int) *int { return &n }

// datenschutzStandards füllt die Vorgaben der vier Felder.
func datenschutzStandards(s *SystemEinstellungen) {
	s.LesehistorieTage = zeiger(StandardLesehistorieTage)
	s.LesehistorieLernmittelTage = zeiger(StandardLesehistorieLernmittelTage)
	s.ThekeLeerenMinuten = zeiger(StandardThekeLeerenMinuten)
	s.SperreMinuten = zeiger(StandardSperreMinuten)
}

// setzeIntZeiger übernimmt einen DB-Wert in ein Zeigerfeld; unlesbare Werte lassen die
// Vorgabe stehen. Negative Werte werden wie 0 (aus) behandelt.
func setzeIntZeiger(val *string, ziel **int) {
	if val == nil || *val == "" {
		return
	}
	v, err := strconv.Atoi(*val)
	if err != nil {
		return
	}
	if v < 0 {
		v = 0
	}
	*ziel = zeiger(v)
}

// anwendenDatenschutzEinstellung meldet true, wenn der Schlüssel hierher gehört.
func anwendenDatenschutzEinstellung(s *SystemEinstellungen, key string, val *string) bool {
	switch key {
	case "lesehistorie_tage":
		setzeIntZeiger(val, &s.LesehistorieTage)
	case "lesehistorie_lernmittel_tage":
		setzeIntZeiger(val, &s.LesehistorieLernmittelTage)
	case "theke_leeren_minuten":
		setzeIntZeiger(val, &s.ThekeLeerenMinuten)
	case "sperre_minuten":
		setzeIntZeiger(val, &s.SperreMinuten)
	default:
		return false
	}
	return true
}

// TageOderStandard liefert den Wert eines Zeigerfelds oder die Vorgabe, wenn es nil ist.
func TageOderStandard(v *int, standard int) int {
	if v == nil {
		return standard
	}
	return *v
}
