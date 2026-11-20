// Package coverablage kennt den Ort der lokal gespeicherten Cover: Es prüft den Pfad einer
// Cover-URL, öffnet das Verzeichnis, legt eine Cover-Datei ab und entfernt sie.
//
// Ohne Bildbibliothek. internal/pkg/coverdatei wandelt Cover für die PDF-Erzeuger und zieht dafür die
// WebP-Bibliothek (cgo) mit; das Löschen eines Titels braucht nur den Pfad, und über
// repository binden es auch die Werkzeuge des Images ein, die ohne cgo gebaut werden.
package coverablage

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"bibliothek/internal/pkg/closeutil"
)

// Wurzel ist das Verzeichnis aller lokal gespeicherten Bilder, relativ zum
// Arbeitsverzeichnis des Servers.
const Wurzel = "uploads"

// MaxBreite und MaxHoehe begrenzen ein abgelegtes Cover, ob hochgeladen oder aus dem Netz
// geholt; was größer ankommt, wird vor dem Ablegen verkleinert.
const (
	MaxBreite = 600
	MaxHoehe  = 900
)

// Zerlege prüft die Cover-URL und liefert den Dateipfad zweimal: einmal relativ zur Wurzel
// (so verlangt es os.Root) und einmal vollständig, weil die PDF-Erzeuger ihn als Namen ihres
// Bildspeichers führen. ok=false heißt: nicht verwendbar.
//
// Die Herkunftsprüfung ist kein Schutz — das ist os.Root beim Zugriff (OeffneWurzel). Sie
// steht hier als Vorfilter: Die meisten Titel haben kein Cover, und für die soll kein
// Verzeichnis geöffnet werden.
func Zerlege(coverURL string) (rel, pfad string, ok bool) {
	if !strings.HasPrefix(coverURL, "/"+Wurzel+"/") {
		return "", "", false
	}
	pfad = filepath.Clean(strings.TrimPrefix(coverURL, "/"))
	rel, err := filepath.Rel(Wurzel, pfad)
	if err != nil {
		return "", "", false
	}
	return rel, pfad, true
}

// OeffneWurzel öffnet das Upload-Verzeichnis als os.Root. Erst diese Klammer hält auch den
// Fall auf, den die Textprüfung nicht sehen kann: eine Verknüpfung innerhalb von uploads,
// die nach draußen zeigt. os.Stat würde ihr folgen, root.Stat verweigert sie.
func OeffneWurzel() (*os.Root, bool) {
	wurzel, err := os.OpenRoot(Wurzel)
	if err != nil {
		return nil, false
	}
	return wurzel, true
}

