package repository

import (
	"context"
	"fmt"
	"time"

	"bibliothek/pkg/schulzeit"
)

// Renner ist ein Eintrag der „Renner"-Liste inkl. Drill-Down-Feldern
// (Fachbereich/Systematik/Erscheinungsjahr) für Frontend-Filter.
type Renner struct {
	ID               string `json:"id"`
	Titel            string `json:"titel"`
	Autor            string `json:"autor"`
	CoverURL         string `json:"cover_url"`
	Fachbereich      string `json:"fachbereich,omitempty"`
	Systematik       string `json:"systematik,omitempty"`
	Erscheinungsjahr int    `json:"erscheinungsjahr,omitempty"`
	Anzahl           int    `json:"count"`
}

// Ladenhueter ist ein Eintrag der „Ladenhüter"-Liste inkl. Drill-Down-Feldern.
type Ladenhueter struct {
	// ID ist Pflicht, auch wenn die Liste sie nicht anzeigt: Sie ist der einzige
	// eindeutige Schlüssel. Zwei Titel dürfen legitim gleich heissen und beide keine
	// ISBN haben (etwa zwei Ausgaben desselben Werks) — ohne ID kollidierten sie im
	// Frontend zu einem doppelten each-Key und rissen die Statistik-Ansicht ab.
	ID               string `json:"id"`
	Titel            string `json:"titel"`
	Autor            string `json:"autor"`
	ISBN             string `json:"isbn"`
	LetzteAusleihe   string `json:"letzte_aus"`
	Fachbereich      string `json:"fachbereich,omitempty"`
	Systematik       string `json:"systematik,omitempty"`
	Erscheinungsjahr int    `json:"erscheinungsjahr,omitempty"`
}

// BestandKennzahlen bündelt alle aggregierten Zahlen aus einem Scan über
// buecher_exemplare (statt mehrerer Subselects pro Kennzahl).
type BestandKennzahlen struct {
	GesamtBestand    int
	AktiverBestand   int // physisch vorhanden = nicht ausgesondert
	AktuellVerliehen int
	// VerloreneExemplare: nur echte Abgänge (aussonderung_grund VERLUST oder
	// BESCHAEDIGUNG). Bewusst Aussortiertes und Bestandskorrekturen zählen nicht.
	VerloreneExemplare int
	// Wiederbeschaffungswert: Summe der Einkaufspreise der echten Verluste
	// (VERLUST/BESCHAEDIGUNG) — also der Bücher, die tatsächlich nachgekauft werden
	// müssen. Kuratiert Aussortiertes und Bestandskorrekturen bleiben aussen vor.
	WiederbeschaffungswertDefekt float64
	VerlustQuote                 float64 // echte Verluste / Gesamtbestand
	Zirkulationsquote            float64 // verliehen / aktiver Bestand
}

// BestandsFilterBedingung mappt den ?type=-Parameter auf ein serverkontrolliertes
// SQL-Fragment. Lernmittel sind an buecher_titel.ist_lernmittel erkennbar (Migration
// 093) — dieselbe Spalte wie im Ausleih-Limit, im OPAC und in der Löschfrist.
func BestandsFilterBedingung(typ string) (fragment string, normalform string) {
	switch typ {
	case "lmf":
		return "AND t.ist_lernmittel", "lmf"
	case "freihand":
		return "AND NOT t.ist_lernmittel", "freihand"
	default:
		return "", "alle"
	}
}

