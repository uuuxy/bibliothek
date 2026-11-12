package littera

// Das Ergebnis der letzten Übernahme, abgelegt in system_einstellungen. Der Befehl läuft am Server
// und nicht im Programm; ohne diese Ablage erführen Selbstprüfung und Alarm-Mail nie, ob er
// vollständig war (internal/bereitschaft, pruefeLitteraUebernahme).

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"bibliothek/db"
	"bibliothek/repository"
)

// ErgebnisSchluessel ist der Schlüssel des Ergebnisses in system_einstellungen.
const ErgebnisSchluessel = "littera_uebernahme_ergebnis"

// Ergebnis ist, was die Selbstprüfung über den letzten schreibenden Lauf wissen muss: wann, ob
// abgebrochen oder vollständig, wie viele Fehler und Warnungen das Protokoll zählt und wo es
// liegt. Namen stehen nicht darin; die Zeilen dazu stehen im Protokoll am Server.
type Ergebnis struct {
	Zeitpunkt    time.Time `json:"zeitpunkt"`
	Abgebrochen  bool      `json:"abgebrochen"`
	Vollstaendig bool      `json:"vollstaendig"`
	Fehler       int       `json:"fehler"`
	Warnungen    int       `json:"warnungen"`
	Protokoll    string    `json:"protokoll"`
}

// ErgebnisAus fasst den Bericht eines Laufs, mit derselben Regel wie der Rückgabewert des
// Befehls (Bericht.IstVollstaendig).
func ErgebnisAus(b Bericht, zeitpunkt time.Time, protokoll string) Ergebnis {
	return Ergebnis{
		Zeitpunkt:    zeitpunkt,
		Abgebrochen:  b.Abbruch != nil,
		Vollstaendig: b.IstVollstaendig(),
		Fehler:       b.Fehler,
		Warnungen:    b.Warnungen,
		Protokoll:    protokoll,
	}
}

// SchreibeErgebnis legt das Ergebnis ab; ein späterer Lauf ersetzt es.
func SchreibeErgebnis(ctx context.Context, pool db.Pool, e Ergebnis) error {
	wert, err := json.Marshal(e)
	if err != nil {
		return fmt.Errorf("ergebnis der übernahme nicht serialisierbar: %w", err)
	}
	return repository.NewBetriebszustandAbfragen(pool).SchreibeEinstellungswert(ctx, ErgebnisSchluessel, string(wert))
}
