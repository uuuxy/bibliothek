package main

import (
	"bytes"
	"compress/gzip"
	"io"
	"strings"
	"testing"

	"bibliothek/internal/backupkrypto"
)

const testSchluessel = "test-passphrase-mind-32-zeichen-xxxxxxxx"

func umgebung(werte map[string]string) func(string) string {
	return func(name string) string { return werte[name] }
}

func gzipDaten(t *testing.T, daten []byte) []byte {
	t.Helper()
	var puffer bytes.Buffer
	w := gzip.NewWriter(&puffer)
	_, err := w.Write(daten)
	if err != nil {
		t.Fatalf("w.Write: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("w.Close: %v", err)
	}
	return puffer.Bytes()
}

// Der eigentliche Test: Was das Werkzeug ausgibt, muss der beschriebene Weg der
// Wiederherstellung (cmd/restore-backup: entschlüsseln, dann gunzip) wieder in genau den Dump
// verwandeln, der hineinging. Gelingt das nicht, lassen sich die Sicherungen aus update.sh und
// scripts/backup.sh nicht wiederherstellen, und der Klartext, den sie ersetzen, ist dann schon
// gelöscht.
func TestFuehreAus_RundwegUeberDenEchtenRestore(t *testing.T) {
	sql := []byte("-- pg_dump\nCREATE TABLE schueler (id uuid PRIMARY KEY);\n")

	var aus, fehler bytes.Buffer
	err := fuehreAus(nil, bytes.NewReader(gzipDaten(t, sql)), &aus, &fehler,
		umgebung(map[string]string{"BACKUP_ENCRYPTION_KEY": testSchluessel}))
	if err != nil {
		t.Fatalf("fuehreAus: %v", err)
	}

	komprimiert, err := backupkrypto.Entschluessele(testSchluessel, aus.Bytes())
	if err != nil {
		t.Fatalf("backupkrypto.Entschluessele: %v", err)
	}
	leser, err := gzip.NewReader(bytes.NewReader(komprimiert))
	if err != nil {
		t.Fatalf("gzip.NewReader: %v", err)
	}
	wieder, err := io.ReadAll(leser)
	if err != nil {
		t.Fatalf("io.ReadAll: %v", err)
	}
	if !bytes.Equal(wieder, sql) {
		t.Errorf("wieder = %q, erwartet %q", wieder, sql)
	}
}

// Der Zweck des Werkzeugs: In der erzeugten Datei darf kein Name eines Schülers mehr lesbar
// stehen.
func TestFuehreAus_AusgabeTraegtKeinenKlartext(t *testing.T) {
	sql := []byte("INSERT INTO schueler VALUES ('Erika Mustermann', 'Musterweg 3');")

	var aus, fehler bytes.Buffer
	if err := fuehreAus(nil, bytes.NewReader(gzipDaten(t, sql)), &aus, &fehler, umgebung(map[string]string{"BACKUP_ENCRYPTION_KEY": testSchluessel})); err != nil {
		t.Fatalf("fuehreAus: %v", err)
	}

	if !backupkrypto.IstScryptFormat(aus.Bytes()) {
		t.Error("Ausgabe trägt nicht die BKDF-Kennung")
	}
	if got := aus.String(); strings.Contains(got, "Mustermann") {
		t.Errorf("aus.String() = %q, erwartet ohne %q", got, "Mustermann")
	}
	if got := aus.String(); strings.Contains(got, "Musterweg") {
		t.Errorf("aus.String() = %q, erwartet ohne %q", got, "Musterweg")
	}
}

func TestFuehreAus_OhneSchluessel(t *testing.T) {
	var aus, fehler bytes.Buffer
	err := fuehreAus(nil, bytes.NewReader(gzipDaten(t, []byte("x"))), &aus, &fehler, umgebung(nil))
	if err == nil || !strings.Contains(err.Error(), "BACKUP_ENCRYPTION_KEY nicht gesetzt") {
		t.Errorf("fuehreAus: Fehler %v, erwartet einen mit %q", err, "BACKUP_ENCRYPTION_KEY nicht gesetzt")
	}
	if got := aus.Bytes(); len(got) != 0 {
		t.Errorf("aus.Bytes() = %q, erwartet leer: ohne Schlüssel darf nichts ausgegeben werden", got)
	}
}

// Der Aufrufer ist ein Skript des Deploys. Ein Abbruch nähme ihm die Sicherung ganz; deshalb
// warnen, nicht verweigern (wie jobs/backup.go).
func TestFuehreAus_KurzerSchluesselWarntUndArbeitet(t *testing.T) {
	var aus, fehler bytes.Buffer
	err := fuehreAus(nil, bytes.NewReader(gzipDaten(t, []byte("x"))), &aus, &fehler,
		umgebung(map[string]string{"BACKUP_ENCRYPTION_KEY": "kurz"}))
	if err != nil {
		t.Fatalf("fuehreAus: %v", err)
	}
	if got := aus.Bytes(); len(got) == 0 {
		t.Error("aus.Bytes() ist leer")
	}
	if got := fehler.String(); !strings.Contains(got, "WARNUNG") {
		t.Errorf("fehler.String() = %q, erwartet darin %q", got, "WARNUNG")
	}
}

func TestFuehreAus_LeereEingabe(t *testing.T) {
	var aus, fehler bytes.Buffer
	err := fuehreAus(nil, strings.NewReader(""), &aus, &fehler,
		umgebung(map[string]string{"BACKUP_ENCRYPTION_KEY": testSchluessel}))
	if err == nil || !strings.Contains(err.Error(), "leere eingabe") {
		t.Errorf("fuehreAus: Fehler %v, erwartet einen mit %q", err, "leere eingabe")
	}
	if got := aus.Bytes(); len(got) != 0 {
		t.Errorf("aus.Bytes() = %q, erwartet leer", got)
	}
}

// TestFuehreAus_KeinGzip und TestFuehreAus_AbgeschnittenesGzip decken den stummen Ausfall ab,
// gegen den die Skripte `pipefail` setzen: Ein abgebrochener pg_dump darf nicht als Sicherung
// durchgehen.
func TestFuehreAus_KeinGzip(t *testing.T) {
	var aus, fehler bytes.Buffer
	err := fuehreAus(nil, strings.NewReader("pg_dump: error: connection failed"), &aus, &fehler,
		umgebung(map[string]string{"BACKUP_ENCRYPTION_KEY": testSchluessel}))
	if err == nil || !strings.Contains(err.Error(), "kein gültiger gzip-Strom") {
		t.Errorf("fuehreAus: Fehler %v, erwartet einen mit %q", err, "kein gültiger gzip-Strom")
	}
	if got := aus.Bytes(); len(got) != 0 {
		t.Errorf("aus.Bytes() = %q, erwartet leer", got)
	}
}

func TestFuehreAus_AbgeschnittenesGzip(t *testing.T) {
	voll := gzipDaten(t, bytes.Repeat([]byte("CREATE TABLE t;\n"), 100))

	var aus, fehler bytes.Buffer
	err := fuehreAus(nil, bytes.NewReader(voll[:len(voll)-8]), &aus, &fehler,
		umgebung(map[string]string{"BACKUP_ENCRYPTION_KEY": testSchluessel}))
	if err == nil || !strings.Contains(err.Error(), "beschädigter gzip-Strom") {
		t.Errorf("fuehreAus: Fehler %v, erwartet einen mit %q", err, "beschädigter gzip-Strom")
	}
	if got := aus.Bytes(); len(got) != 0 {
		t.Errorf("aus.Bytes() = %q, erwartet leer", got)
	}
}

func TestFuehreAus_LeererDumpInGueltigemGzip(t *testing.T) {
	var aus, fehler bytes.Buffer
	err := fuehreAus(nil, bytes.NewReader(gzipDaten(t, nil)), &aus, &fehler,
		umgebung(map[string]string{"BACKUP_ENCRYPTION_KEY": testSchluessel}))
	if err == nil || !strings.Contains(err.Error(), "leeren Dump") {
		t.Errorf("fuehreAus: Fehler %v, erwartet einen mit %q", err, "leeren Dump")
	}
	if got := aus.Bytes(); len(got) != 0 {
		t.Errorf("aus.Bytes() = %q, erwartet leer", got)
	}
}

func TestFuehreAus_ArgumenteWerdenAbgelehnt(t *testing.T) {
	var aus, fehler bytes.Buffer
	err := fuehreAus([]string{"datei.sql.gz"}, bytes.NewReader(gzipDaten(t, []byte("x"))), &aus, &fehler,
		umgebung(map[string]string{"BACKUP_ENCRYPTION_KEY": testSchluessel}))
	if err == nil || !strings.Contains(err.Error(), "keine Argumente") {
		t.Errorf("fuehreAus: Fehler %v, erwartet einen mit %q", err, "keine Argumente")
	}
}