// LadeBestandKennzahlen liefert Verlust-, Finanz- und Zirkulationszahlen in
// einem einzigen aggregierten Statement.
func LadeBestandKennzahlen(ctx context.Context, db Datenbank, typFilter string) (*BestandKennzahlen, error) {
	// Als "Verlust" zählen ausschliesslich unfreiwillige Abgänge: VERLUST (nicht
	// auffindbar) und BESCHAEDIGUNG (Schadensfall). Bewusst nicht enthalten sind
	// AUSSORTIERT (veraltet/verschlissen, kuratierte Entfernung) und Bestandskorrektur
	// (Import-/Sync-Anpassung, "kein echter Abgang") — sie würden
	// Verlustquote und Wiederbeschaffungswert künstlich aufblähen. aussonderung_grund
	// ist per chk_aussonderung_grund nur bei ist_ausgesondert gesetzt.
	// Die Grenze ist die des Abgangsbuchs: Ein bestelltes Exemplar, das nie eintraf und
	// als verloren ausgebucht wurde, war nie im Bestand und ist kein Verlust.
	const istVerlust = sqlIstAbgang + " AND e.aussonderung_grund IN ('VERLUST', 'BESCHAEDIGUNG')"

	// Die offenen Ausleihen stehen als CTE mit DISTINCT vor der Abfrage: So wählt der Planer
	// bei wachsendem Bestand einen Hash-Join statt einer verschachtelten Schleife, und ein
	// Exemplar mit zwei offenen Ausleihen verdoppelt keine Zeile.
	q := fmt.Sprintf(`
		WITH aktive_ausleihen AS (
			SELECT DISTINCT exemplar_id FROM ausleihen WHERE rueckgabe_am IS NULL
		)
		SELECT
			COUNT(*)::int AS gesamt,
			COUNT(*) FILTER (WHERE NOT e.ist_ausgesondert)::int AS aktiv,
			COUNT(*) FILTER (WHERE al.exemplar_id IS NOT NULL AND NOT e.ist_ausgesondert)::int AS verliehen,
			COUNT(*) FILTER (WHERE %[1]s)::int AS verlorene,
			COALESCE(SUM(e.einkaufspreis) FILTER (WHERE %[1]s), 0)::float8 AS wiederbeschaffung,
			CASE WHEN COUNT(*) = 0 THEN 0.0
				 ELSE ROUND(COUNT(*) FILTER (WHERE %[1]s) * 100.0 / COUNT(*), 2)
			END::float8 AS verlust_quote,
			CASE WHEN COUNT(*) FILTER (WHERE NOT e.ist_ausgesondert) = 0 THEN 0.0
				 ELSE ROUND(COUNT(*) FILTER (WHERE al.exemplar_id IS NOT NULL AND NOT e.ist_ausgesondert) * 100.0
					   / COUNT(*) FILTER (WHERE NOT e.ist_ausgesondert), 2)
			END::float8 AS zirkulationsquote
		FROM buecher_exemplare e
		JOIN buecher_titel t ON t.id = e.titel_id
		LEFT JOIN aktive_ausleihen al ON al.exemplar_id = e.id
		-- Gezählt wird, was im Bestand ist oder war; ein bestelltes Exemplar zählt ab dem Eintreffen.
		WHERE %[3]s %[2]s
	`, istVerlust, typFilter, sqlWarImBestand)

	k := &BestandKennzahlen{}
	err := db.QueryRow(ctx, q).Scan(
		&k.GesamtBestand, &k.AktiverBestand, &k.AktuellVerliehen, &k.VerloreneExemplare,
		&k.WiederbeschaffungswertDefekt, &k.VerlustQuote, &k.Zirkulationsquote,
	)
	if err != nil {
		return nil, err
	}
	return k, nil
}

// AusleihZeitraumBedingung bildet den Parameter ?zeitraum= auf ein SQL-Stück für die Rangliste
// der Renner ab; die Werte bestimmt der Server, nie die Anfrage.
//
// Das Schuljahr wird in Go in der Zeitzone der Schule bestimmt (schulzeit), nicht per
// CURRENT_DATE in der Zeitzone der Sitzung (UTC): Am 1. August und am 1. Januar zwischen 0 und
// 1 Uhr Berliner Zeit läge der Stichtag sonst ein Jahr daneben. Das Datum wird als ISO-Literal
// eingesetzt und ist kein Wert der Anfrage.
func AusleihZeitraumBedingung(zeitraum string) string {
	switch zeitraum {
	case "schuljahr":
		return "AND a.ausgeliehen_am >= '" + schuljahresBeginn(schulzeit.Jetzt()).Format("2006-01-02") + "'::date"
	case "monat":
		return "AND a.ausgeliehen_am >= CURRENT_DATE - INTERVAL '30 days'"
	default:
		return ""
	}
}

