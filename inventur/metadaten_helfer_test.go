package inventur

import (
	"testing"
)

func TestAutomatischeKategorisierung(t *testing.T) {
	tests := []struct {
		name       string
		titel      string
		untertitel string
		wantFach   string
		wantStufe  string
	}{
		{
			name:       "Math book, class 7",
			titel:      "Mathematik für Gymnasien",
			untertitel: "Klasse 7",
			wantFach:   "Mathematik",
			wantStufe:  "7",
		},
		{
			name:       "English book, volume 2",
			titel:      "English G Access",
			untertitel: "Band 2",
			wantFach:   "Englisch",
			wantStufe:  "2",
		},
		{
			name:       "Biology, grade 10, uppercase test",
			titel:      "BIOLOGIE",
			untertitel: "Jahrgangsstufe 10",
			wantFach:   "Biologie",
			wantStufe:  "10",
		},
		{
			name:       "French with numeric grade only",
			titel:      "Découvertes",
			untertitel: "für Französisch 6",
			wantFach:   "Französisch",
			wantStufe:  "6",
		},
		{
			name:       "No subject matched, but grade matched",
			titel:      "Allgemeines Buch",
			untertitel: "Stufe 8",
			wantFach:   "",
			wantStufe:  "8",
		},
		{
			name:       "Subject matched, but no grade",
			titel:      "Chemie Grundlagen",
			untertitel: "",
			wantFach:   "Chemie",
			wantStufe:  "",
		},
		{
			name:       "Nothing matched",
			titel:      "Ein spannender Roman",
			untertitel: "Teil Drei", // spelled out part, no digits
			wantFach:   "",
			wantStufe:  "",
		},
		{
			name:       "Math alias algebra",
			titel:      "Algebra und mehr",
			untertitel: "Level 9",
			wantFach:   "Mathematik",
			wantStufe:  "9",
		},
		{
			name:       "Fallback to single number in text",
			titel:      "Geschichte entdecken",
			untertitel: "Ausgabe 12 Hessen",
			wantFach:   "Geschichte",
			wantStufe:  "12",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotFach, gotStufe := automatischeKategorisierung(tt.titel, tt.untertitel)
			if gotFach != tt.wantFach {
				t.Errorf("automatischeKategorisierung() gotFach = %v, want %v", gotFach, tt.wantFach)
			}
			if gotStufe != tt.wantStufe {
				t.Errorf("automatischeKategorisierung() gotStufe = %v, want %v", gotStufe, tt.wantStufe)
			}
		})
	}
}
