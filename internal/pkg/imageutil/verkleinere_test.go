package imageutil

import (
	"image"
	"testing"
)

func TestVerkleinere(t *testing.T) {
	faelle := []struct {
		name                  string
		breite, hoehe         int
		willBreite, willHoehe int
	}{
		{"passt hinein und bleibt", 300, 400, 300, 400},
		{"zu breit und zu hoch", 1200, 1800, 600, 900},
		{"zu hoch: die Höhe begrenzt", 1000, 5000, 180, 900},
		{"zu breit: die Breite begrenzt", 1800, 900, 600, 300},
	}
	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			bild := Verkleinere(image.NewRGBA(image.Rect(0, 0, f.breite, f.hoehe)), 600, 900)
			if got := bild.Bounds(); got.Dx() != f.willBreite || got.Dy() != f.willHoehe {
				t.Errorf("Verkleinere(%d×%d) = %d×%d, erwartet %d×%d", f.breite, f.hoehe, got.Dx(), got.Dy(), f.willBreite, f.willHoehe)
			}
		})
	}
}
