package testdaten

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"bibliothek/repository"
)

// SeedSchueler legt einen Schüler mit dem Nachnamen „Test“ an.
func SeedSchueler(t *testing.T, pool *pgxpool.Pool, barcode, vorname, klasse string) string {
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

// SchuelerAnlegen legt einen Schüler mit dem Nachnamen „Testschueler“ an.
func SchuelerAnlegen(t *testing.T, pool *pgxpool.Pool, vorname, klasse, barcode string) string {
	t.Helper()
	var id string
	err := pool.QueryRow(context.Background(), `
		INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr)
		VALUES ($1, $2, 'Testschueler', $3, 2030) RETURNING id`,
		barcode, vorname, klasse).Scan(&id)
	if err != nil {
		t.Fatalf("Schüler %s anlegen: %v", vorname, err)
	}
	return id
}

// SeedAbgaengerRet legt einen Schüler an, der als Abgänger markiert ist.
func SeedAbgaengerRet(t *testing.T, pool *pgxpool.Pool, barcode, vorname, klasse string) string {
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

// UmbSchueler sind die Angaben für LegeUmbSchuelerAn. Ein Abgänger entsteht gesperrt, mit dem
// Sperrgrund der Abgänger und einer Karenz, die seit zehn Tagen läuft.
type UmbSchueler struct {
	Vorname, Nachname, Klasse, Barcode string
	Geb, Eintritt                      *time.Time
	Abgaenger                          bool
}

// LegeUmbSchuelerAn legt einen Schüler mit Geburtsdatum und Eintritt an, vom LUSD-Import vor
// einem Jahr bestätigt.
func LegeUmbSchuelerAn(t *testing.T, pool *pgxpool.Pool, s UmbSchueler) string {
	t.Helper()
	var id string
	if err := pool.QueryRow(context.Background(), `
		INSERT INTO schueler (vorname, nachname, klasse, barcode_id, abgaenger_jahr, geburtsdatum,
		                      schul_eintritt_am, ist_abgaenger, lusd_bestaetigt_am, ist_gesperrt, block_reason, abgaenger_seit)
		VALUES ($1,$2,$3,$4,2031,$5,$6,$7,NOW() - interval '1 year',$7,
		        CASE WHEN $7 THEN '`+repository.AbgaengerSperrgrundKarenz+`' END,
		        CASE WHEN $7 THEN NOW() - interval '10 days' END)
		RETURNING id`,
		s.Vorname, s.Nachname, s.Klasse, s.Barcode, s.Geb, s.Eintritt, s.Abgaenger).Scan(&id); err != nil {
		t.Fatalf("Schüler %s %s anlegen: %v", s.Vorname, s.Nachname, err)
	}
	return id
}

// SeedPortalLehrkraft legt ein Konto des Kollegiums an.
func SeedPortalLehrkraft(t *testing.T, pool *pgxpool.Pool, email string) string {
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

// AdminFuerAudit legt das Admin-Konto an, auf das ein Protokolleintrag zeigt.
func AdminFuerAudit(t *testing.T, pool *pgxpool.Pool) string {
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

// Klassenleitung trägt die Adresse der Klassenleitung einer Klasse ein.
func Klassenleitung(t *testing.T, pool *pgxpool.Pool, klasse, email string) {
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

// AdminKonto legt ein aktives Admin-Konto mit dieser Adresse an und nennt seine Kennung. Die
// Adresse gehört dem Test: Zwei Tests teilen sich so kein Konto und zählen nicht die
// Protokolleinträge des anderen.
func AdminKonto(t *testing.T, pool *pgxpool.Pool, email, vorname, nachname string) string {
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

// LeseSchuelerStatus liest, ob ein Schüler gesperrt und als Abgänger geführt ist, dazu seinen
// Sperrgrund und Vornamen.
func LeseSchuelerStatus(t *testing.T, pool *pgxpool.Pool, id string) (gesperrt, abgaenger bool, reason *string, vorname string) {
	t.Helper()
	if err := pool.QueryRow(context.Background(),
		`SELECT ist_gesperrt, ist_abgaenger, block_reason, vorname FROM schueler WHERE id = $1`, id).
		Scan(&gesperrt, &abgaenger, &reason, &vorname); err != nil {
		t.Fatalf("Status lesen: %v", err)
	}
	return
}
