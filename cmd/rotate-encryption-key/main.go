// Kommando rotate-encryption-key wechselt den APP_ENCRYPTION_KEY, ohne die damit
// verschlüsselten Daten zu verlieren.
//
// Gebraucht wird es, weil ein neuer Schlüssel allein nichts repariert, sondern zerstört:
// Schülerfotos (schueler_fotos.foto_encrypted) und das gespeicherte SMTP-Passwort
// (mail_settings_config.smtp_password_encrypted) liegen AES-GCM-verschlüsselt in der
// Datenbank. Wer nur die Umgebungsvariable austauscht, hat sie danach unwiederbringlich
// verloren — die Anwendung meldet dann lediglich "entschlüsselung fehlgeschlagen".
//
// Das Kommando entschlüsselt jeden Datensatz mit dem ALTEN und verschlüsselt ihn mit dem
// NEUEN Schlüssel, alles in EINER Transaktion: Entweder ist am Ende alles umgeschlüsselt
// oder nichts. Ein Abbruch mitten im Lauf hinterlässt keinen halb lesbaren Bestand.
//
// Aufruf:
//
//	APP_ENCRYPTION_KEY=<alt> DATABASE_URL=… go run ./cmd/rotate-encryption-key -neu <neu>
//	APP_ENCRYPTION_KEY=<alt> DATABASE_URL=… go run ./cmd/rotate-encryption-key -neu <neu> -pruefen
//
// -pruefen macht einen vollständigen Probelauf ohne zu schreiben. Immer zuerst damit
// laufen lassen, und vorher ein Backup ziehen.
//
// Danach den neuen Wert in die .env eintragen und den Stack neu starten.
package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"bibliothek/internal/crypto"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	neuerSchluessel := flag.String("neu", "", "neuer Schlüssel (32 Zeichen oder 64 Hex-Zeichen); leer = einen erzeugen und nur anzeigen")
	nurPruefen := flag.Bool("pruefen", false, "Probelauf: alles entschlüsseln und neu verschlüsseln, aber nichts speichern")
	flag.Parse()

	if *neuerSchluessel == "" {
		erzeugt := make([]byte, 32)
		if _, err := rand.Read(erzeugt); err != nil {
			log.Fatalf("FATAL: Zufallsschlüssel konnte nicht erzeugt werden: %v", err)
		}
		fmt.Printf("Vorschlag für einen neuen Schlüssel (64 Hex-Zeichen):\n\n  %s\n\n"+
			"Damit erneut aufrufen:\n  -neu %s -pruefen\n", hex.EncodeToString(erzeugt), hex.EncodeToString(erzeugt))
		return
	}

	altSchluessel, err := crypto.GetMasterKey()
	if err != nil {
		log.Fatalf("FATAL: alter Schlüssel (%s) nicht brauchbar: %v", crypto.SchluesselVariable, err)
	}
	neuSchluessel, err := crypto.SchluesselAus(*neuerSchluessel)
	if err != nil {
		log.Fatalf("FATAL: neuer Schlüssel nicht brauchbar: %v", err)
	}
	if string(altSchluessel) == string(neuSchluessel) {
		log.Fatalf("FATAL: alter und neuer Schlüssel sind identisch — nichts zu tun.")
	}

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatalf("FATAL: DATABASE_URL ist nicht gesetzt.")
	}

	ctx, abbrechen := context.WithTimeout(context.Background(), 30*time.Minute)
	defer abbrechen()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		log.Fatalf("FATAL: Datenbankverbindung: %v", err)
	}
	defer pool.Close()

	if *nurPruefen {
		log.Printf("PROBELAUF — es wird nichts geschrieben.")
	}

	gesamt, err := rotiere(ctx, pool, altSchluessel, neuSchluessel, *nurPruefen)
	if err != nil {
		log.Fatalf("FATAL: %v\n\nEs wurde NICHTS verändert (die Transaktion wurde zurückgerollt). "+
			"Der bisherige %s bleibt gültig.", err, crypto.SchluesselVariable)
	}

	if *nurPruefen {
		log.Printf("Probelauf erfolgreich: %d Datensätze ließen sich mit dem alten Schlüssel lesen "+
			"und mit dem neuen schreiben. Jetzt ohne -pruefen wiederholen.", gesamt)
		return
	}

	log.Printf("Fertig: %d Datensätze umgeschlüsselt.", gesamt)
	log.Printf("JETZT in der .env eintragen und den Stack neu starten:\n  %s=%s",
		crypto.SchluesselVariable, *neuerSchluessel)
	log.Printf("Bis dahin läuft die Anwendung mit dem ALTEN Schlüssel und kann die Daten nicht mehr lesen.")
}
