package inventur

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
)

// TestBuchAnfrageDecodesAlleFelder sichert die Regression ab, bei der
// jahrgangVon/jahrgangBis nicht im Request-Struct standen: der JSON-Decoder verwarf
// die Felder still, der Handler schrieb 0/0 in die DB und meldete trotzdem 200
// ("erfolgreich gespeichert", aber der Klassenbereich 11–13 war weg). Der Test
// dekodiert exakt die Nutzlast, die das Frontend sendet, und prüft, dass alle vom
// Formular gebundenen Felder ankommen.
func TestBuchAnfrageDecodesAlleFelder(t *testing.T) {
	// Feldnamen entsprechen den json-Tags aus dem Frontend (BuchEingabefelder*.svelte).
	body := `{
		"isbn": "978-3-16-148410-0",
		"title": "Testtitel",
		"jahrgangVon": 11,
		"jahrgangBis": 13,
		"untertitel": "Ein Untertitel",
		"verlag": "Testverlag",
		"erscheinungsjahr": 2024,
		"beschreibung": "Beschreibungstext"
	}`

	var eingabe BuchAnfrage
	if err := json.NewDecoder(strings.NewReader(body)).Decode(&eingabe); err != nil {
		t.Fatalf("Decode fehlgeschlagen: %v", err)
	}

	if eingabe.JahrgangVon != 11 {
		t.Errorf("JahrgangVon: erwartet 11, bekam %d", eingabe.JahrgangVon)
	}
	if eingabe.JahrgangBis != 13 {
		t.Errorf("JahrgangBis: erwartet 13, bekam %d", eingabe.JahrgangBis)
	}
	if eingabe.Untertitel != "Ein Untertitel" {
		t.Errorf("Untertitel: erwartet 'Ein Untertitel', bekam %q", eingabe.Untertitel)
	}
	if eingabe.Verlag != "Testverlag" {
		t.Errorf("Verlag: erwartet 'Testverlag', bekam %q", eingabe.Verlag)
	}
	if eingabe.Erscheinungsjahr != 2024 {
		t.Errorf("Erscheinungsjahr: erwartet 2024, bekam %d", eingabe.Erscheinungsjahr)
	}
	if eingabe.Beschreibung != "Beschreibungstext" {
		t.Errorf("Beschreibung: erwartet 'Beschreibungstext', bekam %q", eingabe.Beschreibung)
	}
}

func TestBereinigeUndValidiereBuchEingabe(t *testing.T) {
	tests := []struct {
		name        string
		eingabe     BuchAnfrage
		wantErr     bool
		errMsg      string
		wantEingabe *BuchAnfrage
	}{
		{
			name: "Valid input",
			eingabe: BuchAnfrage{
				ISBN:         "978-3-16-148410-0",
				KlassenStufe: 5,
				Bestand:      10,
				Titel:        " Test Titel ",
			},
			wantErr: false,
		},
		{
			name: "Empty ISBN",
			eingabe: BuchAnfrage{
				ISBN:         "",
				KlassenStufe: 5,
				Bestand:      10,
			},
			wantErr: true,
			errMsg:  "isbn darf nicht leer sein",
		},
		{
			name: "Invalid ISBN format",
			eingabe: BuchAnfrage{
				ISBN:         "123",
				KlassenStufe: 5,
				Bestand:      10,
			},
			wantErr: true,
			errMsg:  "ungültiges ISBN-Format",
		},
		{
			name: "Negative gradeLevel",
			eingabe: BuchAnfrage{
				ISBN:         "978-3-16-148410-0",
				KlassenStufe: -1,
				Bestand:      10,
			},
			wantErr: true,
			errMsg:  "gradeLevel muss zwischen 0 und 13 sein",
		},
		{
			name: "gradeLevel too high",
			eingabe: BuchAnfrage{
				ISBN:         "978-3-16-148410-0",
				KlassenStufe: 14,
				Bestand:      10,
			},
			wantErr: true,
			errMsg:  "gradeLevel muss zwischen 0 und 13 sein",
		},
		{
			name: "Negative stock",
			eingabe: BuchAnfrage{
				ISBN:         "978-3-16-148410-0",
				KlassenStufe: 5,
				Bestand:      -1,
			},
			wantErr: true,
			errMsg:  "stock muss >= 0 sein",
		},
		{
			name: "Trims spaces from fields",
			eingabe: BuchAnfrage{
				ISBN:         "  978-3-16-148410-0  ",
				Titel:        "  Titel  ",
				Autor:        "  Autor  ",
				CoverURL:     "  URL  ",
				Fach:         "  Fach  ",
				Schulzweig:   "  Schulzweig  ",
				Medientyp:    "  Medientyp  ",
				Untertitel:   "  Untertitel  ",
				Verlag:       "  Verlag  ",
				Beschreibung: "  Beschreibung  ",
			},
			wantErr: false,
			wantEingabe: &BuchAnfrage{
				ISBN:         "978-3-16-148410-0",
				Titel:        "Titel",
				Autor:        "Autor",
				CoverURL:     "URL",
				Fach:         "Fach",
				Schulzweig:   "Schulzweig",
				Medientyp:    "Medientyp",
				Untertitel:   "Untertitel",
				Verlag:       "Verlag",
				Beschreibung: "Beschreibung",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := bereinigeUndValidiereBuchEingabe(&tt.eingabe)
			if (err != nil) != tt.wantErr {
				t.Errorf("bereinigeUndValidiereBuchEingabe() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err.Error() != tt.errMsg {
				t.Errorf("bereinigeUndValidiereBuchEingabe() expected error message %q, got %q", tt.errMsg, err.Error())
			}
			if !tt.wantErr && tt.wantEingabe != nil {
				assert.Equal(t, *tt.wantEingabe, tt.eingabe)
			}
		})
	}
}
