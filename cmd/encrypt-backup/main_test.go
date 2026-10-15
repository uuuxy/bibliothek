package main

import (
	"bytes"
	"compress/gzip"
	"io"
	"strings"
	"testing"

	"bibliothek/internal/backupkrypto"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	require.NoError(t, err)
	require.NoError(t, w.Close())
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
	require.NoError(t, err)

	komprimiert, err := backupkrypto.Entschluessele(testSchluessel, aus.Bytes())
	require.NoError(t, err)
	leser, err := gzip.NewReader(bytes.NewReader(komprimiert))
	require.NoError(t, err)
	wieder, err := io.ReadAll(leser)
	require.NoError(t, err)
	assert.Equal(t, sql, wieder)
}

// Der Zweck des Werkzeugs: In der erzeugten Datei darf kein Name eines Schülers mehr lesbar
// stehen.
func TestFuehreAus_AusgabeTraegtKeinenKlartext(t *testing.T) {
	sql := []byte("INSERT INTO schueler VALUES ('Erika Mustermann', 'Musterweg 3');")

	var aus, fehler bytes.Buffer
	require.NoError(t, fuehreAus(nil, bytes.NewReader(gzipDaten(t, sql)), &aus, &fehler,
		umgebung(map[string]string{"BACKUP_ENCRYPTION_KEY": testSchluessel})))

	assert.True(t, backupkrypto.IstScryptFormat(aus.Bytes()), "Ausgabe trägt nicht die BKDF-Kennung")
	assert.NotContains(t, aus.String(), "Mustermann")
	assert.NotContains(t, aus.String(), "Musterweg")
}

func TestFuehreAus_OhneSchluessel(t *testing.T) {
	var aus, fehler bytes.Buffer
	err := fuehreAus(nil, bytes.NewReader(gzipDaten(t, []byte("x"))), &aus, &fehler, umgebung(nil))
	assert.ErrorContains(t, err, "BACKUP_ENCRYPTION_KEY nicht gesetzt")
	assert.Empty(t, aus.Bytes(), "ohne Schlüssel darf nichts ausgegeben werden")
}

// Der Aufrufer ist ein Skript des Deploys. Ein Abbruch nähme ihm die Sicherung ganz; deshalb
// warnen, nicht verweigern (wie jobs/backup.go).
func TestFuehreAus_KurzerSchluesselWarntUndArbeitet(t *testing.T) {
	var aus, fehler bytes.Buffer
	err := fuehreAus(nil, bytes.NewReader(gzipDaten(t, []byte("x"))), &aus, &fehler,
		umgebung(map[string]string{"BACKUP_ENCRYPTION_KEY": "kurz"}))
	require.NoError(t, err)
	assert.NotEmpty(t, aus.Bytes())
	assert.Contains(t, fehler.String(), "WARNUNG")
}

func TestFuehreAus_LeereEingabe(t *testing.T) {
	var aus, fehler bytes.Buffer
	err := fuehreAus(nil, strings.NewReader(""), &aus, &fehler,
		umgebung(map[string]string{"BACKUP_ENCRYPTION_KEY": testSchluessel}))
	assert.ErrorContains(t, err, "leere eingabe")
	assert.Empty(t, aus.Bytes())
}

// TestFuehreAus_KeinGzip und TestFuehreAus_AbgeschnittenesGzip decken den stummen Ausfall ab,
// gegen den die Skripte `pipefail` setzen: Ein abgebrochener pg_dump darf nicht als Sicherung
// durchgehen.
func TestFuehreAus_KeinGzip(t *testing.T) {
	var aus, fehler bytes.Buffer
	err := fuehreAus(nil, strings.NewReader("pg_dump: error: connection failed"), &aus, &fehler,
		umgebung(map[string]string{"BACKUP_ENCRYPTION_KEY": testSchluessel}))
	assert.ErrorContains(t, err, "kein gültiger gzip-Strom")
	assert.Empty(t, aus.Bytes())
}

func TestFuehreAus_AbgeschnittenesGzip(t *testing.T) {
	voll := gzipDaten(t, bytes.Repeat([]byte("CREATE TABLE t;\n"), 100))

	var aus, fehler bytes.Buffer
	err := fuehreAus(nil, bytes.NewReader(voll[:len(voll)-8]), &aus, &fehler,
		umgebung(map[string]string{"BACKUP_ENCRYPTION_KEY": testSchluessel}))
	assert.ErrorContains(t, err, "beschädigter gzip-Strom")
	assert.Empty(t, aus.Bytes())
}

func TestFuehreAus_LeererDumpInGueltigemGzip(t *testing.T) {
	var aus, fehler bytes.Buffer
	err := fuehreAus(nil, bytes.NewReader(gzipDaten(t, nil)), &aus, &fehler,
		umgebung(map[string]string{"BACKUP_ENCRYPTION_KEY": testSchluessel}))
	assert.ErrorContains(t, err, "leeren Dump")
	assert.Empty(t, aus.Bytes())
}

func TestFuehreAus_ArgumenteWerdenAbgelehnt(t *testing.T) {
	var aus, fehler bytes.Buffer
	err := fuehreAus([]string{"datei.sql.gz"}, bytes.NewReader(gzipDaten(t, []byte("x"))), &aus, &fehler,
		umgebung(map[string]string{"BACKUP_ENCRYPTION_KEY": testSchluessel}))
	assert.ErrorContains(t, err, "keine Argumente")
}
