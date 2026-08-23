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

const testKey = "test-passphrase-mind-32-zeichen-xxxxxxxx"

func umgebung(werte map[string]string) func(string) string {
	return func(name string) string { return werte[name] }
}

func gzipData(t *testing.T, data []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	_, err := w.Write(data)
	require.NoError(t, err)
	require.NoError(t, w.Close())
	return buf.Bytes()
}

// TestRun_RundwegUeberDenEchtenRestore ist das eigentliche Gate: Was dieses Werkzeug
// ausgibt, muss der dokumentierte Wiederherstellungsweg (cmd/restore-backup:
// entschlüsseln, dann gunzip) wieder in genau den Dump verwandeln, der hineinging.
// Schlägt er fehl, sind die Backups aus update.sh und scripts/backup.sh nicht
// wiederherstellbar — und der Klartext, den sie ersetzen, ist dann bereits gelöscht.
func TestRun_RundwegUeberDenEchtenRestore(t *testing.T) {
	sql := []byte("-- pg_dump\nCREATE TABLE schueler (id uuid PRIMARY KEY);\n")

	var aus, fehler bytes.Buffer
	err := run(nil, bytes.NewReader(gzipData(t, sql)), &aus, &fehler,
		umgebung(map[string]string{"BACKUP_ENCRYPTION_KEY": testKey}))
	require.NoError(t, err)

	komprimiert, err := backupkrypto.EntschluesseleBackup(testKey, aus.Bytes())
	require.NoError(t, err)
	leser, err := gzip.NewReader(bytes.NewReader(komprimiert))
	require.NoError(t, err)
	wieder, err := io.ReadAll(leser)
	require.NoError(t, err)
	assert.Equal(t, sql, wieder)
}

// TestRun_AusgabeTraegtKeinenKlartext hält den Zweck des ganzen Befundes fest (A5): In
// der erzeugten Datei darf kein Schülername mehr lesbar stehen.
func TestRun_AusgabeTraegtKeinenKlartext(t *testing.T) {
	sql := []byte("INSERT INTO schueler VALUES ('Erika Mustermann', 'Musterweg 3');")

	var aus, fehler bytes.Buffer
	require.NoError(t, run(nil, bytes.NewReader(gzipData(t, sql)), &aus, &fehler,
		umgebung(map[string]string{"BACKUP_ENCRYPTION_KEY": testKey})))

	assert.True(t, backupkrypto.IstScryptFormat(aus.Bytes()), "Ausgabe trägt nicht die BKDF-Kennung")
	assert.NotContains(t, aus.String(), "Mustermann")
	assert.NotContains(t, aus.String(), "Musterweg")
}

func TestRun_OhneSchluessel(t *testing.T) {
	var aus, fehler bytes.Buffer
	err := run(nil, bytes.NewReader(gzipData(t, []byte("x"))), &aus, &fehler, umgebung(nil))
	assert.ErrorContains(t, err, "BACKUP_ENCRYPTION_KEY nicht gesetzt")
	assert.Empty(t, aus.Bytes(), "ohne Schlüssel darf nichts ausgegeben werden")
}

// TestRun_KurzerSchluesselWarntUndArbeitet: Der Aufrufer ist ein Deploy-Skript. Ein
// Abbruch nähme ihm die Sicherung ganz — deshalb warnen, nicht verweigern (wie
// jobs/backup.go).
func TestRun_KurzerSchluesselWarntUndArbeitet(t *testing.T) {
	var aus, fehler bytes.Buffer
	err := run(nil, bytes.NewReader(gzipData(t, []byte("x"))), &aus, &fehler,
		umgebung(map[string]string{"BACKUP_ENCRYPTION_KEY": "kurz"}))
	require.NoError(t, err)
	assert.NotEmpty(t, aus.Bytes())
	assert.Contains(t, fehler.String(), "WARNUNG")
}

func TestRun_LeereEingabe(t *testing.T) {
	var aus, fehler bytes.Buffer
	err := run(nil, strings.NewReader(""), &aus, &fehler,
		umgebung(map[string]string{"BACKUP_ENCRYPTION_KEY": testKey}))
	assert.ErrorContains(t, err, "leere eingabe")
	assert.Empty(t, aus.Bytes())
}

// TestRun_KeinGzip und TestRun_AbgeschnittenesGzip decken den stillen Ausfall ab, den
// `pipefail` in den Skripten schon einmal abgestellt hat: ein abgebrochener pg_dump
// darf nicht als Backup durchgehen.
func TestRun_KeinGzip(t *testing.T) {
	var aus, fehler bytes.Buffer
	err := run(nil, strings.NewReader("pg_dump: error: connection failed"), &aus, &fehler,
		umgebung(map[string]string{"BACKUP_ENCRYPTION_KEY": testKey}))
	assert.ErrorContains(t, err, "kein gültiger gzip-Strom")
	assert.Empty(t, aus.Bytes())
}

func TestRun_AbgeschnittenesGzip(t *testing.T) {
	voll := gzipData(t, bytes.Repeat([]byte("CREATE TABLE t;\n"), 100))

	var aus, fehler bytes.Buffer
	err := run(nil, bytes.NewReader(voll[:len(voll)-8]), &aus, &fehler,
		umgebung(map[string]string{"BACKUP_ENCRYPTION_KEY": testKey}))
	assert.ErrorContains(t, err, "beschädigter gzip-Strom")
	assert.Empty(t, aus.Bytes())
}

func TestRun_LeererDumpInGueltigemGzip(t *testing.T) {
	var aus, fehler bytes.Buffer
	err := run(nil, bytes.NewReader(gzipData(t, nil)), &aus, &fehler,
		umgebung(map[string]string{"BACKUP_ENCRYPTION_KEY": testKey}))
	assert.ErrorContains(t, err, "leeren Dump")
	assert.Empty(t, aus.Bytes())
}

func TestRun_ArgumenteWerdenAbgelehnt(t *testing.T) {
	var aus, fehler bytes.Buffer
	err := run([]string{"datei.sql.gz"}, bytes.NewReader(gzipData(t, []byte("x"))), &aus, &fehler,
		umgebung(map[string]string{"BACKUP_ENCRYPTION_KEY": testKey}))
	assert.ErrorContains(t, err, "keine Argumente")
}
