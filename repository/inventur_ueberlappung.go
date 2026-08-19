package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
)

// scopeUeberlappung ist das Ergebnis der syntaktischen Scope-Prüfung. Der dritte
// Ausgang existiert, weil sich die Kreuz-Kombination Signatur vs. Filter nicht an den
// Scope-Feldern entscheiden lässt: Ob "BIB Deu" und Fach "Deutsch" kollidieren, hängt
// daran, ob ein Titel BEIDE Dimensionen trägt — das weiß nur der Bestand.
type scopeUeberlappung int

const (
	ueberlapptNie scopeUeberlappung = iota
	ueberlapptImmer
	ueberlapptJeNachBestand
)

// inventurScopesUeberlappen entscheidet syntaktisch, ob zwei Inventur-Scopes gemeinsame
// Exemplare treffen können. Bewusst konservativ, um legitime parallele Inventuren
// (Signatur "Deu" neben "Mat", Klasse 5 neben 6) NICHT zu blockieren. Alles, was hier
// nicht sicher entscheidbar ist — Signatur vs. Filter, unbekannte Scope-Typen — geht als
// ueberlapptJeNachBestand an den Bestandsabgleich, statt still als "keine Überlappung"
// durchzurutschen (so entstand das frühere Residualrisiko aus 6ef1449c).
func inventurScopesUeberlappen(aType string, a InventurScope, bType string, b InventurScope) scopeUeberlappung {
	// Global überdeckt den gesamten Bestand — überlappt mit jedem anderen Scope.
	if aType == "global" || bType == "global" {
		return ueberlapptImmer
	}
	// Zwei Signatur-Scopes: Überlappung, wenn eine Signatur Präfix der anderen ist
	// (case-insensitiv, genau wie das Scannen). "BIB" enthält "BIB Deu".
	if aType == "signature" && bType == "signature" {
		as := strings.ToLower(strings.TrimSpace(zeigerStr(a.Signatur)))
		bs := strings.ToLower(strings.TrimSpace(zeigerStr(b.Signatur)))
		if as != "" && bs != "" && (strings.HasPrefix(as, bs) || strings.HasPrefix(bs, as)) {
			return ueberlapptImmer
		}
		return ueberlapptNie
	}
	// Zwei Filter-Scopes: Überlappung, wenn Fach UND Klasse kompatibel sind (nil =
	// Platzhalter „alle"). Fach "Deutsch" (alle Klassen) überlappt "Deutsch/Kl. 5".
	if aType == "filter" && bType == "filter" {
		if dimensionKompatibel(a.Subject, b.Subject) && gradeKompatibel(a.Grade, b.Grade) {
			return ueberlapptImmer
		}
		return ueberlapptNie
	}
	return ueberlapptJeNachBestand
}

func zeigerStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func dimensionKompatibel(a, b *string) bool {
	if a == nil || b == nil {
		return true // eine Seite meint „alle Fächer"
	}
	return strings.EqualFold(strings.TrimSpace(*a), strings.TrimSpace(*b))
}

func gradeKompatibel(a, b *int) bool {
	return a == nil || b == nil || *a == *b
}

// inventurScopesIdentisch spiegelt die Semantik der partiellen Unique-Indizes: EXAKT
// gleiche Scopes werden vom Index als ErrInventurLaeuftBereits abgefangen. Die
// Überlappungsprüfung lässt sie deshalb bewusst durch — sonst käme für den simplen
// „läuft schon"-Fall die (unpassendere) Überlappungs-Meldung. Muss genauso streng
// bleiben wie die Indizes (case-sensitiv): wäre sie laxer, würde ein Paar als identisch
// durchgelassen, das der Index dann NICHT abfängt.
func inventurScopesIdentisch(aType string, a InventurScope, bType string, b InventurScope) bool {
	if aType != bType {
		return false
	}
	switch aType {
	case "global":
		return true
	case "signature":
		return strings.TrimSpace(zeigerStr(a.Signatur)) == strings.TrimSpace(zeigerStr(b.Signatur))
	case "filter":
		gleicheKlasse := (a.Grade == nil && b.Grade == nil) || (a.Grade != nil && b.Grade != nil && *a.Grade == *b.Grade)
		return zeigerStr(a.Subject) == zeigerStr(b.Subject) && gleicheKlasse
	}
	return false
}

