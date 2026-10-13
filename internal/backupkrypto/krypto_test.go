package backupkrypto

import (
	"bytes"
	"compress/gzip"
	"io"
	"testing"
)

// packe packt mit gzip wie der Backup-Lauf in jobs.
func packe(t *testing.T, daten []byte) []byte {
	t.Helper()
	var puffer bytes.Buffer
	gz := gzip.NewWriter(&puffer)
	if _, err := gz.Write(daten); err != nil {
		t.Fatalf("gzip schreiben: %v", err)
	}
	if err := gz.Close(); err != nil {
		t.Fatalf("gzip schließen: %v", err)
	}
	return puffer.Bytes()
}

func entpacke(t *testing.T, daten []byte) []byte {
	t.Helper()
	gz, err := gzip.NewReader(bytes.NewReader(daten))
	if err != nil {
		t.Fatalf("gzip öffnen: %v", err)
	}
	entpackt, err := io.ReadAll(gz)
	if err != nil {
		t.Fatalf("gzip lesen: %v", err)
	}
	return entpackt
}

// Verschlüsseln (der Backup-Lauf in jobs und cmd/encrypt-backup) und Entschlüsseln
// (cmd/restore-backup) passen zusammen. Schlägt der Test fehl, lassen sich die automatischen
// Backups (.sql.gz.enc) nicht wiederherstellen.
func TestVerschluesseleUndEntschluessele(t *testing.T) {
	passphrase := "produktions-passphrase-mind-32-zeichen-xx"
	sql := []byte("-- pg_dump\nCREATE TABLE schueler (id uuid PRIMARY KEY);\nINSERT INTO schueler VALUES ('00000000-0000-0000-0000-000000000001');\n")

	gepackt := packe(t, sql)
	verschluesselt, err := Verschluessele(passphrase, gepackt)
	if err != nil {
		t.Fatalf("Verschluessele: %v", err)
	}

	// Ohne die Kennung läse Entschluessele die Datei nicht.
	if !IstScryptFormat(verschluesselt) {
		t.Fatal("das Backup trägt die Kennung des Formats nicht")
	}

	entschluesselt, err := Entschluessele(passphrase, verschluesselt)
	if err != nil {
		t.Fatalf("Entschluessele: %v", err)
	}
	if !bytes.Equal(entpacke(t, entschluesselt), sql) {
		t.Errorf("wiederhergestelltes SQL weicht ab")
	}
}

// Eine Datei ohne die Kennung wird abgelehnt und nicht über die frühere Ableitung mit SHA-256
// gelesen; so begannen die früheren Backups, gleich mit Nonce und Ciphertext.
func TestEntschluessele_LehntDateiOhneKennungAb(t *testing.T) {
	// Beliebige Bytes ohne „BKDF" stehen für ein Backup im früheren Format.
	ohneKennung := bytes.Repeat([]byte{0x01}, 64)
	if IstScryptFormat(ohneKennung) {
		t.Fatal("Testdaten dürfen die Kennung nicht tragen")
	}
	if _, err := Entschluessele("egal-passphrase-mind-32-zeichen-xxxxxxxx", ohneKennung); err == nil {
		t.Fatal("ein Backup ohne die Kennung des Formats muss abgelehnt werden (keine Ableitung mit SHA-256)")
	}
}

func TestEntschluessele_FalschePassphraseScheitert(t *testing.T) {
	verschluesselt, err := Verschluessele("richtige-passphrase-mind-32-zeichen-xx", packe(t, []byte("geheime daten")))
	if err != nil {
		t.Fatalf("Verschluessele: %v", err)
	}
	// Eine falsche Passphrase scheitert an der Authentifizierung von GCM und liefert keinen
	// Datenmüll.
	if _, err := Entschluessele("FALSCHE-passphrase-mind-32-zeichen-xxxx", verschluesselt); err == nil {
		t.Error("Entschlüsselung mit falscher Passphrase soll fehlschlagen, war aber erfolgreich")
	}
}

func TestEntschluessele_ZuKurzeDateiScheitert(t *testing.T) {
	if _, err := Entschluessele("egal-passphrase-mind-32-zeichen-xxxxxxxx", []byte{0x01, 0x02}); err == nil {
		t.Error("eine Datei, die kürzer ist als die Nonce, soll einen Fehler liefern")
	}
}

func TestEntschluessele_VeraenderteDateiScheitert(t *testing.T) {
	verschluesselt, err := Verschluessele("passphrase-mind-32-zeichen-xxxxxxxxxxxx", packe(t, []byte("integritaet")))
	if err != nil {
		t.Fatalf("Verschluessele: %v", err)
	}
	// Das letzte Byte liegt im Tag von GCM; ein gekipptes Bit dort muss auffallen.
	verschluesselt[len(verschluesselt)-1] ^= 0xFF
	if _, err := Entschluessele("passphrase-mind-32-zeichen-xxxxxxxxxxxx", verschluesselt); err == nil {
		t.Error("eine veränderte Datei soll an der Prüfung der Integrität von GCM scheitern")
	}
}
