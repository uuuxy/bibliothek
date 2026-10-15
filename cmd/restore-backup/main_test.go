package main

import (
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"bibliothek/internal/backupkrypto"
)

// erzeugeSicherung erzeugt eine gültige Sicherung im Format mit scrypt über die echte
// Verschlüsselung; so prüft der Test genau das Format, das das Werkzeug im Betrieb vorfindet,
// statt es nachzubauen.
func erzeugeSicherung(t *testing.T, passphrase string, klartext []byte) []byte {
	t.Helper()
	verschluesselt, err := backupkrypto.Verschluessele(passphrase, klartext)
	if err != nil {
		t.Fatalf("backupkrypto.Verschluessele: %v", err)
	}
	return verschluesselt
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

func TestFuehreAus_FalscheArgumente(t *testing.T) {
	argumente := []string{} // kein Argument
	err := fuehreAus(argumente, io.Discard)
	if err == nil || !strings.Contains(err.Error(), "usage:") {
		t.Errorf("fuehreAus: Fehler %v, erwartet einen mit %q", err, "usage:")
	}

	argumente = []string{"in.enc", "out.sql", "extra"} // drei Argumente
	err = fuehreAus(argumente, io.Discard)
	if err == nil || !strings.Contains(err.Error(), "usage:") {
		t.Errorf("fuehreAus: Fehler %v, erwartet einen mit %q", err, "usage:")
	}
}

func TestFuehreAus_OhneSchluessel(t *testing.T) {
	argumente := []string{"in.enc"}
	t.Setenv("BACKUP_ENCRYPTION_KEY", "")
	err := fuehreAus(argumente, io.Discard)
	if err == nil || !strings.Contains(err.Error(), "BACKUP_ENCRYPTION_KEY nicht gesetzt") {
		t.Errorf("fuehreAus: Fehler %v, erwartet einen mit %q", err, "BACKUP_ENCRYPTION_KEY nicht gesetzt")
	}
}

func TestFuehreAus_DateiFehlt(t *testing.T) {
	argumente := []string{"does_not_exist.enc"}
	t.Setenv("BACKUP_ENCRYPTION_KEY", "secret")
	err := fuehreAus(argumente, io.Discard)
	if err == nil || !strings.Contains(err.Error(), "konnte nicht gelesen werden") {
		t.Errorf("fuehreAus: Fehler %v, erwartet einen mit %q", err, "konnte nicht gelesen werden")
	}
}

func TestFuehreAus_EntschluesselungScheitert(t *testing.T) {
	verzeichnis := t.TempDir()
	eingabe := filepath.Join(verzeichnis, "bad.enc")
	err := os.WriteFile(eingabe, []byte("invalid data"), 0644)
	if err != nil {
		t.Fatalf("os.WriteFile: %v", err)
	}

	argumente := []string{eingabe}
	t.Setenv("BACKUP_ENCRYPTION_KEY", "secret")
	err = fuehreAus(argumente, io.Discard)
	if err == nil || !strings.Contains(err.Error(), "entschlüsselung fehlgeschlagen") {
		t.Errorf("fuehreAus: Fehler %v, erwartet einen mit %q", err, "entschlüsselung fehlgeschlagen")
	}
}

func TestFuehreAus_GzipScheitert(t *testing.T) {
	verzeichnis := t.TempDir()
	eingabe := filepath.Join(verzeichnis, "bad_gzip.enc")

	// Gültig verschlüsselt, aber kein gzip.
	verschluesselt := erzeugeSicherung(t, "secret", []byte("not a gzip file"))
	err := os.WriteFile(eingabe, verschluesselt, 0644)
	if err != nil {
		t.Fatalf("os.WriteFile: %v", err)
	}

	argumente := []string{eingabe}
	t.Setenv("BACKUP_ENCRYPTION_KEY", "secret")
	err = fuehreAus(argumente, io.Discard)
	if err == nil || !strings.Contains(err.Error(), "gzip-header ungültig") {
		t.Errorf("fuehreAus: Fehler %v, erwartet einen mit %q", err, "gzip-header ungültig")
	}
}

func TestFuehreAus_ErfolgAufStdout(t *testing.T) {
	verzeichnis := t.TempDir()
	eingabe := filepath.Join(verzeichnis, "backup.enc")

	sqlDaten := []byte("SELECT * FROM books;")
	gepackt := gzipDaten(t, sqlDaten)
	verschluesselt := erzeugeSicherung(t, "secret", gepackt)

	err := os.WriteFile(eingabe, verschluesselt, 0644)
	if err != nil {
		t.Fatalf("os.WriteFile: %v", err)
	}

	argumente := []string{eingabe}
	t.Setenv("BACKUP_ENCRYPTION_KEY", "secret")

	var stdout bytes.Buffer
	err = fuehreAus(argumente, &stdout)
	if err != nil {
		t.Fatalf("fuehreAus: %v", err)
	}

	ausgabeDaten := stdout.Bytes()
	if !bytes.Equal(ausgabeDaten, sqlDaten) {
		t.Errorf("ausgabeDaten = %q, erwartet %q", ausgabeDaten, sqlDaten)
	}
}

func TestFuehreAus_ErfolgInDatei(t *testing.T) {
	verzeichnis := t.TempDir()
	eingabe := filepath.Join(verzeichnis, "backup.enc")
	ausgabeDatei := filepath.Join(verzeichnis, "out.sql")

	sqlDaten := []byte("SELECT * FROM books;")
	gepackt := gzipDaten(t, sqlDaten)
	verschluesselt := erzeugeSicherung(t, "secret", gepackt)

	err := os.WriteFile(eingabe, verschluesselt, 0644)
	if err != nil {
		t.Fatalf("os.WriteFile: %v", err)
	}

	argumente := []string{eingabe, ausgabeDatei}
	t.Setenv("BACKUP_ENCRYPTION_KEY", "secret")

	err = fuehreAus(argumente, io.Discard)
	if err != nil {
		t.Fatalf("fuehreAus: %v", err)
	}

	ausgabeDaten, err := os.ReadFile(ausgabeDatei)
	if err != nil {
		t.Fatalf("os.ReadFile: %v", err)
	}
	if !bytes.Equal(ausgabeDaten, sqlDaten) {
		t.Errorf("ausgabeDaten = %q, erwartet %q", ausgabeDaten, sqlDaten)
	}
}

// Nach einem Wechsel des BACKUP_ENCRYPTION_KEY öffnet jede Sicherung nur der Schlüssel, der bei
// ihrem Entstehen galt (docs/SECURITY.md, „BACKUP_ENCRYPTION_KEY wechseln"). Mit dem anderen
// nennt die Meldung den falschen Schlüssel, und es entsteht keine Ausgabedatei.
func TestFuehreAus_NachSchluesselwechsel(t *testing.T) {
	const (
		alt = "alter-schluessel-mit-mehr-als-32-zeichen-0001"
		neu = "neuer-schluessel-mit-mehr-als-32-zeichen-0002"
	)
	verzeichnis := t.TempDir()
	standVorher, standNachher := []byte("-- Stand vor dem Wechsel\n"), []byte("-- Stand nach dem Wechsel\n")
	vorher := filepath.Join(verzeichnis, "vorher.sql.gz.enc")
	nachher := filepath.Join(verzeichnis, "nachher.sql.gz.enc")
	if err := os.WriteFile(vorher, erzeugeSicherung(t, alt, gzipDaten(t, standVorher)), 0o600); err != nil {
		t.Fatalf("os.WriteFile: %v", err)
	}
	if err := os.WriteFile(nachher, erzeugeSicherung(t, neu, gzipDaten(t, standNachher)), 0o600); err != nil {
		t.Fatalf("os.WriteFile: %v", err)
	}

	faelle := []struct {
		name       string
		schluessel string
		datei      string
		inhalt     []byte // nil: Das Werkzeug lehnt ab.
	}{
		{"laufender Schlüssel, Sicherung von vor dem Wechsel", neu, vorher, nil},
		{"alter Schlüssel, Sicherung von vor dem Wechsel", alt, vorher, standVorher},
		{"laufender Schlüssel, Sicherung nach dem Wechsel", neu, nachher, standNachher},
		{"alter Schlüssel, Sicherung nach dem Wechsel", alt, nachher, nil},
	}
	for _, fall := range faelle {
		t.Run(fall.name, func(t *testing.T) {
			ausgabe := filepath.Join(t.TempDir(), "wiederherstellung.sql")
			argumente := []string{fall.datei, ausgabe}
			t.Setenv("BACKUP_ENCRYPTION_KEY", fall.schluessel)

			err := fuehreAus(argumente, io.Discard)
			if fall.inhalt == nil {
				if err == nil || !strings.Contains(err.Error(), "falscher Schlüssel") {
					t.Errorf("Fehler %v, erwartet einen mit %q", err, "falscher Schlüssel")
				}
				if _, err := os.Stat(ausgabe); err == nil {
					t.Errorf("Datei %s besteht noch", ausgabe)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			inhalt, err := os.ReadFile(ausgabe)
			if err != nil {
				t.Fatalf("os.ReadFile: %v", err)
			}
			if !bytes.Equal(inhalt, fall.inhalt) {
				t.Errorf("inhalt = %q, erwartet %q", inhalt, fall.inhalt)
			}
		})
	}
}

func TestFuehreAus_AusgabedateiScheitert(t *testing.T) {
	verzeichnis := t.TempDir()
	eingabe := filepath.Join(verzeichnis, "backup.enc")

	// Ein Verzeichnis als Ausgabedatei, damit das Öffnen scheitert.
	ausgabeDatei := filepath.Join(verzeichnis, "out_dir")
	err := os.Mkdir(ausgabeDatei, 0755)
	if err != nil {
		t.Fatalf("os.Mkdir: %v", err)
	}

	sqlDaten := []byte("SELECT * FROM books;")
	gepackt := gzipDaten(t, sqlDaten)
	verschluesselt := erzeugeSicherung(t, "secret", gepackt)

	err = os.WriteFile(eingabe, verschluesselt, 0644)
	if err != nil {
		t.Fatalf("os.WriteFile: %v", err)
	}

	argumente := []string{eingabe, ausgabeDatei}
	t.Setenv("BACKUP_ENCRYPTION_KEY", "secret")

	err = fuehreAus(argumente, io.Discard)
	if err == nil || !strings.Contains(err.Error(), "ausgabedatei") {
		t.Errorf("fuehreAus: Fehler %v, erwartet einen mit %q", err, "ausgabedatei")
	}
}

func TestMain_FehlerEndetMitExitEins(t *testing.T) {
	// Im zweiten Durchlauf, erkannt an der Umgebungsvariablen, läuft main selbst.
	if os.Getenv("BE_CRASHER") == "1" {
		main()
		return
	}

	programm := os.Args[0]
	// Das Testprogramm noch einmal starten, nur mit diesem Test und gesetzter Variable.
	befehl := exec.Command(programm, "-test.run=TestMain_FehlerEndetMitExitEins")
	// Ohne Schlüssel scheitert main an jedem Argument, auch an den Schaltern des Testlaufs.
	befehl.Env = append(os.Environ(), "BE_CRASHER=1", "BACKUP_ENCRYPTION_KEY=")
	err := befehl.Run()
	// main endet mit 1; dann liefert Run einen Fehler.
	if exitFehler, ok := err.(*exec.ExitError); ok && !exitFehler.Success() {
		return // erwartet
	}
	t.Fatalf("Prozess endete mit %v, erwartet Exit-Status 1", err)
}
