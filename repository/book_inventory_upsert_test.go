package repository

import (
	"testing"

	"github.com/jackc/pgx/v5"
)

const (
	testQInsert = "INSERT"
	testQUpdate = "UPDATE"
)

func neuerUpsertContext() *titelUpsertContext {
	return &titelUpsertContext{
		isbnToID:  map[string]string{},
		titelToID: map[string]string{},
		seenISBN:  map[string]bool{},
		seenTitel: map[string]bool{},
	}
}

// Regressionstest für den Dubletten-Bug beim Katalogisat-Import: Die Bestands-CSV
// legt Titel OHNE ISBN an. Kommt danach das MAB2-XML mit ISBN, muss der Datensatz
// über den TITEL gematcht werden (UPDATE mit ISBN-Nachtrag) — vor dem Fix wurde
// nur über die ISBN gematcht und für jeden CSV-Titel eine Dublette angelegt.
func TestQueueTitelUpsert_TitelFallbackBeiUnbekannterISBN(t *testing.T) {
	c := neuerUpsertContext()
	c.titelToID["LMF-Deutschbuch 9"] = "id-bestand"

	batch := &pgx.Batch{}
	ok := queueTitelUpsert(batch, BookTitle{
		Titel: "LMF-Deutschbuch 9", ISBN: "9783060619000", Signatur: "De 9",
	}, c, testQInsert, testQUpdate)

	if !ok {
		t.Fatal("Titel wurde übersprungen, erwartet war ein UPDATE über den Titel-Fallback")
	}
	q := batch.QueuedQueries[0]
	if q.SQL != testQUpdate {
		t.Fatalf("SQL = %q, want UPDATE (Titel-Fallback statt INSERT-Dublette)", q.SQL)
	}
	if q.Arguments[0] != "id-bestand" {
		t.Errorf("Update-ID = %v, want id-bestand", q.Arguments[0])
	}
	if q.Arguments[6] != "9783060619000" {
		t.Errorf("ISBN-Nachtrag fehlt im Update: args = %v", q.Arguments)
	}
}

// Eine Datei mit ISBN-Variante und ISBN-loser Variante desselben Titels darf nur
// EINEN Insert erzeugen.
func TestQueueTitelUpsert_InBatchDedupUeberVarianten(t *testing.T) {
	c := neuerUpsertContext()
	batch := &pgx.Batch{}

	if ok := queueTitelUpsert(batch, BookTitle{Titel: "Faust", ISBN: "978-1"}, c, testQInsert, testQUpdate); !ok {
		t.Fatal("erster Datensatz muss eingereiht werden")
	}
	if ok := queueTitelUpsert(batch, BookTitle{Titel: "Faust"}, c, testQInsert, testQUpdate); ok {
		t.Error("ISBN-lose Variante desselben Titels muss als In-Batch-Dublette übersprungen werden")
	}
	if ok := queueTitelUpsert(batch, BookTitle{Titel: "Faust", ISBN: "978-1"}, c, testQInsert, testQUpdate); ok {
		t.Error("identische ISBN muss als In-Batch-Dublette übersprungen werden")
	}
	if len(batch.QueuedQueries) != 1 {
		t.Errorf("QueuedQueries = %d, want 1", len(batch.QueuedQueries))
	}
}

// Littera kodiert Anführungszeichen je nach Exportweg unterschiedlich
// (PDF-CSV: `""South Africa"""`, XML: `"South Africa"`). Der Matching-Schlüssel
// muss beide Varianten auf denselben Wert abbilden — sonst legt jeder Import
// eine Dublette an. Der gespeicherte Titel bleibt davon unberührt.
func TestNormalisiereTitelKey(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{`"Kein Bock auf Lernen?"`, "Kein Bock auf Lernen?"},
		{`""Kein Bock auf Lernen?"""`, "Kein Bock auf Lernen?"},
		{`LMF-Green Line - Topic zum country of reference ""South Africa`, "LMF-Green Line - Topic zum country of reference South Africa"},
		{`LMF-"Green Line - Topic zum country of reference "South Africa"`, "LMF-Green Line - Topic zum country of reference South Africa"},
		{"  Faust   Teil 1 ", "Faust Teil 1"},
		{"Faust", "Faust"},
	}
	for _, tt := range tests {
		if got := NormalisiereTitelKey(tt.in); got != tt.want {
			t.Errorf("NormalisiereTitelKey(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

// Titel, die sich nur in der Anführungszeichen-Kodierung unterscheiden, müssen
// dieselbe Bestandszeile treffen (UPDATE statt INSERT-Dublette).
func TestQueueTitelUpsert_QuoteVariantenMatchen(t *testing.T) {
	c := neuerUpsertContext()
	c.titelToID[NormalisiereTitelKey(`"Kein Bock auf Lernen?"`)] = "id-quoted"

	batch := &pgx.Batch{}
	ok := queueTitelUpsert(batch, BookTitle{Titel: `Kein Bock auf Lernen?`}, c, testQInsert, testQUpdate)

	if !ok || batch.QueuedQueries[0].SQL != testQUpdate || batch.QueuedQueries[0].Arguments[0] != "id-quoted" {
		t.Errorf("Quote-Variante muss die bestehende Zeile updaten, got ok=%v %v", ok, batch.QueuedQueries[0])
	}
}

// ISBN-Match hat Vorrang vor dem Titel-Match: existieren beide, gewinnt die ISBN-Zeile.
func TestQueueTitelUpsert_ISBNVorTitel(t *testing.T) {
	c := neuerUpsertContext()
	c.isbnToID["978-1"] = "id-isbn"
	c.titelToID["Faust"] = "id-titel"

	batch := &pgx.Batch{}
	queueTitelUpsert(batch, BookTitle{Titel: "Faust", ISBN: "978-1"}, c, testQInsert, testQUpdate)

	if q := batch.QueuedQueries[0]; q.SQL != testQUpdate || q.Arguments[0] != "id-isbn" {
		t.Errorf("erwartet UPDATE auf id-isbn, got %q %v", q.SQL, q.Arguments[0])
	}
}