// findeUeberlappendeSession liefert das Label der ersten offenen Session, deren Scope
// sich mit dem gewünschten überschneidet — leerer String, wenn keine überlappt. Läuft
// unter dem Advisory-Lock des Inventur-Starts, die Prüfung ist also race-frei.
func (r *InventoryRepository) findeUeberlappendeSession(ctx context.Context, tx pgx.Tx, scopeType string, scope InventurScope) (string, error) {
	// Erst ALLE offenen Sessions einlesen, dann prüfen: Der Bestandsabgleich braucht
	// eigene Queries auf derselben Tx-Verbindung, und die ist busy, solange rows offen sind.
	type offeneSession struct {
		typ   string
		scope InventurScope
		label string
	}
	rows, err := tx.Query(ctx, `
		SELECT scope_type, scope_signatur, scope_subject, scope_grade, scope_label
		FROM inventur_sessions WHERE abgeschlossen_am IS NULL`)
	if err != nil {
		return "", fmt.Errorf("offene sessions lesen fehlgeschlagen: %w", err)
	}
	defer rows.Close()

	var offene []offeneSession
	for rows.Next() {
		var o offeneSession
		if err := rows.Scan(&o.typ, &o.scope.Signatur, &o.scope.Subject, &o.scope.Grade, &o.label); err != nil {
			return "", fmt.Errorf("offene session lesen fehlgeschlagen: %w", err)
		}
		offene = append(offene, o)
	}
	if err := rows.Err(); err != nil {
		return "", err
	}
	rows.Close()

	for _, offen := range offene {
		// Exakt identische Scopes überlässt die Prüfung dem Unique-Index (→ LaeuftBereits);
		// nur eine ECHTE Überschneidung meldet sie als Ueberlappt.
		if inventurScopesIdentisch(scopeType, scope, offen.typ, offen.scope) {
			continue
		}
		switch inventurScopesUeberlappen(scopeType, scope, offen.typ, offen.scope) {
		case ueberlapptImmer:
			return offen.label, nil
		case ueberlapptJeNachBestand:
			treffen, err := scopesTreffenGemeinsamenBestand(ctx, tx, scope, offen.scope)
			if err != nil {
				return "", err
			}
			if treffen {
				return offen.label, nil
			}
		}
	}
	return "", nil
}

// scopesTreffenGemeinsamenBestand prüft am Bestand, ob es ein Exemplar gibt, das in
// BEIDEN Scopes liegt. Maßstab sind die Dimensions-Prädikate — dieselben, mit denen der
// Scan ein Buch als "im Scope" annimmt: Jedes Exemplar, das beide Sessions annehmen
// würden, kann nur in einer verbucht werden und würde beim Abschluss der anderen als
// Verlust ausgesondert. Ausgesonderte Exemplare zählen nicht (sie kann niemand scannen);
// verliehene schon — sie können während der Inventur zurückkommen.
func scopesTreffenGemeinsamenBestand(ctx context.Context, tx pgx.Tx, a, b InventurScope) (bool, error) {
	condA, argsA := a.DimensionBedingung(1)
	condB, argsB := b.DimensionBedingung(1 + len(argsA))
	var treffen bool
	err := tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM buecher_exemplare e
			JOIN buecher_titel t ON t.id = e.titel_id
			WHERE e.ist_ausgesondert = false
			  AND (`+condA+`)
			  AND (`+condB+`)
		)`, append(argsA, argsB...)...).Scan(&treffen)
	if err != nil {
		return false, fmt.Errorf("bestandsabgleich der scopes fehlgeschlagen: %w", err)
	}
	return treffen, nil
}
