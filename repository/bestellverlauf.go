package repository

import (
	"context"
	"time"
)

// BestellVerlaufPosition ist eine Position einer Bestellung im Verlauf.
type BestellVerlaufPosition struct {
	TitelName   string  `json:"titel_name"`
	ISBN        string  `json:"isbn"`
	Menge       int     `json:"menge"`
	Einzelpreis float64 `json:"einzelpreis"`
	Gesamtpreis float64 `json:"gesamtpreis"`
	// TitelID verweist auf den Titel und kann leer sein: Der Fremdschlüssel steht auf ON DELETE
	// SET NULL, die Bestellung bleibt als Beleg, wenn der Titel aus dem Katalog verschwindet.
	// Die Oberfläche baut daraus keinen Verweis, ohne zu prüfen.
	TitelID string `json:"titel_id,omitempty"`
	// EtikettenOffen zählt die Exemplare dieses Titels ohne gedrucktes Etikett, nicht die
	// dieser Lieferung: Das ist, was die Liste zum Nachdrucken beim Filtern auf den Titel
	// zeigt. Die Zahl trägt den Verweis dorthin; ohne sie öffnete er in der Hälfte der Fälle
	// eine leere Liste. Welche Exemplare aus dieser Lieferung stammen, zeigt die Detailansicht
	// (api/bestellung/bestelldetail.go).
	EtikettenOffen int `json:"etiketten_offen"`
}

// BestellVerlaufEintrag ist eine Bestellung im Verlauf, mit ihren Positionen und dem
// Zustand des Bestätigungs-Wegs.
type BestellVerlaufEintrag struct {
	ID              string                   `json:"id"`
	LieferantName   string                   `json:"lieferant_name"`
	LieferantEmail  string                   `json:"lieferant_email"`
	Kundennummer    string                   `json:"kundennummer"`
	Bestelldatum    time.Time                `json:"bestelldatum"`
	Gesamtbetrag    float64                  `json:"gesamtbetrag"`
	AnzahlExemplare int                      `json:"anzahl_exemplare"`
	Positionen      []BestellVerlaufPosition `json:"positionen"`
	// Mittel ist der Topf ('land' oder 'schultraeger'); leer bei einer alten Bestellung ohne
	// eindeutige Zuordnung.
	Mittel string `json:"mittel"`

	// MitBestaetigung heißt: Diese Bestellung ist mit einem Link zur Bestätigung
	// hinausgegangen, sie hat einen Token. Daran hängt, ob die Oberfläche den Schritt
	// „Bestätigen" zeigt.
	//
	// Es hängt am Token der Bestellung und nicht am heutigen Merkmal des Lieferanten:
	// Hauptlieferant ist immer nur einer, und beim Wechsel verlören sonst alle offenen
	// Bestellungen des bisherigen ihren Schritt, obwohl sie mit Link hinausgegangen sind und
	// weiter warten. Dieselbe Regel nutzt die Übersicht
	// (api/bestellung/bestellhistorie_uebersicht.go).
	MitBestaetigung bool `json:"mit_bestaetigung"`
	// BestaetigtAm ist der Zeitpunkt der Bestätigung; nil, solange sie aussteht.
	BestaetigtAm *time.Time `json:"bestaetigt_am,omitempty"`
	// EtikettenGroesse ist die beim Bestätigen gewählte Größe ('klein' oder 'gross'); nil,
	// solange die Bestätigung aussteht.
	EtikettenGroesse *string `json:"etiketten_groesse,omitempty"`
	// BestaetigtDurch unterscheidet die beiden Wege: 'lieferant' heißt über den verschickten
	// Link bestätigt, 'bibliothek' heißt hier von Hand nachgetragen. Ohne die Unterscheidung
	// sähe eine Vermutung aus wie eine Rückmeldung.
	BestaetigtDurch *string `json:"bestaetigt_durch,omitempty"`
	// LinkAktiv heißt: Für diese Bestellung ist ein gültiger Link zur Bestätigung unterwegs.
	// Der Link selbst steht nirgends; gespeichert ist nur sein Hash.
	LinkAktiv bool `json:"link_aktiv"`
	// LinkGueltigBis ist der Ablauf des Links (nil ohne Link): bis wann der Händler noch
	// bestätigen und Etiketten drucken kann.
	LinkGueltigBis *time.Time `json:"link_gueltig_bis,omitempty"`
	// MailGescheitertAm: Zeitpunkt des letzten gescheiterten Versands der Bestellmail,
	// solange die Bestellung weder bestätigt noch geliefert ist (sqlBestellmailOffen). nil =
	// kein offener gescheiterter Versand.
	MailGescheitertAm *time.Time `json:"mail_gescheitert_am,omitempty"`
}

