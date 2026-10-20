package jobs

import (
	"context"
	"os"
	"strings"
	"testing"

	"bibliothek/internal/testdaten"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Die Aufbewahrung an Postgres: Alt fällt weg, Neu bleibt, und die Löschung hinterlässt genau
// einen Eintrag mit den Zahlen. Das SQL mit Intervallen (make_interval über beide Tabellen mit
// verschiedenen Spalten für die Zeit) ist genau die Sorte, die eine Attrappe nur nachspielen
// würde.
func TestProtokollAufbewahrung_LoeschtAltesUndProtokolliert(t *testing.T) {
	adminDSN := os.Getenv(datenbankVariable)
	if adminDSN == "" {
		t.Skipf("%s nicht gesetzt — Test übersprungen", datenbankVariable)
	}

	_, dsn := legeProbeDatenbankAn(t, adminDSN, "retention")
	befuelleQuelle(t, dsn)

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("Pool: %v", err)
	}
	t.Cleanup(pool.Close)

	// Ein Bearbeiter für die Spalten mit Fremdschlüssel, dazu je Tabelle ein alter und ein junger Eintrag.
	benutzerID := testdaten.LegeKontoAn(t, pool, "Retention", "Kraft", "ret@example.org", "admin", true)
	legeAn := func(abfrage string, alterMonate int) {
		t.Helper()
		if _, err := pool.Exec(ctx, abfrage, benutzerID, alterMonate); err != nil {
			t.Fatalf("Seed (%d Monate): %v", alterMonate, err)
		}
	}
	legeAn(`INSERT INTO audit_log (tabelle, aktion, datensatz_id, bearbeiter_id, timestamp)
	      VALUES ('schueler', 'UPDATE', gen_random_uuid(), $1, now() - make_interval(months => $2))`, 30)
	legeAn(`INSERT INTO audit_log (tabelle, aktion, datensatz_id, bearbeiter_id, timestamp)
	      VALUES ('schueler', 'UPDATE', gen_random_uuid(), $1, now() - make_interval(months => $2))`, 3)
	legeAn(`INSERT INTO audit_logs (admin_id, aktion, ip_adresse, zeitstempel)
	      VALUES ($1, 'LOGIN', '203.0.113.7', now() - make_interval(months => $2))`, 30)
	legeAn(`INSERT INTO audit_logs (admin_id, aktion, ip_adresse, zeitstempel)
	      VALUES ($1, 'LOGIN', '203.0.113.7', now() - make_interval(months => $2))`, 3)

	z := NewZeitplan(pool, repository.NewProtokollAbfragen(pool))
	z.fuehreProtokollAufbewahrungAus(ctx)

	zaehle := func(abfrage string) int {
		t.Helper()
		var n int
		if err := pool.QueryRow(ctx, abfrage).Scan(&n); err != nil {
			t.Fatalf("Zählen: %v", err)
		}
		return n
	}
	// Vorgabe 24 Monate (keine Einstellung gesetzt): 30 Monate alt fällt, 3 bleibt.
	if n := zaehle(`SELECT count(*) FROM audit_log WHERE aktion = 'UPDATE'`); n != 1 {
		t.Errorf("audit_log: %d UPDATE-Einträge, erwartet 1 (der junge)", n)
	}
	if n := zaehle(`SELECT count(*) FROM audit_logs WHERE aktion = 'LOGIN'`); n != 1 {
		t.Errorf("audit_logs: %d LOGIN-Einträge, erwartet 1 (der junge)", n)
	}

	// Genau eine Zeile RETENTION mit beiden Zahlen.
	var details string
	if err := pool.QueryRow(ctx,
		`SELECT details::text FROM audit_log WHERE aktion = 'RETENTION'`).Scan(&details); err != nil {
		t.Fatalf("Meta-Eintrag fehlt: %v", err)
	}
	for _, muss := range []string{`"geloescht_audit_log": 1`, `"geloescht_admin_log": 1`, `"frist_monate": 24`} {
		if !enthaeltJSON(details, muss) {
			t.Errorf("Meta-Eintrag ohne %s: %s", muss, details)
		}
	}

	// Untergrenze: Eine versehentliche 0 in der Einstellung darf nicht alles wegräumen. Geprüft
	// wird über den Weg, den der Lauf nimmt (Einstellungen lesen, dann die Untergrenze anwenden),
	// nicht über eine eigene Abfrage daneben.
	if _, err := pool.Exec(ctx, `INSERT INTO system_einstellungen (schluessel, wert)
		VALUES ('audit_aufbewahrung_monate', '0')
		ON CONFLICT (schluessel) DO UPDATE SET wert = excluded.wert`); err != nil {
		t.Fatalf("Einstellung setzen: %v", err)
	}
	einst, err := repository.NewEinstellungsAbfragen(pool).Lade(ctx)
	if err != nil {
		t.Fatalf("Einstellungen lesen: %v", err)
	}
	if m := repository.ProtokollAufbewahrungMonateOderVorgabe(einst.ProtokollAufbewahrungMonate); m != repository.ProtokollAufbewahrungMonateVorgabe {
		t.Errorf("Frist bei Fehlkonfiguration 0 = %d, erwartet Vorgabe %d", m, repository.ProtokollAufbewahrungMonateVorgabe)
	}
	// Und ein gültiger Wert muss auch wirklich ankommen, sonst bewiese der Test nur, dass immer
	// die Vorgabe herauskommt, und die Einstellung wäre wirkungslos.
	if _, err := pool.Exec(ctx, `INSERT INTO system_einstellungen (schluessel, wert)
		VALUES ('audit_aufbewahrung_monate', '9')
		ON CONFLICT (schluessel) DO UPDATE SET wert = excluded.wert`); err != nil {
		t.Fatalf("Einstellung setzen: %v", err)
	}
	if einst, err = repository.NewEinstellungsAbfragen(pool).Lade(ctx); err != nil {
		t.Fatalf("Einstellungen lesen: %v", err)
	}
	if m := repository.ProtokollAufbewahrungMonateOderVorgabe(einst.ProtokollAufbewahrungMonate); m != 9 {
		t.Errorf("eingestellte Frist 9 kam als %d an — die Einstellung wirkt nicht", m)
	}
}

// enthaeltJSON vergleicht ohne Rücksicht auf Leerzeichen in der Ausgabe von JSON.
func enthaeltJSON(text, teil string) bool {
	entferne := strings.NewReplacer(" ", "")
	return strings.Contains(entferne.Replace(text), entferne.Replace(teil))
}
