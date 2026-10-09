package repository

import (
	"testing"

	"bibliothek/pkg/mitteltopf"
)

// Die zweite Kundennummer gilt nur für den Topf des Schulträgers und nur, wenn sie hinterlegt
// ist: Leer heißt „dieselbe Nummer", nicht „keine Nummer".
func TestSupplierKundennummerFuer(t *testing.T) {
	ohne := Supplier{Kundennummer: "K-1"}
	mit := Supplier{Kundennummer: "K-1", KundennummerSchultraeger: "B-2"}

	if got := ohne.KundennummerFuer(mitteltopf.Schultraeger); got != "K-1" {
		t.Errorf("ohne zweite Nummer: %q, want K-1", got)
	}
	if got := mit.KundennummerFuer(mitteltopf.Schultraeger); got != "B-2" {
		t.Errorf("mit zweiter Nummer: %q, want B-2", got)
	}
	if got := mit.KundennummerFuer(mitteltopf.Land); got != "K-1" {
		t.Errorf("Land nimmt immer die erste Nummer: %q, want K-1", got)
	}
}
