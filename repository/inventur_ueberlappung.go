package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
)

// bereichUeberlappung ist das Ergebnis der syntaktischen Scope-Prüfung. Der dritte
// Ausgang existiert, weil sich die Kreuz-Kombination Signatur vs. Filter nicht an den
// Scope-Feldern entscheiden lässt: Ob "BIB Deu" und Fach "Deutsch" kollidieren, hängt
// daran, ob ein Titel BEIDE Dimensionen trägt — das weiß nur der Bestand.
type bereichUeberlappung int

const (
	ueberlapptNie bereichUeberlappung = iota
	ueberlapptImmer
	ueberlapptJeNachBestand
)

// bestimmeUeberlappung entscheidet syntaktisch, ob zwei Inventur-Scopes gemeinsame
// Exemplare treffen können. Bewusst konservativ, um legitime parallele Inventuren
// (Signatur "Deu" neben "Mat", Klasse 5 neben 6) NICHT zu blockieren. Alles, was hier
// nicht sicher entscheidbar ist — Signatur vs. Filter, unbekannte Scope-Typen — geht als
// ueberlapptJeNachBestand an den Bestandsabgleich, statt still als "keine Überlappung"
// durchzurutschen (so entstand das frühere Residualrisiko aus 6ef1449c).
func bestimmeUeberlappung(aArt string, a InventurBereich, bArt string, b InventurBereich) bereichUeberlappung {
	// Global überdeckt den gesamten Bestand — überlappt mit jedem anderen Scope.
	if aArt == "global" || bArt == "global" {
		return ueberlapptImmer
	}
	// Zwei Signatur-Scopes: Überlappung, wenn eine Signatur Präfix der anderen ist
	// (case-insensitiv, genau wie das Scannen). "BIB" enthält "BIB Deu".
	if aArt == "signature" && bArt == "signature" {
		as := strings.ToLower(strings.TrimSpace(zeigerStr(a.Signatur)))
		bs := strings.ToLower(strings.TrimSpace(zeigerStr(b.Signatur)))
		if as != "" && bs != "" && (strings.HasPrefix(as, bs) || strings.HasPrefix(bs, as)) {
			return ueberlapptImmer
		}
		return ueberlapptNie
	}
	// Zwei Filter-Scopes: Überlappung, wenn Fach UND Klasse kompatibel sind (nil =
	// Platzhalter „alle"). Fach "Deutsch" (alle Klassen) überlappt "Deutsch/Kl. 5".
	if aArt == "filter" && bArt == "filter" {
		if istFachVereinbar(a.Fach, b.Fach) && istJahrgangVereinbar(a.Jahrgang, b.Jahrgang) {
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

func istFachVereinbar(a, b *string) bool {
	if a == nil || b == nil {
		return true // eine Seite meint „alle Fächer"
	}
	return strings.EqualFold(strings.TrimSpace(*a), strings.TrimSpace(*b))
}

func istJahrgangVereinbar(a, b *int) bool {
	return a == nil || b == nil || *a == *b
}

// istGleicherBereich spiegelt die Semantik der partiellen Unique-Indizes: EXAKT
// gleiche Scopes werden vom Index als ErrInventurLaeuftBereits abgefangen. Die
// Überlappungsprüfung lässt sie deshalb bewusst durch — sonst käme für den simplen
// „läuft schon"-Fall die (unpassendere) Überlappungs-Meldung. Muss genauso streng
// bleiben wie die Indizes (case-sensitiv): wäre sie laxer, würde ein Paar als identisch
// durchgelassen, das der Index dann NICHT abfängt.
func istGleicherBereich(aArt string, a InventurBereich, bArt string, b InventurBereich) bool {
	if aArt != bArt {
		return false
	}
	switch aArt {
	case "global":
		return true
	case "signature":
		return strings.TrimSpace(zeigerStr(a.Signatur)) == strings.TrimSpace(zeigerStr(b.Signatur))
	case "filter":
		gleicheKlasse := (a.Jahrgang == nil && b.Jahrgang == nil) || (a.Jahrgang != nil && b.Jahrgang != nil && *a.Jahrgang == *b.Jahrgang)
		return zeigerStr(a.Fach) == zeigerStr(b.Fach) && gleicheKlasse
	}
	return false
}

// findeUeberlappendenLauf liefert das Label der ersten offenen Session, deren Scope
// sich mit dem gewünschten überschneidet — leerer String, wenn keine überlappt. Läuft
// unter dem Advisory-Lock des Inventur-Starts, die Prüfung ist also race-frei.
func (a *InventurAbfragen) findeUeberlappendenLauf(ctx context.Context, tx pgx.Tx, bereichArt string, bereich InventurBereich) (string, error) {
	// Erst alle offenen Sessions einlesen, dann prüfen: Der Bestandsabgleich braucht
	// eigene Queries auf derselben Tx-Verbindung, und die ist belegt, solange rows offen sind.
	offene, err := ladeBereicheOffenerLaeufe(ctx, tx)
	if err != nil {
		return "", err
	}

	for _, offen := range offene {
		// Exakt identische Scopes überlässt die Prüfung dem Unique-Index (→ LaeuftBereits);
		// nur eine echte Überschneidung meldet sie als Ueberlappt.
		if istGleicherBereich(bereichArt, bereich, offen.art, offen.bereich) {
			continue
		}
		switch bestimmeUeberlappung(bereichArt, bereich, offen.art, offen.bereich) {
		case ueberlapptImmer:
			return offen.bezeichnung, nil
		case ueberlapptJeNachBestand:
			treffen, err := hatGemeinsamenBestand(ctx, tx, bereich, offen.bereich)
			if err != nil {
				return "", err
			}
			if treffen {
				return offen.bezeichnung, nil
			}
		}
	}
	return "", nil
}

// offenerLauf ist eine Session ohne Abschluss mit ihrem Scope.
type offenerLauf struct {
	art         string
	bereich     InventurBereich
	bezeichnung string
}

// ladeBereicheOffenerLaeufe liest die Sessions ohne Abschluss ganz ein.
func ladeBereicheOffenerLaeufe(ctx context.Context, tx pgx.Tx) ([]offenerLauf, error) {
	rows, err := tx.Query(ctx, `
		SELECT scope_type, scope_signatur, scope_subject, scope_grade, scope_label
		FROM inventur_sessions WHERE abgeschlossen_am IS NULL`)
	if err != nil {
		return nil, fmt.Errorf("offene sessions lesen fehlgeschlagen: %w", err)
	}
	defer rows.Close()

	var offene []offenerLauf
	for rows.Next() {
		var o offenerLauf
		if err := rows.Scan(&o.art, &o.bereich.Signatur, &o.bereich.Fach, &o.bereich.Jahrgang, &o.bezeichnung); err != nil {
			return nil, fmt.Errorf("offene session lesen fehlgeschlagen: %w", err)
		}
		offene = append(offene, o)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return offene, nil
}

// hatGemeinsamenBestand prüft am Bestand, ob es ein Exemplar gibt, das in
// BEIDEN Scopes liegt. Maßstab sind die Dimensions-Prädikate — dieselben, mit denen der
// Scan ein Buch als "im Scope" annimmt: Jedes Exemplar, das beide Sessions annehmen
// würden, kann nur in einer verbucht werden und würde beim Abschluss der anderen als
// Verlust ausgesondert. Ausgesonderte Exemplare zählen nicht (sie kann niemand scannen);
// verliehene schon — sie können während der Inventur zurückkommen.
func hatGemeinsamenBestand(ctx context.Context, tx pgx.Tx, a, b InventurBereich) (bool, error) {
	bedingungA, argsA := a.dimensionBedingung(1)
	bedingungB, argsB := b.dimensionBedingung(1 + len(argsA))
	var treffen bool
	err := tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM buecher_exemplare e
			JOIN buecher_titel t ON t.id = e.titel_id
			WHERE e.ist_ausgesondert = false
			  AND (`+bedingungA+`)
			  AND (`+bedingungB+`)
		)`, append(argsA, argsB...)...).Scan(&treffen)
	if err != nil {
		return false, fmt.Errorf("bestandsabgleich der scopes fehlgeschlagen: %w", err)
	}
	return treffen, nil
}
