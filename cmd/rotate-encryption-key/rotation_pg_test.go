package main

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"bibliothek/internal/crypto"
	"bibliothek/internal/pgtest"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Der Schlüsselwechsel läuft genau einmal gegen den echten Bestand — Schülerfotos und
// das SMTP-Passwort. Was er falsch macht, macht er dauerhaft: Ein Foto, das mit dem neuen
// Schlüssel geschrieben wurde, während das Passwort noch am alten hängt, ist ein Bestand
// mit zwei Schlüsseln, und keiner der beiden liest ihn ganz. Bis zum 21.09.2026 hatte das
// Werkzeug keinen Test.
//
// Warum echtes Postgres: Die drei Zusicherungen (beide Tabellen in EINER Transaktion,
// Probelauf schreibt nichts, ein unlesbarer Datensatz rollt alles zurück) leben im
// Verhalten der Transaktion — pgxmock spielt Antworten nach, kennt aber keinen Rollback.
// Dazu kommt die Rückschreibung über die Kennung als Text (`id::text` beim Lesen,
// `WHERE id = $2` beim Schreiben): Ob Postgres '1' wieder als SERIAL nimmt, entscheidet
// nur Postgres.

// Drei Schlüssel im Hex-Format (64 Zeichen), wie sie der Aufruf `-neu …` erwartet.
const (
	altSchluesselHex   = "00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff"
	neuSchluesselHex   = "ffeeddccbbaa99887766554433221100ffeeddccbbaa99887766554433221100"
	fremdSchluesselHex = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
)

var (
	fotoKlartext     = []byte("RIFF....WEBP-Probe eines Schülerfotos")
	passwortKlartext = []byte("smtp-geheimnis-vor-dem-wechsel")
)

func schluessel(t *testing.T, hex string) []byte {
	t.Helper()
	k, err := crypto.SchluesselAus(hex)
	if err != nil {
		t.Fatalf("Schlüssel %q: %v", hex, err)
	}
	return k
}

func verschluesselt(t *testing.T, key, klartext []byte) []byte {
	t.Helper()
	b, err := crypto.EncryptMit(key, klartext)
	if err != nil {
		t.Fatalf("verschlüsseln: %v", err)
	}
	return b
}

