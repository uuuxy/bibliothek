package tuer

import (
	"context"
	"log/slog"

	"bibliothek/auth"
	"bibliothek/repository"
)

// Einträge der Verwaltung im Protokoll (audit_logs): wohin Mahnlisten und Bestellungen
// gehen, was in den Mails steht und wann eine Klasse ihre Schulbücher abgibt. Ein Eintrag
// nennt den Gegenstand und die Namen der geänderten Felder, keine Mailadresse: Sie bliebe
// bis zur Aufbewahrungsfrist des Protokolls stehen, und die Tilgung kennt sie dort nicht.
const (
	AktionKlassenleitungGeaendert = "KLASSENLEITUNG_GEAENDERT"
	AktionKlassenleitungEntfernt  = "KLASSENLEITUNG_ENTFERNT"
	AktionMailvorlageGeaendert    = "MAILVORLAGE_GEAENDERT"
	AktionLieferantAngelegt       = "LIEFERANT_ANGELEGT"
	AktionLieferantGeaendert      = "LIEFERANT_GEAENDERT"
	AktionLieferantGeloescht      = "LIEFERANT_GELOESCHT"
	AktionFristKlasseGeaendert    = "FRIST_KLASSE_GEAENDERT"
	// Schlagworte: Zusammenführen, Umleiten und Löschen lassen sich nicht zurücknehmen. Die
	// Wörter sind Vokabular des Katalogs und stehen deshalb im Eintrag.
	AktionSchlagwortZusammengefuehrt = "SCHLAGWORT_ZUSAMMENGEFUEHRT"
	AktionSchlagwortVerweis          = "SCHLAGWORT_VERWEIS"
	AktionSchlagwortGeloescht        = "SCHLAGWORT_GELOESCHT"
)

// ProtokolliereVerwaltung schreibt einen Eintrag mit der angemeldeten Person als Bearbeiter.
// Die Änderung gilt auch, wenn das Protokoll klemmt; der Fehlversuch steht im Server-Log.
func (b *Basis) ProtokolliereVerwaltung(ctx context.Context, aktion string, details map[string]any) {
	claims, ok := auth.ClaimsAusKontext(ctx)
	if !ok {
		return
	}
	// Server ohne Datenbank (nackte Testkonstruktion &Server{}): nichts zu schreiben.
	if b.DB == nil || b.DB.Pool == nil {
		return
	}
	if err := repository.NewProtokollAbfragen(b.DB.Pool).
		ProtokolliereAdminAktion(ctx, claims.KontoID, aktion, "", details); err != nil {
		slog.Error("Protokoll fehlgeschlagen", "aktion", aktion, "fehler", err)
	}
}

// ProtokolliereKontoAnlage schreibt USER_CREATE mit der angemeldeten Person als Bearbeiter. Zwei
// Themen legen ein Konto an, Benutzer & Rechte und die Leserdatei beim Anlegen einer Lehrkraft;
// der Eintrag steht deshalb hier. ziel_id ist die Kennung des Kontos: Daran findet die Auskunft
// den Eintrag, auch nachdem das Konto gelöscht ist (repository/dsgvo_konto.go).
func (b *Basis) ProtokolliereKontoAnlage(ctx context.Context, kontoID, email, rolle, vorname, nachname string) {
	b.ProtokolliereVerwaltung(ctx, "USER_CREATE", map[string]any{
		"ziel_id": kontoID, "email": email, "rolle": rolle, "vorname": vorname, "nachname": nachname,
	})
}

// SchreibeAdminProtokoll schreibt einen Eintrag, dessen Details als JSON-Text vorliegen, mit
// der Adresse des Aufrufers oder ohne. Der Vorgang gilt auch, wenn das Protokoll klemmt; der
// Fehlversuch steht im Server-Log.
func (b *Basis) SchreibeAdminProtokoll(ctx context.Context, adminID, aktion, ip, details string) {
	if err := repository.SchreibeAdminProtokoll(ctx, b.DB.Pool, adminID, aktion, ip, details); err != nil {
		slog.Error("audit/idempotenz: schreibvorgang fehlgeschlagen", "fehler", err)
	}
}

// ProtokolliereGeaenderteFelder schreibt den Eintrag einer Änderung mit den Namen der
// geänderten Felder. Ein Speichern, das nichts geändert hat, schreibt keinen Eintrag.
func (b *Basis) ProtokolliereGeaenderteFelder(ctx context.Context, aktion string, details map[string]any, felder []string) {
	if len(felder) == 0 {
		return
	}
	details["felder"] = felder
	b.ProtokolliereVerwaltung(ctx, aktion, details)
}

// GeaenderteFelder liefert die Namen, deren Wert sich geändert hat, in der Reihenfolge der
// Aufzählung.
func GeaenderteFelder(wechsel ...FeldWechsel) []string {
	felder := []string{}
	for _, w := range wechsel {
		if w.Geaendert {
			felder = append(felder, w.Name)
		}
	}
	return felder
}

// FeldWechsel nennt ein Feld und ob sein Wert sich geändert hat.
type FeldWechsel struct {
	Name      string
	Geaendert bool
}
