package littera

import (
	"testing"

	"bibliothek/repository"
)

// Die Platzhalter-Domäne steht an zwei Stellen — hier und in repository.
//
// Hier, weil die Übernahme die Ersatzadressen BAUT (schreiber_personen.go: die
// Leserdatei aus Littera führt keine Adressen, ein Konto braucht aber eine). Dort, weil
// das Zusammenführen sie WIEDERERKENNEN muss: Ein Konto unter dieser Domäne ist kein
// Zugang, sondern Füllmaterial, und darf einem echten weichen.
//
// Zwei Stellen, weil repository dieses Paket nicht importieren darf — littera importiert
// repository, ein Zyklus wäre die Folge. Also hält dieser Test sie zusammen: Wer die
// Domäne hier umbenennt, ohne es dort zu tun, bekommt kein stilles Verhalten, sondern
// einen roten Test. Sonst erkennte das Zusammenführen die Platzhalter nicht mehr und
// bräche wieder ab, wo „das ist dieselbe Person" gebraucht wird.
func TestPlatzhalterDomainIstDieselbe(t *testing.T) {
	if platzhalterDomain != repository.PlatzhalterDomain {
		t.Fatalf("die Übernahme baut %q, das Zusammenführen sucht %q — ein Platzhalter-Konto "+
			"wird dann nicht mehr als solches erkannt",
			platzhalterDomain, repository.PlatzhalterDomain)
	}
}
