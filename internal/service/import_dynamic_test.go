package service

import (
	"bibliothek/repository"
	"context"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
)

func TestCleanISBN(t *testing.T) {
	tests := []struct {
		val  string
		want string
	}{
		{"978-3-16-148410-0", "9783161484100"},
		{" 978 3 16 148410 0 ", "9783161484100"},
		{"9783161484100", "9783161484100"},
		{"", ""},
		{"abc-def ghi", "abcdefghi"},
	}

	for _, tt := range tests {
		if got := cleanISBN(tt.val); got != tt.want {
			t.Errorf("cleanISBN(%q) = %q, want %q", tt.val, got, tt.want)
		}
	}
}

func TestMatchTitelID(t *testing.T) {
	isbnToID := map[string]string{
		"1234567890": "id-isbn",
	}
	titelToID := map[string]string{
		repository.NormalisiereTitelKey("Buch Titel"): "id-titel",
	}

	tests := []struct {
		name  string
		isbn  string
		titel string
		want  string
	}{
		{"match by isbn", "1234567890", "Anderer Titel", "id-isbn"},
		{"match by titel", "", "Buch Titel", "id-titel"},
		{"match by titel with isbn missing in map", "999", "Buch Titel", "id-titel"},
		{"no match", "999", "Unbekannt", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := matchTitelID(tt.isbn, tt.titel, isbnToID, titelToID); got != tt.want {
				t.Errorf("matchTitelID(%q, %q) = %q, want %q", tt.isbn, tt.titel, got, tt.want)
			}
		})
	}
}

func TestBaueNeuTitelAusZeile(t *testing.T) {
	headerMap := map[string]int{
		"titel":     0,
		"autor":     1,
		"verlag":    2,
		"isbn":      3,
		"jahr":      4,
		"kategorie": 5,
		"signatur":  6,
		"barcode":   7,
	}

	isbnToID := map[string]string{"123": "id1"}
	titelToID := map[string]string{repository.NormalisiereTitelKey("Bekannt"): "id2"}

	tests := []struct {
		name         string
		row          []string
		wantCacheKey string
		wantOk       bool
		wantISBN     string
	}{
		{
			name:         "new title by isbn",
			row:          []string{"Neu 1", "Autor", "Verlag", "978-3", "2020", "Kat", "Sig", "BC1"},
			wantCacheKey: "9783",
			wantOk:       true,
			wantISBN:     "9783",
		},
		{
			name:         "new title by titel (no isbn)",
			row:          []string{"Neu 2", "Autor", "Verlag", "", "2020", "Kat", "Sig", "BC2"},
			wantCacheKey: repository.NormalisiereTitelKey("Neu 2"),
			wantOk:       true,
			wantISBN:     "",
		},
		{
			name:         "existing title by isbn",
			row:          []string{"Neu 3", "Autor", "Verlag", "123", "2020", "Kat", "Sig", "BC3"},
			wantCacheKey: "",
			wantOk:       false,
		},
		{
			name:         "existing title by titel",
			row:          []string{"Bekannt", "Autor", "Verlag", "", "2020", "Kat", "Sig", "BC4"},
			wantCacheKey: "",
			wantOk:       false,
		},
		{
			name:         "missing titel",
			row:          []string{"", "Autor", "Verlag", "999", "2020", "Kat", "Sig", "BC5"},
			wantCacheKey: "",
			wantOk:       false,
		},
		{
			name:         "missing barcode",
			row:          []string{"Titel", "Autor", "Verlag", "999", "2020", "Kat", "Sig", ""},
			wantCacheKey: "",
			wantOk:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cacheKey, newTitel, ok := baueNeuTitelAusZeile(tt.row, headerMap, isbnToID, titelToID)
			if ok != tt.wantOk {
				t.Errorf("baueNeuTitelAusZeile ok = %v, want %v", ok, tt.wantOk)
			}
			if ok {
				if cacheKey != tt.wantCacheKey {
					t.Errorf("cacheKey = %q, want %q", cacheKey, tt.wantCacheKey)
				}
				if newTitel.ISBN != tt.wantISBN {
					t.Errorf("ISBN = %q, want %q", newTitel.ISBN, tt.wantISBN)
				}
			}
		})
	}
}

