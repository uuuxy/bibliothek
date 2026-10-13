package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
)

// bereichUeberlappung ist das Ergebnis der Prüfung zweier Bereiche an ihren Feldern. Den
// dritten Ausgang gibt es, weil sich Signatur gegen Fach und Jahrgang nicht an den Feldern
// entscheiden lässt: Ob „BIB Deu" und das Fach „Deutsch" zusammenstoßen, hängt daran, ob ein
// Titel beides trägt, und das weiß nur der Bestand.
type bereichUeberlappung int

const (
	ueberlapptNie bereichUeberlappung = iota
	ueberlapptImmer
	ueberlapptJeNachBestand
)

// bestimmeUeberlappung entscheidet an den Feldern, ob zwei Bereiche gemeinsame Exemplare
// treffen können. Vorsichtig, damit Läufe nebeneinander möglich bleiben (Signatur „Deu" neben
// „Mat", Jahrgang 5 neben 6). Was sich hier nicht sicher entscheiden lässt (Signatur gegen Fach
// und Jahrgang, eine unbekannte Art), geht als ueberlapptJeNachBestand an den Abgleich mit dem
// Bestand und gilt nicht als „keine Überlappung".
func bestimmeUeberlappung(aArt string, a InventurBereich, bArt string, b InventurBereich) bereichUeberlappung {
	// „global" deckt den ganzen Bestand und überlappt mit jedem anderen Bereich.
	if aArt == "global" || bArt == "global" {
		return ueberlapptImmer
	}
	// Zwei Signaturen überlappen, wenn eine Präfix der anderen ist, ohne Rücksicht auf Groß-
	// und Kleinschreibung wie beim Scannen: „BIB" enthält „BIB Deu".
	if aArt == "signature" && bArt == "signature" {
		as := strings.ToLower(strings.TrimSpace(zeigerText(a.Signatur)))
		bs := strings.ToLower(strings.TrimSpace(zeigerText(b.Signatur)))
		if as != "" && bs != "" && (strings.HasPrefix(as, bs) || strings.HasPrefix(bs, as)) {
			return ueberlapptImmer
		}
		return ueberlapptNie
	}
	// Zwei Filter überlappen, wenn Fach und Jahrgang vereinbar sind (nil steht für „alle"): Das
	// Fach „Deutsch" für alle Jahrgänge überlappt „Deutsch, Jahrgang 5".
	if aArt == "filter" && bArt == "filter" {
		if istFachVereinbar(a.Fach, b.Fach) && istJahrgangVereinbar(a.Jahrgang, b.Jahrgang) {
			return ueberlapptImmer
		}
		return ueberlapptNie
	}
	return ueberlapptJeNachBestand
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

// istGleicherBereich folgt den partiellen Unique-Indizes: Genau gleiche Bereiche fängt der
// Index als ErrInventurLaeuftBereits. Die Prüfung auf Überlappung lässt sie deshalb durch;
// sonst käme für „läuft schon" die weniger passende Meldung der Überlappung. Sie muss so streng
// bleiben wie die Indizes (Groß- und Kleinschreibung zählt): Wäre sie großzügiger, ließe sie
// ein Paar als gleich durch, das der Index nicht fängt.
func istGleicherBereich(aArt string, a InventurBereich, bArt string, b InventurBereich) bool {
	if aArt != bArt {
		return false
	}
	switch aArt {
	case "global":
		return true
	case "signature":
		return strings.TrimSpace(zeigerText(a.Signatur)) == strings.TrimSpace(zeigerText(b.Signatur))
	case "filter":
		gleicheKlasse := (a.Jahrgang == nil && b.Jahrgang == nil) || (a.Jahrgang != nil && b.Jahrgang != nil && *a.Jahrgang == *b.Jahrgang)
		return zeigerText(a.Fach) == zeigerText(b.Fach) && gleicheKlasse
	}
	return false
}

// findeUeberlappendenLauf liefert die Bezeichnung des ersten offenen Laufs, dessen Bereich sich
// mit dem gewünschten überschneidet, oder "" ohne Überschneidung. Läuft unter der Sperre des
// Starts.
func (a *InventurAbfragen) findeUeberlappendenLauf(ctx context.Context, tx pgx.Tx, bereichArt string, bereich InventurBereich) (string, error) {
	// Erst alle offenen Läufe einlesen, dann prüfen: Der Abgleich mit dem Bestand braucht
	// eigene Abfragen auf derselben Verbindung, und die ist belegt, solange rows offen ist.
	offene, err := ladeBereicheOffenerLaeufe(ctx, tx)
	if err != nil {
		return "", err
	}

	for _, offen := range offene {
		// Genau gleiche Bereiche überlässt die Prüfung dem Unique-Index; sie meldet nur die
		// echte Überschneidung.
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

// offenerLauf ist ein Lauf ohne Abschluss mit seinem Bereich.
type offenerLauf struct {
	art         string
	bereich     InventurBereich
	bezeichnung string
}

// ladeBereicheOffenerLaeufe liest die Läufe ohne Abschluss ganz ein.
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

// hatGemeinsamenBestand prüft am Bestand, ob ein Exemplar in beiden Bereichen liegt. Maßstab
// sind dieselben Einschränkungen, mit denen der Scan ein Buch als „im Bereich" annimmt: Ein
// Exemplar, das beide Läufe annähmen, kann nur in einem verbucht werden und würde beim
// Abschluss des anderen als Verlust ausgesondert. Ausgesonderte Exemplare zählen nicht (niemand
// kann sie scannen), verliehene schon: Sie können während der Inventur zurückkommen.
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
