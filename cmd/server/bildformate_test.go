package main

import (
	"bytes"
	"encoding/base64"
	"image"
	"testing"
)

// Die Bibliotheken dekodieren mit image.Decode und finden nur die Formate, deren Decoder das
// Programm einbindet. Fehlt einer, scheitert erst das Hochladen oder Abrufen eines Covers mit
// „image: unknown format". Die Proben sind fertige Bytes, weil ein Encoder im Test seinen
// Decoder selbst anmeldete.
//
// Blindheit: GIF, JPEG und PNG melden auch die PDF-Bibliotheken an (gofpdf, pdfcpu, maroto),
// WebP auch internal/pkg/imageutil. Rot wird die Prüfung erst, wenn keiner das Format mehr einbindet.
func TestServer_LiestDieBildformate(t *testing.T) {
	proben := map[string]string{
		"gif":  "R0lGODlhAQABAIAAAAAAAP///ywAAAAAAQABAAACAkQBADs=",
		"jpeg": "/9j/2wCEAAgGBgcGBQgHBwcJCQgKDBQNDAsLDBkSEw8UHRofHh0aHBwgJC4nICIsIxwcKDcpLDAxNDQ0Hyc5PTgyPC4zNDIBCQkJDAsMGA0NGDIhHCEyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMv/AAAsIAAEAAQEBEQD/xADSAAABBQEBAQEBAQAAAAAAAAAAAQIDBAUGBwgJCgsQAAIBAwMCBAMFBQQEAAABfQECAwAEEQUSITFBBhNRYQcicRQygZGhCCNCscEVUtHwJDNicoIJChYXGBkaJSYnKCkqNDU2Nzg5OkNERUZHSElKU1RVVldYWVpjZGVmZ2hpanN0dXZ3eHl6g4SFhoeIiYqSk5SVlpeYmZqio6Slpqeoqaqys7S1tre4ubrCw8TFxsfIycrS09TV1tfY2drh4uPk5ebn6Onq8fLz9PX29/j5+v/aAAgBAQAAPwD5/r//2Q==",
		"png":  "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAIAAACQd1PeAAAAEUlEQVR4nAAEAPv/AshkMgMAA14BYWV63BUAAAAASUVORK5CYII=",
		"webp": "UklGRh4AAABXRUJQVlA4TBEAAAAvAAAAAAdQsiJXpv+BiOh/AAA=",
	}
	for format, probe := range proben {
		daten, err := base64.StdEncoding.DecodeString(probe)
		if err != nil {
			t.Fatalf("Probe %s entpacken: %v", format, err)
		}
		bild, gelesen, err := image.Decode(bytes.NewReader(daten))
		if err != nil {
			t.Errorf("image.Decode(%s-Probe) = %v, erwartet ein Bild: Kein Paket des Servers bindet den Decoder ein, er gehört in main.go", format, err)
			continue
		}
		if gelesen != format || bild.Bounds().Dx() != 1 {
			t.Errorf("image.Decode(%s-Probe) = Format %q mit %d Pixel Breite, erwartet %q mit 1", format, gelesen, bild.Bounds().Dx(), format)
		}
	}
}
