package imageutil

import (
	"image"

	"golang.org/x/image/draw"
)

// Verkleinere liefert das Bild höchstens maxBreite × maxHoehe groß, im Seitenverhältnis des
// Originals. Ein Bild, das hineinpasst, kommt unverändert zurück.
func Verkleinere(bild image.Image, maxBreite, maxHoehe int) image.Image {
	rahmen := bild.Bounds()
	breite, hoehe := rahmen.Dx(), rahmen.Dy()
	if breite <= maxBreite && hoehe <= maxHoehe {
		return bild
	}

	verhaeltnis := float64(breite) / float64(hoehe)
	neueBreite, neueHoehe := breite, hoehe
	if neueBreite > maxBreite {
		neueBreite = maxBreite
		neueHoehe = int(float64(neueBreite) / verhaeltnis)
	}
	if neueHoehe > maxHoehe {
		neueHoehe = maxHoehe
		neueBreite = int(float64(neueHoehe) * verhaeltnis)
	}

	ziel := image.NewRGBA(image.Rect(0, 0, neueBreite, neueHoehe))
	draw.ApproxBiLinear.Scale(ziel, ziel.Bounds(), bild, rahmen, draw.Over, nil)
	return ziel
}