func TestSammleNeueTitel(t *testing.T) {
	headerMap := map[string]int{"titel": 0, "isbn": 1, "barcode": 2}
	rows := [][]string{
		{"Titel", "ISBN", "Barcode"}, // Header
		{"Titel A", "111", "BC1"},
		{"Titel B", "222", "BC2"},
		{"Titel A", "111", "BC3"}, // Duplicate row in CSV
		{"Titel C", "333", ""},    // Missing barcode -> skip
	}

	isbnToID := map[string]string{}
	titelToID := map[string]string{}

	newTitlesMap, newTitlesOrder := sammleNeueTitel(rows, headerMap, isbnToID, titelToID)

	if len(newTitlesOrder) != 2 {
		t.Fatalf("len(newTitlesOrder) = %d, want 2", len(newTitlesOrder))
	}
	if newTitlesOrder[0] != "111" || newTitlesOrder[1] != "222" {
		t.Errorf("newTitlesOrder = %v", newTitlesOrder)
	}
	if len(newTitlesMap) != 2 {
		t.Fatalf("len(newTitlesMap) = %d, want 2", len(newTitlesMap))
	}
}

func TestSammleExemplare(t *testing.T) {
	headerMap := map[string]int{"titel": 0, "isbn": 1, "barcode": 2, "zustand": 3}
	rows := [][]string{
		{"Titel", "ISBN", "Barcode", "Zustand"}, // Header
		{"Titel A", "111", "BC1", ""},           // ausleihbar
		{"Titel B", "222", "BC2", "verliehen"},  // nicht ausleihbar
		{"Titel C", "333", "BC3", "kaputt"},     // ausleihbar, mit Notiz
		{"Titel D", "444", "", ""},              // missing barcode -> skip
		{"", "555", "BC4", ""},                  // missing titel -> skip
	}

	isbnToID := map[string]string{"111": "id1", "222": "id2", "333": "id3"}
	titelToID := map[string]string{}

	copies := sammleExemplare(rows, headerMap, isbnToID, titelToID)

	if len(copies) != 3 {
		t.Fatalf("len(copies) = %d, want 3", len(copies))
	}

	tests := []struct {
		i             int
		titelID       string
		istAusleihbar bool
		zustandNotiz  string
	}{
		{0, "id1", true, ""},
		{1, "id2", false, "verliehen"},
		{2, "id3", true, "kaputt"},
	}

	for _, tt := range tests {
		if copies[tt.i].TitelID != tt.titelID {
			t.Errorf("copies[%d].TitelID = %q, want %q", tt.i, copies[tt.i].TitelID, tt.titelID)
		}
		if copies[tt.i].IstAusleihbar != tt.istAusleihbar {
			t.Errorf("copies[%d].IstAusleihbar = %v, want %v", tt.i, copies[tt.i].IstAusleihbar, tt.istAusleihbar)
		}
		if copies[tt.i].ZustandNotiz != tt.zustandNotiz {
			t.Errorf("copies[%d].ZustandNotiz = %q, want %q", tt.i, copies[tt.i].ZustandNotiz, tt.zustandNotiz)
		}
	}
}

