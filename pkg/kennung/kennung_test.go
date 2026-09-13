package kennung

import "testing"

func TestIstUUID(t *testing.T) {
	faelle := []struct {
		wert  string
		ist   bool
		warum string
	}{
		{"7c9e6679-7425-40de-944b-e07fc1f90ae7", true, "kanonische Form"},
		{"7C9E6679-7425-40DE-944B-E07FC1F90AE7", true, "Großbuchstaben nimmt Postgres an"},
		{"", false, "leer"},
		{"x", false, "kein Format"},
		{"urn:uuid:7c9e6679-7425-40de-944b-e07fc1f90ae7", false, "besteht uuid.Validate, Postgres weist sie ab"},
		{"{7c9e6679-7425-40de-944b-e07fc1f90ae7}", false, "Postgres nähme sie, erzeugt wird sie nirgends"},
		{"7c9e6679742540de944be07fc1f90ae7", false, "ohne Bindestriche"},
		{"7c9e6679-7425-40de-944b-e07fc1f90ae7\n", false, "Zeilenende hinter der Kennung"},
		{" 7c9e6679-7425-40de-944b-e07fc1f90ae7", false, "Leerzeichen davor"},
		{"7c9e6679x7425x40dex944bxe07fc1f90ae7", false, "andere Trenner an den Stellen der Bindestriche"},
		{"7c9e6679-7425-40de-944b-e07fc1f90ae", false, "eine Ziffer zu wenig"},
		{"7c9e6679-7425-40de-944b-e07fc1f90aeg", false, "g ist keine Hex-Ziffer"},
	}
	for _, f := range faelle {
		if got := IstUUID(f.wert); got != f.ist {
			t.Errorf("IstUUID(%q) = %v, erwartet %v (%s)", f.wert, got, f.ist, f.warum)
		}
	}
}
