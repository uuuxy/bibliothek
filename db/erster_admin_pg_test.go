package db

import (
	"context"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
)

// LegeErstenAdminAn legt beim ersten Start das eine Konto an, über das alle weiteren entstehen.
//
// Jeder Fall läuft in einer Transaktion, die zurückgerollt wird: Für den ersten Start muss die
// Tabelle leer sein, und die Pakete teilen sich eine Datenbank.

// txAlsPool reicht die Abfragen von LegeErstenAdminAn an eine Transaktion.
type txAlsPool struct {
	Pool
	tx pgx.Tx
}

func (p txAlsPool) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	return p.tx.QueryRow(ctx, sql, args...)
}

// ersterStart leert die Konten in der Transaktion und setzt die Variable.
func ersterStart(t *testing.T, tx pgx.Tx, adresse string) *Verbindung {
	t.Helper()
	if _, err := tx.Exec(context.Background(), `TRUNCATE benutzer CASCADE`); err != nil {
		t.Fatalf("Konten leeren: %v", err)
	}
	t.Setenv("INITIAL_ADMIN_EMAIL", adresse)
	return &Verbindung{Pool: txAlsPool{tx: tx}}
}

func zaehleKonten(t *testing.T, tx pgx.Tx, bedingung string, args ...any) int {
	t.Helper()
	var n int
	if err := tx.QueryRow(context.Background(),
		`SELECT count(*) FROM benutzer WHERE `+bedingung, args...).Scan(&n); err != nil {
		t.Fatalf("Konten zählen (%s): %v", bedingung, err)
	}
	return n
}

// findetDieAnmeldung nimmt die Bedingung, mit der die Anmeldung ein Konto sucht
// (auth/anmelden.go).
func findetDieAnmeldung(t *testing.T, tx pgx.Tx, getippt string) bool {
	t.Helper()
	return zaehleKonten(t, tx, `LOWER(email) = LOWER($1) AND rolle = 'admin' AND aktiv`, getippt) == 1
}

func TestLegeErstenAdminAn_ErsterStartLegtDasEineAdminKontoAn(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()
	inTx(t, pool, func(tx pgx.Tx) {
		d := ersterStart(t, tx, "erste.leitung@schule.example")
		if err := d.LegeErstenAdminAn(ctx); err != nil {
			t.Fatalf("LegeErstenAdminAn: %v", err)
		}
		if n := zaehleKonten(t, tx, `true`); n != 1 {
			t.Fatalf("%d Konten nach dem ersten Start, erwartet 1", n)
		}
		if !findetDieAnmeldung(t, tx, "erste.leitung@schule.example") {
			t.Error("die Anmeldung findet kein aktives Admin-Konto zu der Adresse")
		}
		// Das Konto hängt an einer Leserzeile (Trigger trg_benutzer_hat_leserzeile).
		var art string
		if err := tx.QueryRow(ctx, `SELECT l.art FROM benutzer b JOIN leser l ON l.id = b.leser_id`).Scan(&art); err != nil {
			t.Fatalf("Leserzeile des Kontos: %v", err)
		}
		if art != "lehrkraft" {
			t.Errorf("Leserzeile der Art %q, erwartet lehrkraft", art)
		}
		// Ein aktives Konto bekommt seine Ausweisnummer mit dem Commit
		// (trg_aktives_konto_hat_ausweis ist aufgeschoben); hier wird er vorgezogen.
		if _, err := tx.Exec(ctx, `SET CONSTRAINTS ALL IMMEDIATE`); err != nil {
			t.Fatalf("aufgeschobene Trigger auslösen: %v", err)
		}
		var nummer *string
		if err := tx.QueryRow(ctx, `SELECT l.barcode_id FROM benutzer b JOIN leser l ON l.id = b.leser_id`).Scan(&nummer); err != nil {
			t.Fatalf("Ausweisnummer des Kontos: %v", err)
		}
		if nummer == nil || !strings.HasPrefix(*nummer, "A-") {
			t.Errorf("das Admin-Konto trägt keine Ausweisnummer der Form A-…: %v", nummer)
		}

		// Jeder weitere Start findet Konten vor und legt nichts an, auch mit anderer Adresse oder
		// ohne eine: Ein System mit Konten braucht die Variable nicht mehr.
		for _, adresse := range []string{"erste.leitung@schule.example", "zweite@schule.example", ""} {
			t.Setenv("INITIAL_ADMIN_EMAIL", adresse)
			if err := d.LegeErstenAdminAn(ctx); err != nil {
				t.Fatalf("zweiter Start (%s): %v", adresse, err)
			}
			if n := zaehleKonten(t, tx, `true`); n != 1 {
				t.Errorf("nach einem weiteren Start mit %s: %d Konten, erwartet 1", adresse, n)
			}
		}
	})
}

// Die Adresse kommt aus einer Datei, die ein Mensch von Hand schreibt. Gespeichert wird
// sie so, wie die Anmeldung sucht; sonst gäbe es ein Admin-Konto, in das niemand kommt,
// und die Variable wirkt danach nicht mehr.
func TestLegeErstenAdminAn_AdresseInDerFormDerAnmeldung(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()
	inTx(t, pool, func(tx pgx.Tx) {
		d := ersterStart(t, tx, "  Erika.Muster@Schule.Example ")
		if err := d.LegeErstenAdminAn(ctx); err != nil {
			t.Fatalf("LegeErstenAdminAn: %v", err)
		}
		if !findetDieAnmeldung(t, tx, "erika.muster@schule.example") {
			t.Error("die Anmeldung findet das Konto nicht: die Adresse steht nicht in ihrer Normalform in der Zeile")
		}
		var gespeichert string
		if err := tx.QueryRow(ctx, `SELECT email FROM benutzer`).Scan(&gespeichert); err != nil {
			t.Fatal(err)
		}
		if gespeichert != "erika.muster@schule.example" {
			t.Errorf("gespeichert %q, erwartet die Adresse klein und ohne Leerraum", gespeichert)
		}
	})
}

// Fehlt die Adresse oder hat der Wert nicht ihre Form, bricht der erste Start ab und legt
// nichts an: Die Tabelle bleibt leer, die berichtigte Variable wirkt beim nächsten Start.
func TestLegeErstenAdminAn_KeineAdresseBrichtDenStartAb(t *testing.T) {
	pool := pgTestPool(t)
	for _, adresse := range []string{"", "   ", "admin", "admin@", "@schule.example", "erika muster@schule.example", "Erika <erika@schule.example>"} {
		inTx(t, pool, func(tx pgx.Tx) {
			d := ersterStart(t, tx, adresse)
			err := d.LegeErstenAdminAn(context.Background())
			if err == nil {
				t.Errorf("Adresse %q: kein Fehler", adresse)
			} else if !strings.Contains(err.Error(), "INITIAL_ADMIN_EMAIL") {
				t.Errorf("Adresse %q: der Fehler nennt die Variable nicht: %v", adresse, err)
			}
			if n := zaehleKonten(t, tx, `true`); n != 0 {
				t.Errorf("Adresse %q: %d Konten angelegt, erwartet 0", adresse, n)
			}
		})
	}
}
