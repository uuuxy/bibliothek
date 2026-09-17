package api

import (
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"bibliothek/apierrors"
	"bibliothek/repository"
)

// Die Barcode-Liste der Theke (Stufe 2 des Offline-Baus, Commit 12, OFFEN.md 2.2).
//
// Ohne Netz muss der Theken-Rechner eine nackte Ziffernfolge selbst einordnen: Buch oder
// Ausweis? Die Vorsilben B-/LMF- und S-/L- sagen es; die Littera-Etiketten des Altbestands
// und die alten Ausweise sagen es nicht — ein Schülerausweis liefert gemessen
// `B97601826457`, ein Buchetikett eine 13-stellige EAN. Verwechselt der Rechner beides,
// laufen die folgenden Bücher auf die falsche Person, und das fällt erst beim Nachbuchen
// auf (entschieden am 13.09.2026).
//
// Deshalb hält der Rechner die Barcodes aller Exemplare, auch der ausgesonderten (das
// Nachbuchen holt ein zurückgekommenes Buch in den Umlauf). Ein 13-stelliges Littera-Etikett
// rechnet der Rechner zuerst selbst auf die Nummer zurück (frontend/src/lib/litteraEtikett.js,
// dieselben Prüffälle wie der Server); steht die Nummer auf der Liste, ist sie ein Buch; sonst
// gilt sie als unklar und sperrt die Zuordnung, bis ein eindeutiger Ausweis kommt. KEINE
// Personendaten: nur Buchnummern.
//
// Bewusst OHNE LIMIT — anders als jede andere Listen-Route (vgl. api/audit_limit_pg_test.go).
// Eine halbe Liste wäre schlimmer als keine: Die fehlenden Bücher gälten offline als
// „unklar", und niemand sähe, dass es an der Kappung liegt. Der Umfang ist durch den
// Bestand begrenzt und wächst nicht von selbst weiter (rund 35.000 Exemplare erwartet,
// je Zeile ~14 Byte). Größe und Dauer stehen im Log jeder Auslieferung, damit die Annahme
// am Server überprüfbar bleibt statt geschätzt.

// buchbarcodesWarnAb: Ab dieser Antwortgröße steht eine Warnung im Log. Kein Abbruch —
// die Liste bleibt richtig; die Zeile sagt nur, dass die Annahme „passt bequem" kippt.
const buchbarcodesWarnAb = 4 << 20 // 4 MB

// BuchbarcodesResponse ist die Liste für den Theken-Rechner.
type BuchbarcodesResponse struct {
	// Stand ist der Merker dieser Fassung — derselbe Wert wie im ETag. Der Rechner legt
	// ihn neben die Liste und schickt ihn beim nächsten Holen mit.
	Stand string `json:"stand"`
	// Anzahl spart dem Rechner das Zählen und macht die Antwort im Log lesbar.
	Anzahl int `json:"anzahl"`
	// Barcodes: alle Buchnummern, aufsteigend — nur Nummern, keine Personendaten.
	Barcodes []string `json:"barcodes"`
}

// BuchbarcodesHandler liefert die Barcodes aller Exemplare, auch der ausgesonderten.
// @Summary      Buch-Barcodes für die Theke
// @Description  Alle Barcodes aller Exemplare (auch ausgesonderter), damit die Theke ohne Netz Buch von Ausweis unterscheiden kann. Littera-Etiketten rechnet die Theke selbst auf die Nummer zurück. Mit ETag; unverändert antwortet 304.
// @Tags         theke
// @Produce      json
// @Success      200 {object} BuchbarcodesResponse
// @Success      304 "unverändert"
// @Router       /action/buchbarcodes [get]
func (s *Server) BuchbarcodesHandler() http.HandlerFunc {
	return apierrors.Wrap(func(w http.ResponseWriter, r *http.Request) error {
		ctx := r.Context()
		begonnen := time.Now()

		// Der Stand kommt aus Anzahl und jüngster Änderung — eine Abfrage über den Index,
		// ohne die Liste zu lesen. Ein unveränderter Bestand spart damit die ganze
		// Auslieferung. Ausgesonderte zählen nicht mit: Sie fehlen auch in der Liste,
		// also muss ihr Wegfallen den Stand ändern.
		kennzahl, err := repository.LiesBuchbarcodeStand(ctx, s.DB.Pool)
		if err != nil {
			return apierrors.Internal("Buch-Barcodes konnten nicht gezählt werden", err)
		}
		stand := buchbarcodesStand(kennzahl)

		// Unverändert: Der Rechner behält seine Liste. Das ist der Normalfall — der
		// Bestand ändert sich selten, angemeldet wird täglich.
		if passtStand(r.Header.Get("If-None-Match"), stand) {
			w.Header().Set("ETag", `"`+stand+`"`)
			w.WriteHeader(http.StatusNotModified)
			return nil
		}

		barcodes, err := repository.ListeBuchbarcodes(ctx, s.DB.Pool, kennzahl.Anzahl)
		if err != nil {
			return apierrors.Internal("Buch-Barcodes konnten nicht gelesen werden", err)
		}
		antwort := BuchbarcodesResponse{Stand: stand, Anzahl: len(barcodes), Barcodes: barcodes}

		w.Header().Set("ETag", `"`+stand+`"`)
		w.Header().Set("Cache-Control", "no-cache") // erneut fragen, aber der ETag darf sparen
		bytes, err := schreibeVielleichtGepackt(w, r, antwort)
		if err != nil {
			// Nach dem ersten geschriebenen Byte hilft kein Statuscode mehr — nur das Log.
			log.Printf("buchbarcodes: Antwort abgebrochen (%d Einträge): %v", antwort.Anzahl, err)
			return nil
		}
		if bytes > buchbarcodesWarnAb {
			log.Printf("buchbarcodes: WARNUNG %d Einträge, %d Byte in %s — die Liste ist größer als erwartet; vor dem nächsten Ausbau die Annahme prüfen (api/buchbarcodes_handler.go)",
				antwort.Anzahl, bytes, time.Since(begonnen).Round(time.Millisecond))
			return nil
		}
		log.Printf("buchbarcodes: %d Einträge, %d Byte in %s ausgeliefert", antwort.Anzahl, bytes, time.Since(begonnen).Round(time.Millisecond))
		return nil
	})
}

