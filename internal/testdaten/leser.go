package testdaten

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"bibliothek/repository"
)

// LegeSchuelerAn legt einen Schüler mit dem Nachnamen „Test“ an und liefert seine Kennung.
func LegeSchuelerAn(t *testing.T, pool *pgxpool.Pool, barcode, vorname, klasse string) string {
	t.Helper()
	var id string
	if err := pool.QueryRow(context.Background(),
		`INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr)
		 VALUES ($1, $2, 'Test', $3, 2030) RETURNING id`,
		barcode, vorname, klasse).Scan(&id); err != nil {
		t.Fatalf("Schüler %q anlegen: %v", vorname, err)
	}
	return id
}

// LegeSchuelerMitNamenAn legt einen Schüler mit Nachnamen und Abgangsjahr an und liefert
// seine Kennung.
func LegeSchuelerMitNamenAn(t *testing.T, pool *pgxpool.Pool, barcode, vorname, nachname, klasse string, abgaengerJahr int) string {
	t.Helper()
	var id string
	if err := pool.QueryRow(context.Background(),
		`INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr)
		 VALUES ($1, $2, $3, $4, $5) RETURNING id`,
		barcode, vorname, nachname, klasse, abgaengerJahr).Scan(&id); err != nil {
		t.Fatalf("Schüler %s %s anlegen: %v", vorname, nachname, err)
	}
	return id
}

// LegeLeserAn legt einen Leser der Art an und liefert seine Kennung. Ein leerer Barcode steht als
// NULL in der Tabelle, wie bei einem Kollegen, dem noch keine Ausweisnummer vergeben ist.
func LegeLeserAn(t *testing.T, pool *pgxpool.Pool, barcode, vorname, nachname, art string) string {
	t.Helper()
	var id string
	if err := pool.QueryRow(context.Background(),
		`INSERT INTO leser (barcode_id, vorname, nachname, art) VALUES (NULLIF($1, ''), $2, $3, $4) RETURNING id`,
		barcode, vorname, nachname, art).Scan(&id); err != nil {
		t.Fatalf("Leser %s %s anlegen: %v", vorname, nachname, err)
	}
	return id
}

// LegeAbgaengerAn legt einen Schüler an, der als Abgänger markiert ist.
func LegeAbgaengerAn(t *testing.T, pool *pgxpool.Pool, barcode, vorname, klasse string) string {
	t.Helper()
	var id string
	if err := pool.QueryRow(context.Background(),
		`INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr, ist_abgaenger)
		 VALUES ($1, $2, 'Test', $3, 2030, true) RETURNING id`,
		barcode, vorname, klasse).Scan(&id); err != nil {
		t.Fatalf("Abgänger %q anlegen: %v", vorname, err)
	}
	return id
}

// LUSDSchuelerNeu sind die Angaben für LegeLUSDSchuelerAn. Ein Abgänger entsteht gesperrt, mit
// dem Sperrgrund der Abgänger und einer Karenz, die seit zehn Tagen läuft.
type LUSDSchuelerNeu struct {
	Vorname, Nachname, Klasse, Barcode string
	Geburtsdatum, SchulEintrittAm      *time.Time
	IstAbgaenger                       bool
}

// LegeLUSDSchuelerAn legt einen Schüler an, wie ihn der LUSD-Import hinterlässt: mit
// Geburtsdatum und Schuleintritt, vor einem Jahr bestätigt.
func LegeLUSDSchuelerAn(t *testing.T, pool *pgxpool.Pool, s LUSDSchuelerNeu) string {
	t.Helper()
	var id string
	if err := pool.QueryRow(context.Background(), `
		INSERT INTO schueler (vorname, nachname, klasse, barcode_id, abgaenger_jahr, geburtsdatum,
		                      schul_eintritt_am, ist_abgaenger, lusd_bestaetigt_am, ist_gesperrt, block_reason, abgaenger_seit)
		VALUES ($1,$2,$3,$4,2031,$5,$6,$7,NOW() - interval '1 year',$7,
		        CASE WHEN $7 THEN '`+repository.AbgaengerSperrgrundKarenz+`' END,
		        CASE WHEN $7 THEN NOW() - interval '10 days' END)
		RETURNING id`,
		s.Vorname, s.Nachname, s.Klasse, s.Barcode, s.Geburtsdatum, s.SchulEintrittAm, s.IstAbgaenger).Scan(&id); err != nil {
		t.Fatalf("Schüler %s %s anlegen: %v", s.Vorname, s.Nachname, err)
	}
	return id
}

