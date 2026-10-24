package jobs

import (
	"context"
	"log/slog"
	"time"

	"bibliothek/repository"
)

// Die Läufe der Anonymisierung nach DSGVO. Getrennt von cron.go, weil dort steht, wann etwas
// läuft, und hier, was gelöscht und anonymisiert wird: die Fristen aus dem Fachkonzept und die
// einzige Stelle, an der das Programm von sich aus Personendaten entfernt. Das gehört nicht
// zwischen die Anmeldungen beim Cron.

// ── DSGVO: Bearbeiter an alten Ausleihen ─────────────────────────────────────────

// FuehreBearbeiterAnonymisierungAus leert an Ausleihen, die seit mehr als 14 Tagen
// abgeschlossen sind, die Kennung des Bearbeiters. Das erfüllt die Datensparsamkeit nach DSGVO
// für die Person an der Theke.
func (z *Zeitplan) FuehreBearbeiterAnonymisierungAus(ctx context.Context) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	anzahl, err := repository.AnonymisiereBearbeiterAlterAusleihen(ctx, z.pool)
	if err != nil {
		slog.Error("Scheduler GDPR Anonymize: Error anonymizing operator IDs", "fehler", err)
		return
	}
	slog.Info("Scheduler GDPR Anonymize: anonymized loans (returned > 14 days ago)", "anzahl", anzahl)

	// Eintrag des Systems im Protokoll.
	if anzahl > 0 {
		if err := z.protokollAbfragen.ProtokolliereSystemAktion(ctx, "ausleihen", "ANONYMIZE",
			"GDPR 14-Tage-Anonymisierung der Bearbeiter-IDs",
			map[string]any{
				"betroffene_ausleihen": anzahl,
				"schwellwert_tage":     14,
				"ausgefuehrt_am":       time.Now().UTC().Format(time.RFC3339),
			},
		); err != nil {
			slog.Error("audit: ANONYMIZE konnte nicht protokolliert werden", "fehler", err)
		}
	}
}

// ── DSGVO: Anonymisierung alter Leser (180 Tage im Papierkorb, Karenz nach dem Abgang) ──

// fuehreSchuelerAnonymisierungAus anonymisiert Schüler, die
//   - seit mehr als 180 Tagen im Papierkorb liegen (deleted_at < NOW - 180 Tage) oder
//   - länger als die Karenz (Einstellung abgaenger_karenz_tage, Vorgabe 90) Abgänger sind
//     (abgaenger_seit) und keine offenen Vorgänge mehr haben.
//
// Vorname, Nachname und Klasse werden geleert oder gehasht, anonymized_at wird gesetzt.
func (z *Zeitplan) fuehreSchuelerAnonymisierungAus(ctx context.Context) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	// Bei unlesbaren Einstellungen wird nicht anonymisiert. Die Vorgabe von 90 Tagen wäre
	// kein sicherer Rückfall: Eine kleinere Karenz wählt mehr Zeilen, und hat die Schule 365
	// Tage eingestellt, träfe ein einziger Lesefehler die Abgänger zwischen Tag 91 und 365.
	einst, err := repository.NewEinstellungsAbfragen(z.pool).Lade(ctx)
	if err != nil {
		slog.Error("Scheduler GDPR Anonymize: Einstellungen nicht lesbar, Lauf übersprungen", "fehler", err)
		return
	}
	karenzTage := repository.AbgaengerKarenzTageOderVorgabe(einst)
	// Anweisung und Bedingung stehen in repository/: dieselbe Bedingung, die die Selbstprüfung
	// als Rückstand zählt.
	anzahl, err := repository.AnonymisiereSchueler(ctx, z.pool, karenzTage)
	if err != nil {
		slog.Error("Scheduler GDPR Anonymize: Error anonymizing old students", "fehler", err)
		return
	}

	// Scheitert das Löschen der Fotos, läuft die Tilgung der Spuren trotzdem; die nächste
	// Nacht räumt die Fotos nach.
	if fotoFehler := repository.LoescheFotosAnonymisierterSchueler(ctx, z.pool); fotoFehler != nil {
		slog.Error("Scheduler GDPR Anonymize: Fotos anonymisierter Schüler konnten nicht gelöscht werden", "fehler", fotoFehler)
	}

	// Die Spuren anonymisierter Schüler in den übrigen Tabellen tilgen. Die Zeile in leser zu
	// leeren reicht nicht: Klarname (audit_log.details) und LUSD-ID (audit_logs.details) lebten sonst
	// bis zum Ende der Aufbewahrung des Protokolls (24 Monate) weiter, bis zu anderthalb Jahre
	// Personenbezug nach der Löschung. Das Merkmal ist anonymized_at, damit auch der Altbestand
	// nachgezogen wird.
	z.bereinigeAnonymisierteSchuelerSpuren(ctx)

	if anzahl > 0 {
		slog.Info("Scheduler GDPR Anonymize: successfully anonymized old student records", "anzahl", anzahl)
		if err := z.protokollAbfragen.ProtokolliereSystemAktion(ctx, "schueler", "ANONYMIZE",
			"DSGVO Anonymisierung alter Datensätze (Soft-Delete > 180T oder Abgänger > Karenzzeit)",
			map[string]any{
				"betroffene_schueler": anzahl,
				"karenz_tage":         karenzTage,
				"ausgefuehrt_am":      time.Now().UTC().Format(time.RFC3339),
			},
		); err != nil {
			slog.Error("audit: ANONYMIZE konnte nicht protokolliert werden", "fehler", err)
		}
	} else {
		slog.Info("Scheduler GDPR Anonymize: no old students found to anonymize.")
	}
}

// bereinigeAnonymisierteSchuelerSpuren tilgt die Personendaten anonymisierter Schüler aus den
// übrigen Tabellen, die fuehreSchuelerAnonymisierungAus selbst nicht anfasst. Die Anweisungen
// kommen aus repository.SpurTilgungen, derselben Liste, die auch die endgültige Löschung und der
// Abgang aus der LUSD nehmen; jobs/dsgvo_spuren_paarung_pg_test.go hält die Paarung fest.
//
// Das Merkmal ist anonymized_at IS NOT NULL, jede Nacht über den ganzen Bestand (idempotent).
// Jede Anweisung steht einzeln im Log; scheitert eine (etwa am Trigger gegen Änderungen auf
// audit_log, den es nur in manchem Altbestand gibt), laufen die übrigen weiter. Das ist der
// Unterschied zur Transaktion der endgültigen Löschung.
func (z *Zeitplan) bereinigeAnonymisierteSchuelerSpuren(ctx context.Context) {
	ids, err := repository.LadeAnonymisierteSchuelerIDs(ctx, z.pool)
	if err != nil {
		slog.Error("Scheduler GDPR Anonymize: anonymisierte Schüler konnten nicht gelesen werden", "fehler", err)
		return
	}
	if len(ids) == 0 {
		return
	}

	for _, tilgung := range repository.SpurTilgungen() {
		if n, err := tilgung.Tilge(ctx, z.pool, ids, "DSGVO-Anonymisierung"); err != nil {
			slog.Error("Scheduler GDPR Anonymize: Spur konnte nicht getilgt werden", "spur", tilgung.Beschreibung, "fehler", err)
		} else if n > 0 {
			slog.Info("Scheduler GDPR Anonymize: Spur bereinigt", "spur", tilgung.Beschreibung, "anzahl", n)
		}
	}
}