// buchbarcodesStand baut den Merker aus der Kennzahl des Bestands.
func buchbarcodesStand(k repository.BuchbarcodeStand) string {
	roh := fmt.Sprintf("%d|", k.Anzahl)
	if k.Juengste != nil {
		roh += k.Juengste.UTC().Format(time.RFC3339Nano)
	}
	summe := sha256.Sum256([]byte(roh))
	return hex.EncodeToString(summe[:16])
}

// passtStand prüft den If-None-Match-Kopf gegen den Stand. Der Kopf darf mehrere Werte
// tragen und die Anführungszeichen führen; W/ (schwach) gilt hier wie stark, weil der
// Stand ohnehin nur die Fassung benennt.
func passtStand(kopf, stand string) bool {
	if kopf == "" {
		return false
	}
	for _, teil := range strings.Split(kopf, ",") {
		t := strings.TrimSpace(teil)
		t = strings.TrimPrefix(t, "W/")
		if strings.Trim(t, `"`) == stand {
			return true
		}
	}
	return false
}

// schreibeVielleichtGepackt packt die Antwort, wenn der Rechner es anbietet — Caddy
// komprimiert nicht (Caddyfile ohne `encode`), und ohne Packen wandern hier je Anmeldung
// ein paar hundert Kilobyte durchs Schulnetz. Liefert die Zahl der geschriebenen Bytes.
func schreibeVielleichtGepackt(w http.ResponseWriter, r *http.Request, daten any) (int, error) {
	w.Header().Set("Content-Type", "application/json")
	if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
		z := &zaehlendeSchreiber{w: w}
		return z.n, json.NewEncoder(z).Encode(daten)
	}
	w.Header().Set("Content-Encoding", "gzip")
	w.Header().Add("Vary", "Accept-Encoding")
	z := &zaehlendeSchreiber{w: w}
	packer := gzip.NewWriter(z)
	if err := json.NewEncoder(packer).Encode(daten); err != nil {
		return z.n, err
	}
	return z.n, packer.Close()
}

// zaehlendeSchreiber zählt, wie viel wirklich über die Leitung ging — die Zahl im Log
// soll die ausgelieferte Größe nennen, nicht die vor dem Packen.
//
// CodeQL meldet hier go/reflected-xss (Alarm 30, 15.09.2026) und liegt falsch. Der
// gemeldete Weg führt vom Anfragekörper des Mahnwesens über api/mail_sender.go zu
// `part.Write([]byte(req.Body))` — und von dort hierher: `part` ist ein io.Writer, und
// CodeQL löst den Aufruf gegen JEDE Write([]byte)-Methode des Programms auf, auch gegen
// diese. In Wirklichkeit schreibt die Mail nie in diesen Schreiber, und die Antwort
// dieses Handlers enthält keinen Wert aus der Anfrage: nur den Stand (SHA-256 über
// Bestandszahlen), die Anzahl und die Barcodes aus der Datenbank. Wer den Alarm erneut
// sieht: Es ist die Überannäherung der Schnittstellen-Auflösung, nicht der Content-Type.
type zaehlendeSchreiber struct {
	w http.ResponseWriter
	n int
}

func (z *zaehlendeSchreiber) Write(p []byte) (int, error) {
	n, err := z.w.Write(p)
	z.n += n
	return n, err
}