// schuljahresBeginn: der 1. August des laufenden Schuljahres zu t (Juli gehört noch
// zum vorigen Schuljahr).
func schuljahresBeginn(t time.Time) time.Time {
	jahr := t.Year()
	if t.Month() < time.August {
		jahr--
	}
	return time.Date(jahr, time.August, 1, 0, 0, 0, 0, t.Location())
}

// LadeRenner liefert die meistausgeliehenen Titel.
func LadeRenner(ctx context.Context, db Datenbank, ausleihenFilter, typFilter string, limit int) ([]Renner, error) {
	renner := []Renner{}
	q := fmt.Sprintf(`
		SELECT t.id, t.titel, coalesce(t.autor, ''), coalesce(t.cover_url, ''),
		       coalesce(t.subject, ''), coalesce(t.signatur, ''), coalesce(t.erscheinungsjahr, 0),
		       COUNT(a.id) AS count
		FROM buecher_titel t
		JOIN buecher_exemplare e ON t.id = e.titel_id
		JOIN ausleihen a ON e.id = a.exemplar_id
		WHERE 1=1 %s %s
		GROUP BY t.id, t.titel, t.autor, t.cover_url, t.subject, t.signatur, t.erscheinungsjahr
		ORDER BY count DESC
		LIMIT %d
	`, ausleihenFilter, typFilter, limit)
	rows, err := db.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var p Renner
		if err := rows.Scan(&p.ID, &p.Titel, &p.Autor, &p.CoverURL, &p.Fachbereich, &p.Systematik, &p.Erscheinungsjahr, &p.Anzahl); err != nil {
			return nil, zeileUnlesbar(err)
		}
		renner = append(renner, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return renner, nil
}

// LadeLadenhueter liefert die Ladenhüter: entweder seit >2 Jahren nicht mehr
// ausgeliehen, oder noch nie ausgeliehen und bereits seit >2 Jahren im Bestand. Der
// Bestandsalter-Filter (MIN(e.erstellt_am)) verhindert, dass frisch gekaufte Neuzugänge
// — nie ausgeliehen, weil brandneu — sofort auf der Aussonderungsliste landen und
// versehentlich entsorgt werden.
func LadeLadenhueter(ctx context.Context, db Datenbank, typFilter string, limit int) ([]Ladenhueter, error) {
	ladenhueter := []Ladenhueter{}
	// t.id mitliefern: Es wird ohnehin danach gruppiert (eine Zeile je Titel), war aber
	// nicht Teil der Projektion — dem Client fehlte damit der eindeutige Schlüssel.
	q := fmt.Sprintf(`
		SELECT t.id, t.titel, coalesce(t.autor, ''), coalesce(t.isbn, ''),
		       coalesce(t.subject, ''), coalesce(t.signatur, ''), coalesce(t.erscheinungsjahr, 0),
		       MAX(a.ausgeliehen_am) AS last_loan
		FROM buecher_titel t
		LEFT JOIN buecher_exemplare e ON t.id = e.titel_id
		LEFT JOIN ausleihen a ON e.id = a.exemplar_id
		WHERE 1=1 %s
		GROUP BY t.id, t.titel, t.autor, t.isbn, t.subject, t.signatur, t.erscheinungsjahr
		HAVING MAX(a.ausgeliehen_am) < NOW() - INTERVAL '2 years'
		    OR (MAX(a.ausgeliehen_am) IS NULL AND MIN(e.erstellt_am) < NOW() - INTERVAL '2 years')
		ORDER BY last_loan ASC NULLS FIRST
		LIMIT %d
	`, typFilter, limit)
	rows, err := db.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var l Ladenhueter
		var letzteAusleihe *time.Time
		if err := rows.Scan(&l.ID, &l.Titel, &l.Autor, &l.ISBN, &l.Fachbereich, &l.Systematik, &l.Erscheinungsjahr, &letzteAusleihe); err != nil {
			return nil, zeileUnlesbar(err)
		}
		l.LetzteAusleihe = "Nie ausgeliehen"
		if letzteAusleihe != nil {
			l.LetzteAusleihe = letzteAusleihe.In(schulzeit.Zone()).Format(datumsformat)
		}
		ladenhueter = append(ladenhueter, l)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return ladenhueter, nil
}

// MonatsTrendPunkt ist ein Monat der Aktivitäts-Zeitreihe: wie viele Ausleihen
// getätigt und wie viele Rückgaben verbucht wurden. Zwei Serien derselben Einheit
// (Vorgänge/Monat) → eine gemeinsame Y-Achse im Frontend, kein Dual-Axis.
type MonatsTrendPunkt struct {
	Monat      string `json:"monat"` // "2026-07"
	Ausleihen  int    `json:"ausleihen"`
	Rueckgaben int    `json:"rueckgaben"`
}

// LadeMonatsTrend liefert die letzten 12 Monate Ausleih-/Rückgabe-Aktivität als
// lückenlose (zero-gefüllte) Zeitreihe. Ausleihen zählen nach ausgeliehen_am, Rückgaben
// nach rueckgabe_am — dieselbe Ausleihe kann also in unterschiedlichen Monaten in beide
// Serien fallen. Der typeFilter (LMF/Freihand) bindet an t.titel. Bewusst anonym: keine
// Schülerdaten.
func LadeMonatsTrend(ctx context.Context, db Datenbank, typFilter string) ([]MonatsTrendPunkt, error) {
	trend := []MonatsTrendPunkt{}
	q := fmt.Sprintf(`
		WITH monate AS (
			SELECT date_trunc('month', m) AS monat_start
			FROM generate_series(
				date_trunc('month', CURRENT_DATE) - INTERVAL '11 months',
				date_trunc('month', CURRENT_DATE),
				INTERVAL '1 month'
			) AS m
		),
		ausl AS (
			SELECT date_trunc('month', a.ausgeliehen_am) AS monat, COUNT(*) AS n
			FROM ausleihen a
			JOIN buecher_exemplare e ON e.id = a.exemplar_id
			JOIN buecher_titel t ON t.id = e.titel_id
			WHERE a.ausgeliehen_am >= date_trunc('month', CURRENT_DATE) - INTERVAL '11 months' %[1]s
			GROUP BY 1
		),
		rueck AS (
			SELECT date_trunc('month', a.rueckgabe_am) AS monat, COUNT(*) AS n
			FROM ausleihen a
			JOIN buecher_exemplare e ON e.id = a.exemplar_id
			JOIN buecher_titel t ON t.id = e.titel_id
			WHERE a.rueckgabe_am IS NOT NULL
			  AND a.rueckgabe_am >= date_trunc('month', CURRENT_DATE) - INTERVAL '11 months' %[1]s
			GROUP BY 1
		)
		SELECT to_char(m.monat_start, 'YYYY-MM'),
		       COALESCE(al.n, 0)::int,
		       COALESCE(r.n, 0)::int
		FROM monate m
		LEFT JOIN ausl al ON al.monat = m.monat_start
		LEFT JOIN rueck r ON r.monat = m.monat_start
		ORDER BY m.monat_start
	`, typFilter)
	rows, err := db.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var p MonatsTrendPunkt
		if err := rows.Scan(&p.Monat, &p.Ausleihen, &p.Rueckgaben); err != nil {
			return nil, zeileUnlesbar(err)
		}
		trend = append(trend, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return trend, nil
}