// rotationsProbe räumt die beiden betroffenen Tabellen und legt eine Leserzeile an, an
// der ein Foto hängen kann. Aufgeräumt wird nur, was diese Tests selbst hinterlassen;
// schema.sql hat die eine Zeile der Mail-Konfiguration bereits angelegt.
func rotationsProbe(t *testing.T) (*pgxpool.Pool, string) {
	t.Helper()
	pool := pgtest.Pool(t)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `TRUNCATE schueler_fotos`); err != nil {
		t.Fatalf("Fotos leeren: %v", err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM leser WHERE barcode_id LIKE 'ROT-%'`); err != nil {
		t.Fatalf("Probe-Leser löschen: %v", err)
	}
	if _, err := pool.Exec(ctx, `UPDATE mail_settings_config SET smtp_password_encrypted = NULL WHERE id = 1`); err != nil {
		t.Fatalf("Passwort leeren: %v", err)
	}
	var leserID string
	if err := pool.QueryRow(ctx, `INSERT INTO leser (barcode_id, vorname, nachname, art)
		VALUES ('ROT-1', 'Probe', 'Schlüsselwechsel', 'lehrkraft') RETURNING id`).Scan(&leserID); err != nil {
		t.Fatalf("Probe-Leser anlegen: %v", err)
	}
	return pool, leserID
}

func schreibeFoto(t *testing.T, pool *pgxpool.Pool, leserID string, key []byte) {
	t.Helper()
	if _, err := pool.Exec(context.Background(),
		`INSERT INTO schueler_fotos (schueler_id, foto_encrypted) VALUES ($1, $2)`,
		leserID, verschluesselt(t, key, fotoKlartext)); err != nil {
		t.Fatalf("Foto schreiben: %v", err)
	}
}

func schreibePasswort(t *testing.T, pool *pgxpool.Pool, key []byte) {
	t.Helper()
	if _, err := pool.Exec(context.Background(),
		`UPDATE mail_settings_config SET smtp_password_encrypted = $1 WHERE id = 1`,
		verschluesselt(t, key, passwortKlartext)); err != nil {
		t.Fatalf("Passwort schreiben: %v", err)
	}
}

func liesFoto(t *testing.T, pool *pgxpool.Pool, leserID string) []byte {
	t.Helper()
	var b []byte
	if err := pool.QueryRow(context.Background(),
		`SELECT foto_encrypted FROM schueler_fotos WHERE schueler_id = $1`, leserID).Scan(&b); err != nil {
		t.Fatalf("Foto lesen: %v", err)
	}
	return b
}

func liesPasswort(t *testing.T, pool *pgxpool.Pool) []byte {
	t.Helper()
	var b []byte
	if err := pool.QueryRow(context.Background(),
		`SELECT smtp_password_encrypted FROM mail_settings_config WHERE id = 1`).Scan(&b); err != nil {
		t.Fatalf("Passwort lesen: %v", err)
	}
	return b
}

// TestRotiere_SchluesseltBeideTabellenUm: Nach dem Lauf liest der neue Schlüssel beide
// Werte, der alte keinen mehr — und gezählt sind genau die zwei Datensätze.
func TestRotiere_SchluesseltBeideTabellenUm(t *testing.T) {
	pool, leserID := rotationsProbe(t)
	alt, neu := schluessel(t, altSchluesselHex), schluessel(t, neuSchluesselHex)
	schreibeFoto(t, pool, leserID, alt)
	schreibePasswort(t, pool, alt)

	gesamt, err := rotiere(context.Background(), pool, alt, neu, false)
	if err != nil {
		t.Fatalf("rotiere: %v", err)
	}
	if gesamt != 2 {
		t.Fatalf("gesamt = %d, erwartet 2 (ein Foto, ein Passwort)", gesamt)
	}

	for name, geschrieben := range map[string]struct {
		wert, klartext []byte
	}{
		"Foto":     {liesFoto(t, pool, leserID), fotoKlartext},
		"Passwort": {liesPasswort(t, pool), passwortKlartext},
	} {
		klar, err := crypto.DecryptMit(neu, geschrieben.wert)
		if err != nil {
			t.Errorf("%s: mit dem NEUEN Schlüssel nicht lesbar: %v", name, err)
		} else if !bytes.Equal(klar, geschrieben.klartext) {
			t.Errorf("%s: Klartext nach dem Wechsel verändert: %q", name, klar)
		}
		if _, err := crypto.DecryptMit(alt, geschrieben.wert); err == nil {
			t.Errorf("%s: mit dem ALTEN Schlüssel noch lesbar — der Wechsel hat nicht geschrieben", name)
		}
	}
}

// TestRotiere_ProbelaufSchreibtNichts: `-pruefen` liest und rechnet alles, lässt aber
// jedes Byte stehen. Die Zahl meldet, was ein echter Lauf umschlüsseln WÜRDE.
func TestRotiere_ProbelaufSchreibtNichts(t *testing.T) {
	pool, leserID := rotationsProbe(t)
	alt, neu := schluessel(t, altSchluesselHex), schluessel(t, neuSchluesselHex)
	schreibeFoto(t, pool, leserID, alt)
	schreibePasswort(t, pool, alt)
	fotoVorher, passwortVorher := liesFoto(t, pool, leserID), liesPasswort(t, pool)

	gesamt, err := rotiere(context.Background(), pool, alt, neu, true)
	if err != nil {
		t.Fatalf("Probelauf: %v", err)
	}
	if gesamt != 2 {
		t.Fatalf("Probelauf zählt %d, erwartet 2", gesamt)
	}
	if !bytes.Equal(liesFoto(t, pool, leserID), fotoVorher) {
		t.Errorf("Probelauf hat das Foto verändert")
	}
	if !bytes.Equal(liesPasswort(t, pool), passwortVorher) {
		t.Errorf("Probelauf hat das Passwort verändert")
	}
}

// TestRotiere_EinUnlesbarerDatensatzLaesstAllesStehen: Das Foto (erste Tabelle) ist mit
// dem alten Schlüssel lesbar und wird umgeschlüsselt; das Passwort (zweite Tabelle) trägt
// einen fremden Schlüssel. Der Lauf muss scheitern, den Datensatz nennen — und das schon
// umgeschlüsselte Foto wieder zurückrollen. Sonst bliebe ein Bestand mit zwei Schlüsseln.
func TestRotiere_EinUnlesbarerDatensatzLaesstAllesStehen(t *testing.T) {
	pool, leserID := rotationsProbe(t)
	alt, neu, fremd := schluessel(t, altSchluesselHex), schluessel(t, neuSchluesselHex), schluessel(t, fremdSchluesselHex)
	schreibeFoto(t, pool, leserID, alt)
	schreibePasswort(t, pool, fremd)
	fotoVorher, passwortVorher := liesFoto(t, pool, leserID), liesPasswort(t, pool)

	gesamt, err := rotiere(context.Background(), pool, alt, neu, false)
	if err == nil {
		t.Fatalf("rotiere lief durch (%d Datensätze), obwohl das Passwort unlesbar ist", gesamt)
	}
	for _, erwartet := range []string{"SMTP-Passwort", "datensatz 1", crypto.SchluesselVariable} {
		if !strings.Contains(err.Error(), erwartet) {
			t.Errorf("Fehler nennt %q nicht: %v", erwartet, err)
		}
	}
	if !bytes.Equal(liesFoto(t, pool, leserID), fotoVorher) {
		t.Errorf("Foto wurde umgeschlüsselt, obwohl der Lauf scheiterte — Bestand mit zwei Schlüsseln")
	}
	if !bytes.Equal(liesPasswort(t, pool), passwortVorher) {
		t.Errorf("Passwort wurde verändert, obwohl es unlesbar war")
	}
}

// TestRotiere_JedeVerschluesselteSpalteStehtInDerListe: Die Liste `tabellen` ist von Hand
// gepflegt. Kommt eine dritte BYTEA-Spalte dazu, die der Schlüsselwechsel nicht kennt,
// wäre sie nach dem Wechsel still unlesbar. Deshalb muss jede BYTEA-Spalte der Datenbank
// in der Liste stehen — und jeder Eintrag der Liste in der Datenbank existieren (eine
// umbenannte Spalte fiele sonst erst beim echten Lauf auf).
func TestRotiere_JedeVerschluesselteSpalteStehtInDerListe(t *testing.T) {
	pool := pgtest.Pool(t)
	rows, err := pool.Query(context.Background(), `
		SELECT table_name, column_name FROM information_schema.columns
		WHERE table_schema = 'public' AND data_type = 'bytea'
		ORDER BY table_name, column_name`)
	if err != nil {
		t.Fatalf("Spalten lesen: %v", err)
	}
	defer rows.Close()
	inDerDB := map[string]bool{}
	for rows.Next() {
		var tabelle, spalte string
		if err := rows.Scan(&tabelle, &spalte); err != nil {
			t.Fatalf("Spalte lesen: %v", err)
		}
		inDerDB[tabelle+"."+spalte] = true
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("Spalten lesen: %v", err)
	}
	// Nicht-leer-Garantie: Findet die Abfrage keine Spalte, misst der Test nichts.
	if len(inDerDB) < 2 {
		t.Fatalf("nur %d BYTEA-Spalten gefunden — die Abfrage über information_schema greift nicht", len(inDerDB))
	}

	inDerListe := map[string]bool{}
	for _, e := range tabellen {
		inDerListe[e.tabelle+"."+e.datenSpalte] = true
	}
	for spalte := range inDerDB {
		if !inDerListe[spalte] {
			t.Errorf("BYTEA-Spalte %s fehlt in `tabellen` — nach einem Schlüsselwechsel wäre sie unlesbar. "+
				"Eintragen, oder hier begründet ausnehmen, wenn sie nicht mit %s verschlüsselt ist.",
				spalte, crypto.SchluesselVariable)
		}
	}
	for spalte := range inDerListe {
		if !inDerDB[spalte] {
			t.Errorf("`tabellen` nennt %s, die Datenbank kennt die Spalte nicht (umbenannt?)", spalte)
		}
	}
}
