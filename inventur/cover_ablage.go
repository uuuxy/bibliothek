package inventur

import (
	"image"

	"bibliothek/pkg/imageutil"

	"golang.org/x/image/draw"
)

// bereiteBildVor re-enkodiert ein Cover-Bild einheitlich als WebP und skaliert
// übergroße Bilder auf maxWidth x maxHeight herunter. Das Ergebnis ist immer WebP
// (".webp"), damit der lokale Cover-Cache kompakt und einheitlich bleibt.
func bereiteBildVor(bild image.Image, maxBreite, maxHoehe int, qualitaet float32) ([]byte, string, error) {
	rahmen := bild.Bounds()
	breite, hoehe := rahmen.Dx(), rahmen.Dy()

	quelle := bild
	if breite > maxBreite || hoehe > maxHoehe {
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
		quelle = ziel
	}

	ausgabe, err := imageutil.ErzeugeWebP(quelle, qualitaet)
	if err != nil {
		return nil, "", err
	}
	return ausgabe, ".webp", nil
}