func TestImportDynamic_EmptyRows(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to init pgxmock: %v", err)
	}
	defer mock.Close()

	mock.ExpectBegin()
	// ladeVorhandeneTitel queries
	mock.ExpectQuery("SELECT id, coalesce\\(isbn, ''\\), titel FROM buecher_titel").
		WillReturnRows(pgxmock.NewRows([]string{"id", "isbn", "titel"}))
	mock.ExpectCommit()
	mock.ExpectRollback() // SafeRollback

	svc := NewImportService(nil, mock)

	headerMap := map[string]int{"titel": 0, "isbn": 1, "barcode": 2}
	rows := [][]string{{"Titel", "ISBN", "Barcode"}} // Only header

	newTitles, newCopies, err := svc.ImportDynamic(context.Background(), rows, headerMap)
	if err != nil {
		t.Errorf("ImportDynamic() error = %v", err)
	}
	if newTitles != 0 {
		t.Errorf("newTitles = %v, want 0", newTitles)
	}
	if newCopies != 0 {
		t.Errorf("newCopies = %v, want 0", newCopies)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestSpaltenWert(t *testing.T) {
	row := []string{"Wert1", " Wert2 ", "Wert3"}
	headerMap := map[string]int{"col1": 0, "col2": 1, "col3": 2, "col4": 3}

	if got := spaltenWert(row, headerMap, "col1"); got != "Wert1" {
		t.Errorf("spaltenWert col1 = %v, want Wert1", got)
	}
	if got := spaltenWert(row, headerMap, "col2"); got != "Wert2" {
		t.Errorf("spaltenWert col2 = %v, want Wert2 (trimmed)", got)
	}
	if got := spaltenWert(row, headerMap, "col4"); got != "" {
		t.Errorf("spaltenWert col4 = %v, want empty string for out of bounds", got)
	}
	if got := spaltenWert(row, headerMap, "notexist"); got != "" {
		t.Errorf("spaltenWert notexist = %v, want empty string", got)
	}
}

// This adds proper unit coverage for import_dynamic.go.

func TestSammleSignaturUpdatesDynamic(t *testing.T) {
	headerMap := map[string]int{"titel": 0, "isbn": 1, "signatur": 2}
	rows := [][]string{
		{"Titel", "ISBN", "Signatur"},
		{"Bekannt", "111", "Sig1"},
		{"Bekannt2", "", "Sig2"},
		{"Neu", "999", "Sig3"},
	}

	isbnToID := map[string]string{"111": "id1"}
	titelToID := map[string]string{repository.NormalisiereTitelKey("Bekannt2"): "id2"}

	updates := sammleSignaturUpdates(rows, headerMap, isbnToID, titelToID)
	if len(updates) != 2 {
		t.Fatalf("len(updates) = %v, want 2", len(updates))
	}
	if updates["id1"] != "Sig1" {
		t.Errorf("updates[id1] = %v, want Sig1", updates["id1"])
	}
	if updates["id2"] != "Sig2" {
		t.Errorf("updates[id2] = %v, want Sig2", updates["id2"])
	}
}

// Die Kategorie-Spalte wird nur dann zum Fach, wenn sie ein Fach IST. Die aus dem
// Littera-PDF gewonnene Bestands-CSV trug dort Standorttexte („Buch Pg/Kaf 078829");
// der Rückfall machte daraus 1.677 „Fächer" samt Systematik-Zeilen (Test-Server,
// 19.08.2026) — im Portal-Reiter Schulbücher stand je Titel eine eigene Fach-Kachel.
func TestFachDerZeile_KategorieNurWennFach(t *testing.T) {
	faelle := []struct{ fach, kategorie, want string }{
		{"Mathematik", "Buch Ma 6/Gri", "Mathematik"}, // Lernmittelsignatur schlägt Kategorie
		{"", "Mathe", "Mathematik"},                   // Kategorie ist ein Fach → kanonisch
		{"", "Deutsch", "Deutsch"},
		{"", "Buch Pg/Kaf 078829", ""},                      // Standorttext ist kein Fach
		{"", "Buch Deu 6/Cha 126 Exemplare 1. Auflage", ""}, // auch mit Fachkürzel darin nicht
		{"", "", ""},
	}
	for _, f := range faelle {
		got := fachDerZeile(&importNewTitle{Fach: f.fach, Kategorie: f.kategorie})
		if got != f.want {
			t.Errorf("fachDerZeile(Fach=%q, Kategorie=%q) = %q, want %q", f.fach, f.kategorie, got, f.want)
		}
	}
}
