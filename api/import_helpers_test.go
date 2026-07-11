package api

import "testing"

func TestDetectCSVDelimiter(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    rune
	}{
		{"Komma-Header", "Titel,Autor,ISBN\nx,y,z", ','},
		{"Semikolon-Header", "Titel;Autor;ISBN;Zustand\nx;y;z;verfuegbar", ';'},
		{"Semikolon trotz Komma im Wert", "Titel;Kategorie\nx;\"a, b, c\"", ';'},
		{"nur eine Spalte", "Titel", ','},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := detectCSVDelimiter(tt.content); got != tt.want {
				t.Errorf("detectCSVDelimiter(%q) = %q, want %q", tt.content, got, tt.want)
			}
		})
	}
}

func TestBuildImportHeaderMap(t *testing.T) {
	// Schlanker Littera-Export (7 Spalten, ohne Zustand)
	m := buildImportHeaderMap([]string{"Titel", "Autor", "Verlag", "ISBN", "Jahr", "Kategorie", "Barcode"})
	for col, want := range map[string]int{"titel": 0, "autor": 1, "verlag": 2, "isbn": 3, "jahr": 4, "kategorie": 5, "barcode": 6} {
		if m[col] != want {
			t.Errorf("Littera-Header: %s = %d, want %d", col, m[col], want)
		}
	}
	if _, ok := m["zustand"]; ok {
		t.Errorf("Littera-Header sollte keine Zustand-Spalte haben")
	}

	// Volle Bestandsdatei (8 Spalten, inkl. Zustand)
	full := buildImportHeaderMap([]string{"Titel", "Autor", "Verlag", "ISBN", "Jahr", "Kategorie", "Barcode", "Zustand"})
	if idx, ok := full["zustand"]; !ok || idx != 7 {
		t.Errorf("Bestand-Header: zustand = %d (ok=%v), want 7", idx, ok)
	}

	// Alternative Spaltennamen (Exemplarnummer statt Barcode, Systematik statt Kategorie)
	alt := buildImportHeaderMap([]string{"Titelliste", "Verfasser", "Systematik", "Exemplarnummer"})
	if alt["titel"] != 0 || alt["autor"] != 1 || alt["kategorie"] != 2 || alt["barcode"] != 3 {
		t.Errorf("Alternative Header falsch gemappt: %+v", alt)
	}
}
