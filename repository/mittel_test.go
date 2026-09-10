package repository

import "testing"

func TestMittelGueltig(t *testing.T) {
	for _, m := range []string{MittelLand, MittelSchultraeger} {
		if !MittelGueltig(m) {
			t.Errorf("%q gehört zum Vokabular", m)
		}
	}
	for _, m := range []string{"", "kreis", "Land", "LAND", "schultraeger "} {
		if MittelGueltig(m) {
			t.Errorf("%q darf nicht gültig sein — die Tür muss 400 sagen", m)
		}
	}
}

// Die zweite Kundennummer gilt NUR für den Schulträger-Topf und nur, wenn sie hinterlegt
// ist — leer heißt „dieselbe Nummer", nicht „keine Nummer".
func TestSupplierKundennummerFuer(t *testing.T) {
	ohne := Supplier{Kundennummer: "K-1"}
	mit := Supplier{Kundennummer: "K-1", KundennummerSchultraeger: "B-2"}

	if got := ohne.KundennummerFuer(MittelSchultraeger); got != "K-1" {
		t.Errorf("ohne zweite Nummer: %q, want K-1", got)
	}
	if got := mit.KundennummerFuer(MittelSchultraeger); got != "B-2" {
		t.Errorf("mit zweiter Nummer: %q, want B-2", got)
	}
	if got := mit.KundennummerFuer(MittelLand); got != "K-1" {
		t.Errorf("Land nimmt immer die erste Nummer: %q, want K-1", got)
	}
}