// LadeBestellVerlauf lädt die Köpfe der Bestellungen, die neuesten zuerst, und dazu je
// Bestellung ihren Platz in der Liste, an dem LadeBestellVerlaufPositionen die Positionen
// einhängt.
func LadeBestellVerlauf(ctx context.Context, db Datenbank, limit int, mittel string) ([]BestellVerlaufEintrag, map[string]int, error) {
	// Ohne JOIN auf lieferanten: Beide Angaben zur Bestätigung stehen an der Bestellung.
	// Deshalb überlebt eine Bestellung ihren gelöschten Lieferanten als vollständiger Beleg
	// (lieferant_id ON DELETE SET NULL).
	args := []any{limit}
	abfrage := `
		SELECT b.id, b.lieferant_name, b.lieferant_email, b.kundennummer, b.bestelldatum,
		       b.gesamtbetrag, b.anzahl_exemplare,
		       ` + sqlBestellungMitBestaetigung + `,
		       b.bestaetigt_am, b.etiketten_groesse, b.bestaetigt_durch,
		       (b.bestaetigungs_token_hash IS NOT NULL
		        AND (b.token_gueltig_bis IS NULL OR b.token_gueltig_bis > now())),
		       b.token_gueltig_bis, coalesce(b.mittel, ''), ` + sqlBestellmailOffen + `
		FROM bestellungen_verlauf b
		WHERE true`
	if bedingung, arg := mittelBedingung(mittel, "b.mittel", len(args)+1); bedingung != "" {
		abfrage += bedingung
		if arg != nil {
			args = append(args, arg)
		}
	}
	abfrage += `
		ORDER BY b.bestelldatum DESC
		LIMIT $1`

	rows, err := db.Query(ctx, abfrage, args...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	bestellungen := make([]BestellVerlaufEintrag, 0)
	platzJeBestellung := map[string]int{}

	for rows.Next() {
		var b BestellVerlaufEintrag
		if err := rows.Scan(&b.ID, &b.LieferantName, &b.LieferantEmail, &b.Kundennummer,
			&b.Bestelldatum, &b.Gesamtbetrag, &b.AnzahlExemplare, &b.MitBestaetigung,
			&b.BestaetigtAm, &b.EtikettenGroesse, &b.BestaetigtDurch, &b.LinkAktiv, &b.LinkGueltigBis, &b.Mittel,
			&b.MailGescheitertAm); err != nil {
			return nil, nil, err
		}
		b.Positionen = []BestellVerlaufPosition{}
		platzJeBestellung[b.ID] = len(bestellungen)
		bestellungen = append(bestellungen, b)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	return bestellungen, platzJeBestellung, nil
}

// LadeBestellVerlaufPositionen lädt die Positionen aller geladenen Bestellungen in einer
// Abfrage und hängt sie über platzJeBestellung ein; den Gesamtpreis rechnet es je Position.
func LadeBestellVerlaufPositionen(ctx context.Context, db Datenbank, bestellungen []BestellVerlaufEintrag, platzJeBestellung map[string]int) error {
	// EtikettOffenBedingung ist dieselbe Bestimmung, die die Liste zum Nachdrucken und ihr
	// Zähler nehmen (api/druck/etiketten_offen.go). Hier abgeschrieben, zeigte der Verweis
	// irgendwann eine andere Zahl als die Liste, die er öffnet.
	rows, err := db.Query(ctx, `
		SELECT p.bestellung_id, p.titel_name, p.isbn, p.menge, p.einzelpreis,
		       coalesce(p.titel_id::text, ''),
		       coalesce(e_counts.etiketten_offen, 0)
		FROM bestellungen_positionen p
		LEFT JOIN (
			SELECT titel_id, count(*) as etiketten_offen
			FROM buecher_exemplare e
			WHERE `+EtikettOffenBedingung+`
			GROUP BY titel_id
		) e_counts ON e_counts.titel_id = p.titel_id
		WHERE p.bestellung_id = ANY($1)
		ORDER BY p.bestellung_id, p.titel_name
	`, geladeneIDs(platzJeBestellung))
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var bestellungID string
		var p BestellVerlaufPosition
		if err := rows.Scan(&bestellungID, &p.TitelName, &p.ISBN, &p.Menge, &p.Einzelpreis,
			&p.TitelID, &p.EtikettenOffen); err != nil {
			return err
		}
		p.Gesamtpreis = float64(p.Menge) * p.Einzelpreis
		if platz, ok := platzJeBestellung[bestellungID]; ok {
			bestellungen[platz].Positionen = append(bestellungen[platz].Positionen, p)
		}
	}
	return rows.Err()
}

// geladeneIDs liefert die IDs der geladenen Bestellungen. Die Abfrage der Positionen fragt nur
// nach ihnen; ohne die Grenze läse sie die Positionen aller Bestellungen und würfe die
// überzähligen weg.
func geladeneIDs(platzJeBestellung map[string]int) []string {
	ids := make([]string, 0, len(platzJeBestellung))
	for id := range platzJeBestellung {
		ids = append(ids, id)
	}
	return ids
}