// LegeKontoAn legt ein Konto mit Rolle und Zustand an und liefert seine Kennung.
func LegeKontoAn(t *testing.T, pool *pgxpool.Pool, vorname, nachname, email, rolle string, aktiv bool) string {
	t.Helper()
	var id string
	if err := pool.QueryRow(context.Background(),
		`INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv) VALUES ($1, $2, $3, $4, $5) RETURNING id`,
		vorname, nachname, email, rolle, aktiv).Scan(&id); err != nil {
		t.Fatalf("Konto %s anlegen: %v", email, err)
	}
	return id
}

// LegeKollegiumskontoAn legt ein Konto des Kollegiums mit der Adresse an und liefert seine
// Kennung; gibt es die Adresse schon, liefert es deren Konto.
func LegeKollegiumskontoAn(t *testing.T, pool *pgxpool.Pool, email string) string {
	t.Helper()
	var id string
	if err := pool.QueryRow(context.Background(), `
		INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv)
		VALUES ('Portal', 'Lehrkraft', $1, 'kollegium', true)
		ON CONFLICT (email) DO UPDATE SET vorname = EXCLUDED.vorname
		RETURNING id`, email).Scan(&id); err != nil {
		t.Fatalf("Lehrkraft %s anlegen: %v", email, err)
	}
	return id
}

// LegeBearbeiterAn legt das eine Admin-Konto an, das in allen Tests als Bearbeiter handelt, und
// liefert seine Kennung: Ein Protokolleintrag zeigt auf ein Konto.
func LegeBearbeiterAn(t *testing.T, pool *pgxpool.Pool) string {
	t.Helper()
	var id string
	err := pool.QueryRow(context.Background(), `
		INSERT INTO benutzer (email, vorname, nachname, rolle, aktiv)
		VALUES ('mahnlauf@schule.invalid', 'Mahn', 'Lauf', 'admin', true)
		ON CONFLICT (email) DO UPDATE SET aktiv = true
		RETURNING id`).Scan(&id)
	if err != nil {
		t.Fatalf("Admin anlegen: %v", err)
	}
	return id
}

// SetzeKlassenleitung trägt die Adresse der Klassenleitung einer Klasse ein.
func SetzeKlassenleitung(t *testing.T, pool *pgxpool.Pool, klasse, email string) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `
		INSERT INTO klassen_lehrer_mapping (klasse, lehrer_email) VALUES ($1, $2)
		ON CONFLICT (klasse) DO UPDATE SET lehrer_email = EXCLUDED.lehrer_email`, klasse, email)
	if err != nil {
		t.Fatalf("Klassenleitung für %s eintragen: %v", klasse, err)
	}
}

// ZusammenfuehrenAuftrag baut den Auftrag, die Quelle in das Ziel zusammenzuführen, ohne
// Bearbeiter.
func ZusammenfuehrenAuftrag(ziel, quelle string) repository.ZusammenfuehrenAuftrag {
	return repository.ZusammenfuehrenAuftrag{ZielID: ziel, QuelleID: quelle}
}

// LegeAdminKontoAn legt ein aktives Admin-Konto mit dieser Adresse an und liefert seine
// Kennung. Die Adresse gehört dem Test: Zwei Tests teilen sich so kein Konto und zählen nicht
// die Protokolleinträge des anderen.
func LegeAdminKontoAn(t *testing.T, pool *pgxpool.Pool, email, vorname, nachname string) string {
	t.Helper()
	var id string
	if err := pool.QueryRow(context.Background(), `
		INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv)
		VALUES ($1, $2, $3, 'admin', true)
		ON CONFLICT (lower(email)) DO UPDATE SET aktiv = true
		RETURNING id`, vorname, nachname, email).Scan(&id); err != nil {
		t.Fatalf("Konto %s anlegen: %v", email, err)
	}
	return id
}

// LadeSchuelerStatus liest, ob ein Schüler gesperrt und als Abgänger geführt ist, dazu seinen
// Sperrgrund und Vornamen.
func LadeSchuelerStatus(t *testing.T, pool *pgxpool.Pool, id string) (gesperrt, abgaenger bool, sperrgrund *string, vorname string) {
	t.Helper()
	if err := pool.QueryRow(context.Background(),
		`SELECT ist_gesperrt, ist_abgaenger, block_reason, vorname FROM schueler WHERE id = $1`, id).
		Scan(&gesperrt, &abgaenger, &sperrgrund, &vorname); err != nil {
		t.Fatalf("Status lesen: %v", err)
	}
	return
}