// Loesche entfernt die Datei eines lokal gespeicherten Covers, etwa nachdem sein Titel
// gelöscht ist. Nur Dateien direkt in der Wurzel: In ihren Unterordnern liegen andere Bilder
// (Ausweisfotos, der Zwischenspeicher des Cover-Abrufs), und die Cover-URL eines Titels kommt
// aus einer Maske. Eine fehlende Datei und eine Adresse außerhalb sind kein Fehler; der
// Aufrufer räumt nur auf.
func Loesche(coverURL string) error {
	rel, _, ok := Zerlege(coverURL)
	if !ok || rel == "." || rel != filepath.Base(rel) {
		return nil
	}
	wurzel, ok := OeffneWurzel()
	if !ok {
		return nil
	}
	defer closeutil.Schliesse(wurzel, "coverablage wurzel")

	// Lstat: Eine Verknüpfung oder ein Ordner unter diesem Namen ist kein Cover.
	if info, err := wurzel.Lstat(rel); err != nil || !info.Mode().IsRegular() {
		return nil
	}
	if err := wurzel.Remove(rel); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// Schreibe legt inhalt als Datei name direkt in der Wurzel ab und legt die Wurzel an, wenn es sie
// noch nicht gibt. Ein Name, der aus der Wurzel herausführt, scheitert am Betriebssystem
// (os.Root), nicht an einer Prüfung der Zeichenkette: Zwischen Prüfung und Zugriff könnte sich
// der Pfad ändern, eine Verknüpfung in uploads genügt.
func Schreibe(name string, inhalt []byte) error {
	return mitAngelegterWurzel(func(wurzel *os.Root) error {
		if err := SchreibeUeberZwischendatei(wurzel, name, inhalt); err != nil {
			return fmt.Errorf("datei %q konnte nicht geschrieben werden: %w", name, err)
		}
		return nil
	})
}

// SchreibeUeberZwischendatei legt inhalt unter name in wurzel ab: erst unter einem eigenen
// vorläufigen Namen, geschlossen, dann umbenannt. Wer die Datei zugleich ausliefert, sieht so nie
// eine halbe, und ein Fehler beim Schließen hinterlässt keine. Zwei gleichzeitige Schreiber kommen
// einander nicht in die Quere: Jeder hat seine Zwischendatei, der spätere ersetzt den früheren ganz.
//
// Liegt unter name eine Verknüpfung oder ein Ordner, lehnt es ab wie Loesche: Das Umbenennen
// folgte ihr zwar nicht, ersetzte sie aber still, und eine Verknüpfung in der Ablage ist ein Befund.
func SchreibeUeberZwischendatei(wurzel *os.Root, name string, inhalt []byte) error {
	if info, err := wurzel.Lstat(name); err == nil && !info.Mode().IsRegular() {
		return fmt.Errorf("unter %q liegt keine Datei", name)
	}
	zufall := make([]byte, 8)
	if _, err := rand.Read(zufall); err != nil {
		return err
	}
	vorlaeufig := name + ".teil-" + hex.EncodeToString(zufall)
	if err := schreibeGanz(wurzel, vorlaeufig, inhalt); err != nil {
		return err
	}
	if err := wurzel.Rename(vorlaeufig, name); err != nil {
		entferneZwischendatei(wurzel, vorlaeufig)
		return err
	}
	return nil
}

// schreibeGanz legt eine neue Datei an, schreibt inhalt und schließt sie; ein Fehler beim Schließen
// zählt wie einer beim Schreiben. Scheitert es nach dem Anlegen, entfernt es die Datei wieder.
func schreibeGanz(wurzel *os.Root, name string, inhalt []byte) error {
	datei, err := wurzel.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	_, err = datei.Write(inhalt)
	if schliessFehler := datei.Close(); schliessFehler != nil && err == nil {
		err = schliessFehler
	}
	if err != nil {
		entferneZwischendatei(wurzel, name)
	}
	return err
}

// entferneZwischendatei räumt eine angefangene Zwischendatei weg; scheitert das, steht es im Log.
func entferneZwischendatei(wurzel *os.Root, name string) {
	if err := wurzel.Remove(name); err != nil && !errors.Is(err, fs.ErrNotExist) {
		slog.Warn("coverablage: Zwischendatei nicht entfernt", "datei", name, "fehler", err)
	}
}

// IstVorhanden sagt, ob name schon als Datei direkt in der Wurzel liegt.
func IstVorhanden(name string) bool {
	vorhanden := false
	_ = mitAngelegterWurzel(func(wurzel *os.Root) error { //nolint:errcheck // ohne Verzeichnis gibt es die Datei nicht
		info, err := wurzel.Stat(name)
		vorhanden = err == nil && info.Mode().IsRegular()
		return nil
	})
	return vorhanden
}

// mitAngelegterWurzel legt die Wurzel an, öffnet sie und übergibt sie an fn. Die Wurzel wird in
// jedem Fall geschlossen.
func mitAngelegterWurzel(fn func(*os.Root) error) error {
	if err := os.MkdirAll(Wurzel, 0750); err != nil {
		return fmt.Errorf("uploads-Verzeichnis konnte nicht angelegt werden: %w", err)
	}
	wurzel, err := os.OpenRoot(Wurzel)
	if err != nil {
		return fmt.Errorf("uploads-Verzeichnis konnte nicht geöffnet werden: %w", err)
	}
	defer closeutil.Schliesse(wurzel, "uploads dir")
	return fn(wurzel)
}
