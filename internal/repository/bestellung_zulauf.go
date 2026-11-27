package repository

import (
	"context"
	"time"

	"bibliothek/internal/pkg/schulzeit"
)

// ZulaufExemplar ist ein bestelltes Exemplar, das noch nicht eingetroffen ist, mit seinem Titel
// und seiner Bestellung.
type ZulaufExemplar struct {
	ExemplarID   string
	TitelID      string
	ErstelltAm   time.Time
	ZustandNotiz string
	Titel        string
	ISBN         string
	CoverURL     string
	// Bestellung, Lieferant und Bestelldatum fehlen an Exemplaren, die bestellt wurden, bevor
	// ein Exemplar seine Bestellung trug. Als Zeiger: Ein NULL in einer der drei Spalten bräche
	// sonst das Lesen der ganzen Liste ab.
	BestellungID  *string
	LieferantName *string
	Bestelldatum  *time.Time
	// Der Stand der Bestellung, wie ihn die Bestellhistorie liest. Ohne Bestellung ist
	// MitBestaetigung false und die Zeiger bleiben leer.
	MitBestaetigung   bool
	BestaetigtAm      *time.Time
	BestaetigtDurch   *string
	MailGescheitertAm *time.Time
}

// LadeExemplareImZulauf liefert die bestellten, noch nicht eingetroffenen Exemplare, das jüngste
// zuerst. Eine fehlende Notiz kommt als leerer Text, aus demselben Grund wie die Zeiger.
func LadeExemplareImZulauf(ctx context.Context, db Datenbank) ([]ZulaufExemplar, error) {
	rows, err := db.Query(ctx, `
		SELECT e.id, e.titel_id, e.erstellt_am, COALESCE(e.zustand_notiz, ''), t.titel, COALESCE(t.isbn, ''),
		       `+sqlCoverOderDNB("t")+`,
		       e.bestellung_id::text, b.lieferant_name, b.bestelldatum,
		       `+sqlBestellungMitBestaetigung+`, b.bestaetigt_am, b.bestaetigt_durch, `+sqlBestellmailOffen+`
		FROM buecher_exemplare e
		JOIN buecher_titel t ON e.titel_id = t.id
		LEFT JOIN bestellungen_verlauf b ON b.id = e.bestellung_id
		WHERE e.ist_ausleihbar = false
		  AND e.bestellstatus IS NOT NULL
		  AND e.ist_ausgesondert = false
		ORDER BY e.erstellt_am DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var exemplare []ZulaufExemplar
	for rows.Next() {
		var e ZulaufExemplar
		if err := rows.Scan(&e.ExemplarID, &e.TitelID, &e.ErstelltAm, &e.ZustandNotiz, &e.Titel, &e.ISBN, &e.CoverURL,
			&e.BestellungID, &e.LieferantName, &e.Bestelldatum,
			&e.MitBestaetigung, &e.BestaetigtAm, &e.BestaetigtDurch, &e.MailGescheitertAm); err != nil {
			return nil, err
		}
		exemplare = append(exemplare, e)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return exemplare, nil
}

// EingebuchtesExemplar ist ein Exemplar, das der Wareneingang freigegeben hat.
type EingebuchtesExemplar struct {
	BarcodeID       string
	Titel           string
	Autor           string
	EtikettGedruckt bool
	// Abholfach nennt den Schüler mit Klasse, für den das Exemplar ins Abholfach gehört; leer heißt
	// ins Regal.
	Abholfach string
}

// BucheZulaufEin gibt die genannten Exemplare aus dem Zulauf frei und liefert die, die es
// traf. Es trifft nur ein Exemplar, das bestellt und nicht ausgesondert ist: Eines, das der
// Händler nicht liefert und das deshalb ausgesondert wurde, kommt so nicht in den Bestand
// zurück und behält seine Notiz. Wartet jemand auf den Titel, geht das Exemplar an ihn wie bei
// einer Rückgabe (BedieneWartendenMitEingang); db ist deshalb am besten eine Transaktion.
func BucheZulaufEin(ctx context.Context, db Datenbank, exemplarIDs []string) ([]EingebuchtesExemplar, error) {
	rows, err := db.Query(ctx, `
		UPDATE buecher_exemplare e
		SET ist_ausleihbar = true, zustand_notiz = '', bestellstatus = NULL
		FROM buecher_titel t
		WHERE e.titel_id = t.id
		  AND e.ist_ausleihbar = false
		  -- Nur echte Zulauf-Exemplare: Ein ausgesondertes (Händler liefert nicht) darf
		  -- „alle einbuchen" nicht wiederbeleben und seine Notiz nicht verlieren.
		  AND e.bestellstatus IS NOT NULL
		  AND e.ist_ausgesondert = false
		  AND e.id = ANY($1)
		RETURNING e.id, e.titel_id, e.barcode_id, t.titel, coalesce(t.autor, '') AS autor, e.etikett_gedruckt
	`, exemplarIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var eingebucht []EingebuchtesExemplar
	var eingebuchteIDs, titelIDs []string
	for rows.Next() {
		var e EingebuchtesExemplar
		var exemplarID, titelID string
		if err := rows.Scan(&exemplarID, &titelID, &e.BarcodeID, &e.Titel, &e.Autor, &e.EtikettGedruckt); err != nil {
			return nil, err
		}
		eingebucht = append(eingebucht, e)
		eingebuchteIDs = append(eingebuchteIDs, exemplarID)
		titelIDs = append(titelIDs, titelID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	rows.Close()
	for i := range eingebucht {
		fuer, err := BedieneWartendenMitEingang(ctx, db, eingebuchteIDs[i], titelIDs[i], schulzeit.Jetzt())
		if err != nil {
			return nil, err
		}
		eingebucht[i].Abholfach = fuer
	}
	return eingebucht, nil
}
