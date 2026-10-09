package repository

import (
	"context"
	"fmt"
	"log"
	"time"

	"bibliothek/pkg/schulzeit"
)

// PopularTitle ist ein Eintrag der „Renner"-Liste inkl. Drill-Down-Feldern
// (Fachbereich/Systematik/Erscheinungsjahr) für Frontend-Filter.
type PopularTitle struct {
	ID               string `json:"id"`
	Titel            string `json:"titel"`
	Autor            string `json:"autor"`
	CoverURL         string `json:"cover_url"`
	Fachbereich      string `json:"fachbereich,omitempty"`
	Systematik       string `json:"systematik,omitempty"`
	Erscheinungsjahr int    `json:"erscheinungsjahr,omitempty"`
	Count            int    `json:"count"`
}

// ShelfWarmer ist ein Eintrag der „Ladenhüter"-Liste inkl. Drill-Down-Feldern.
type ShelfWarmer struct {
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

// BestandKennzahlen bündelt alle aggregierten Zahlen aus EINEM Scan über
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
func BestandsFilterBedingung(typ string) (fragment string, normalized string) {
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
func LadeBestandKennzahlen(ctx context.Context, db DBQueryer, typeFilter string) (*BestandKennzahlen, error) {
	// Als "Verlust" zählen ausschliesslich unfreiwillige Abgänge: VERLUST (nicht
	// auffindbar) und BESCHAEDIGUNG (Schadensfall). Bewusst NICHT enthalten sind
	// AUSSORTIERT (veraltet/verschlissen, kuratierte Entfernung) und BESTANDSKORREKTUR
	// (Import-/Sync-Anpassung, laut Migration 043 "kein echter Abgang") — sie würden
	// Verlustquote und Wiederbeschaffungswert künstlich aufblähen. aussonderung_grund
	// ist per chk_aussonderung_grund (Migration 043) nur bei ist_ausgesondert gesetzt.
	// Die Grenze ist die des Abgangsbuchs: Ein bestelltes Exemplar, das nie eintraf und
	// als verloren ausgebucht wurde, war nie im Bestand und ist kein Verlust.
	const istVerlust = SQLIstAbgang + " AND e.aussonderung_grund IN ('VERLUST', 'BESCHAEDIGUNG')"

	// ⚡ Bolt: Extracted active loans subquery into a CTE and restored DISTINCT.
	// This prevents the PostgreSQL planner from executing a suboptimal nested loop left join
	// when the dataset scales, instead allowing a highly efficient parallel hash join,
	// while strictly preventing row duplication if a single copy has multiple active loans.
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
	`, istVerlust, typeFilter, SQLWarImBestand)

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

// AusleihZeitraumBedingung mappt den ?zeitraum=-Parameter auf ein serverkontrolliertes
// SQL-Fragment für das Renner-Ranking (Werte sind nie nutzergesteuertes SQL).
//
// Das Schuljahr wird in Go in der Schulzeitzone bestimmt (schulzeit), nicht per
// CURRENT_DATE in der Sitzungszone der DB (UTC): am 1. August bzw. 1. Januar zwischen
// 0 und 1 Uhr Berliner Zeit lag der Stichtag sonst ein Jahr daneben (Rasterdurchgang
// 02.09.2026). Das Datum wird als ISO-Literal eingesetzt — kein Nutzerwert.
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

// ListeRenner liefert die meistausgeliehenen Titel (best-effort: bei einem
// Query- oder Iterationsfehler wird eine leere Liste statt eines Fehlers geliefert).
func ListeRenner(ctx context.Context, db DBQueryer, ausleihenFilter, typeFilter string, limit int) []PopularTitle {
	popularTitles := []PopularTitle{}
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
	`, ausleihenFilter, typeFilter, limit)
	rows, err := db.Query(ctx, q)
	if err != nil {
		return popularTitles
	}
	defer rows.Close()
	for rows.Next() {
		var p PopularTitle
		// Scan-Fehler nicht stillschweigend überspringen: Laufen Query und Struct
		// auseinander (Spalte ergänzt/entfernt), wäre die Liste sonst einfach leer —
		// nicht von "keine Treffer" zu unterscheiden.
		if err := rows.Scan(&p.ID, &p.Titel, &p.Autor, &p.CoverURL, &p.Fachbereich, &p.Systematik, &p.Erscheinungsjahr, &p.Count); err != nil {
			log.Printf("stats: Renner-Zeile unlesbar: %v", err)
			continue
		}
		popularTitles = append(popularTitles, p)
	}
	// Bei einem Abbruch mitten in der Iteration keine irreführende Teil-Top-Liste
	// zeigen (best-effort-Sektion, daher verwerfen statt 500).
	if err := rows.Err(); err != nil {
		return []PopularTitle{}
	}
	return popularTitles
}

// ListeLadenhueter liefert die Ladenhüter: entweder seit >2 Jahren nicht mehr
// ausgeliehen, ODER noch nie ausgeliehen UND bereits seit >2 Jahren im Bestand. Der
// Bestandsalter-Filter (MIN(e.erstellt_am)) verhindert, dass frisch gekaufte Neuzugänge
// — nie ausgeliehen, weil brandneu — sofort auf der Aussonderungsliste landen und
// versehentlich entsorgt werden. Best-effort mit leerer Liste bei Fehlern.
func ListeLadenhueter(ctx context.Context, db DBQueryer, typeFilter string, limit int) []ShelfWarmer {
	shelfWarmers := []ShelfWarmer{}
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
	`, typeFilter, limit)
	rows, err := db.Query(ctx, q)
	if err != nil {
		return shelfWarmers
	}
	defer rows.Close()
	for rows.Next() {
		var sw ShelfWarmer
		var lastLoan *time.Time
		// Wie oben: ein verschluckter Scan-Fehler sähe aus wie "keine Ladenhüter".
		if err := rows.Scan(&sw.ID, &sw.Titel, &sw.Autor, &sw.ISBN, &sw.Fachbereich, &sw.Systematik, &sw.Erscheinungsjahr, &lastLoan); err != nil {
			log.Printf("stats: Ladenhüter-Zeile unlesbar: %v", err)
			continue
		}
		sw.LetzteAusleihe = "Nie ausgeliehen"
		if lastLoan != nil {
			sw.LetzteAusleihe = lastLoan.Format(dateFormatDE)
		}
		shelfWarmers = append(shelfWarmers, sw)
	}
	// Bei Iterationsabbruch keine irreführende Teil-Ladenhüterliste zeigen.
	if err := rows.Err(); err != nil {
		return []ShelfWarmer{}
	}
	return shelfWarmers
}

// MonatsTrendPunkt ist ein Monat der Aktivitäts-Zeitreihe: wie viele Ausleihen
// getätigt und wie viele Rückgaben verbucht wurden. Zwei Serien derselben Einheit
// (Vorgänge/Monat) → eine gemeinsame Y-Achse im Frontend, kein Dual-Axis.
type MonatsTrendPunkt struct {
	Monat     string `json:"monat"` // "2026-07"
	Ausleihen int    `json:"ausleihen"`
	Ruckgaben int    `json:"rueckgaben"`
}

// LadeMonatsTrend liefert die letzten 12 Monate Ausleih-/Rückgabe-Aktivität als
// lückenlose (zero-gefüllte) Zeitreihe. Ausleihen zählen nach ausgeliehen_am, Rückgaben
// nach rueckgabe_am — dieselbe Ausleihe kann also in unterschiedlichen Monaten in beide
// Serien fallen. Best-effort mit leerer Liste bei Fehlern (wie die übrigen Sektionen);
// der typeFilter (LMF/Freihand) bindet an t.titel. Bewusst anonym: keine Schülerdaten.
func LadeMonatsTrend(ctx context.Context, db DBQueryer, typeFilter string) []MonatsTrendPunkt {
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
	`, typeFilter)
	rows, err := db.Query(ctx, q)
	if err != nil {
		return trend
	}
	defer rows.Close()
	for rows.Next() {
		var p MonatsTrendPunkt
		if err := rows.Scan(&p.Monat, &p.Ausleihen, &p.Ruckgaben); err != nil {
			log.Printf("stats: Monatstrend-Zeile unlesbar: %v", err)
			continue
		}
		trend = append(trend, p)
	}
	if err := rows.Err(); err != nil {
		return []MonatsTrendPunkt{}
	}
	return trend
}
