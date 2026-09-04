package littera

import (
	"context"
	"fmt"
	"strings"
	"time"

	"bibliothek/internal/uebernahme"

	"github.com/jackc/pgx/v5"
)

// PersonenBericht ist die Bilanz des Personenteils.
type PersonenBericht struct {
	QuellLeser    int
	Schueler      int
	Lehrkraefte   int
	Uebersprungen int // Sonstige, Unklare und an einem Fehler gescheiterte

	IstSchueler    int
	IstLehrkraefte int
	AbgleichOK     bool

	// EntleiherIDs bildet Littera-Leser → Ziel ab. Der Ausleihteil braucht sie.
	EntleiherIDs map[string]Entleiher
}

// Entleiher sagt, in welcher Tabelle eine Person gelandet ist. Die Ausleihe muss genau
// eine der beiden Spalten setzen (check_loan_borrower).
type Entleiher struct {
	SchuelerID string
	BenutzerID string
}

// Der DSGVO-Rahmen dieses Imports, bewusst eng gefasst:
//
// Übernommen werden Name, Klasse, Ausweisnummer und Geburtsdatum. Das Geburtsdatum ist
// nicht Beiwerk, sondern der Schlüssel gegen Doppelanlage: unique_schueler_name_gebdatum
// greift nur, wenn es gesetzt ist — ohne es legt der spätere LUSD-Import dieselben
// Schüler ein zweites Mal an.
//
// NICHT übernommen werden Anschrift und E-Mail, obwohl Littera sie führt (Adresse bei
// 1.927 von 1.991). Ihr Zweck laut schema.sql ist der Versand von Schadens-Rechnungen und
// Eltern-Mahnungen; die gepflegte Quelle dafür ist die LUSD. Eine Anschrift aus einem
// Altbestand ist im Zweifel veraltet, und eine Rechnung an die falsche Adresse ist
// schlechter als gar keine Adresse.
const sqlSchuelerEinfuegen = `
	INSERT INTO schueler
		(barcode_id, vorname, nachname, klasse, geburtsdatum,
		 abgaenger_jahr, ist_abgaenger, lusd_id, erstellt_am)
	VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
	RETURNING id`

// benutzer.email ist NOT NULL UNIQUE, im Altbestand aber bei 157 von 158 Lehrkräften
// leer. Ein Platzhalter muss also her, und er muss unzustellbar sein: .invalid ist nach
// RFC 2606 dauerhaft reserviert und wird von keinem Mailserver aufgelöst. Ein erfundener
// Wert unter der Schuldomäne ginge dagegen irgendwann an eine echte, fremde Person.
//
// DIESE Adresse ist auch der Grund, warum importierte Lehrkräfte trotzdem aktiv=true
// bekommen dürfen: Die Anmeldung läuft ausschließlich über IMAP gegen den Schul-
// Mailserver (auth/handlers.go), und littera-4908@littera.invalid gibt es dort nicht.
// Login und Ausweis sind damit über das richtige Feld getrennt — nicht über aktiv, das
// die Omnibox für die Ausweis-Suche braucht.
const platzhalterDomain = "@littera.invalid"

const sqlBenutzerEinfuegen = `
	INSERT INTO benutzer (barcode_id, vorname, nachname, email, rolle, aktiv, erstellt_am)
	VALUES ($1,$2,$3,$4,'kollegium',$5,$6)
	RETURNING id`

// SchreibePersonen überträgt Schüler und Lehrkräfte.
//
// Sonstige (Praktikanten, Sekretariat, Fachbereichs-Sammelkonten) und Unklare werden
// bewusst NICHT geschrieben: Bei Personendaten ist eine ausgelassene Zeile das kleinere
// Übel als eine falsch einsortierte. Ihre Ausleihen fallen damit weg und werden im
// Ausleihteil einzeln als FEHLER gemeldet, nicht stillschweigend verschluckt.
func (s *Schreiber) SchreibePersonen(ctx context.Context, ab *Altbestand) (PersonenBericht, error) {
	bericht := PersonenBericht{
		QuellLeser:   len(ab.Leser),
		EntleiherIDs: make(map[string]Entleiher, len(ab.Leser)),
	}

	vorherS, vorherL, err := s.zaehlePersonen(ctx)
	if err != nil {
		return bericht, err
	}

	lauf := &personenlauf{s: s, bericht: &bericht,
		belegteAusweise: map[string]bool{}, belegteMails: map[string]bool{}}
	if err := lauf.vorbelegen(ctx); err != nil {
		return bericht, err
	}
	if err := lauf.alleBatches(ctx, ab.Leser); err != nil {
		return bericht, err
	}

	nachherS, nachherL, err := s.zaehlePersonen(ctx)
	if err != nil {
		return bericht, fmt.Errorf("der Abgleich nach der Übernahme schlug fehl: %w", err)
	}
	bericht.IstSchueler, bericht.IstLehrkraefte = nachherS-vorherS, nachherL-vorherL
	bericht.AbgleichOK = bericht.IstSchueler == bericht.Schueler &&
		bericht.IstLehrkraefte == bericht.Lehrkraefte
	return bericht, nil
}

func (s *Schreiber) zaehlePersonen(ctx context.Context) (schueler, lehrkraefte int, err error) {
	err = s.pool.QueryRow(ctx, `
		SELECT (SELECT count(*) FROM schueler),
		       (SELECT count(*) FROM benutzer WHERE rolle = 'kollegium')
	`).Scan(&schueler, &lehrkraefte)
	if err != nil {
		return 0, 0, fmt.Errorf("konnte die Personen nicht zählen: %w", err)
	}
	return schueler, lehrkraefte, nil
}

type personenlauf struct {
	s               *Schreiber
	bericht         *PersonenBericht
	belegteAusweise map[string]bool
	belegteMails    map[string]bool
}

// vorbelegen liest die schon vergebenen Ausweisnummern und Adressen ein — dieselbe
// Vorsichtsmaßnahme wie bei den Barcodes: Eine Kollision mit vorhandenen Zeilen kostet
// sonst die ganze Person.
func (p *personenlauf) vorbelegen(ctx context.Context) error {
	if err := p.lade(ctx, `SELECT barcode_id FROM schueler WHERE deleted_at IS NULL`, p.belegteAusweise); err != nil {
		return err
	}
	if err := p.lade(ctx, `SELECT barcode_id FROM benutzer WHERE barcode_id IS NOT NULL`, p.belegteAusweise); err != nil {
		return err
	}
	return p.lade(ctx, `SELECT lower(email) FROM benutzer`, p.belegteMails)
}

func (p *personenlauf) lade(ctx context.Context, sql string, ziel map[string]bool) error {
	rows, err := p.s.pool.Query(ctx, sql)
	if err != nil {
		return fmt.Errorf("konnte die vorhandenen Personen nicht lesen: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var wert string
		if err := rows.Scan(&wert); err != nil {
			return fmt.Errorf("konnte die vorhandenen Personen nicht lesen: %w", err)
		}
		ziel[wert] = true
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("konnte die vorhandenen Personen nicht lesen: %w", err)
	}
	return nil
}

func (p *personenlauf) alleBatches(ctx context.Context, leser []Leser) error {
	for i := 0; i < len(leser); i += p.s.opt.BatchGroesse {
		ende := min(i+p.s.opt.BatchGroesse, len(leser))
		if err := p.einBatch(ctx, leser[i:ende]); err != nil {
			return fmt.Errorf("abgebrochen ab Leser %s: %w", leser[i].ID, err)
		}
	}
	return nil
}

func (p *personenlauf) einBatch(ctx context.Context, batch []Leser) error {
	tx, err := p.s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("konnte die Transaktion nicht öffnen: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }() //nolint:errcheck

	for _, l := range batch {
		if err := p.einePerson(ctx, tx, l); err != nil {
			return err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("der COMMIT schlug fehl – keine Person dieses Batches wurde geschrieben: %w", err)
	}
	return nil
}

func (p *personenlauf) einePerson(ctx context.Context, tx pgx.Tx, l Leser) error {
	if l.Art == ArtSonstige || l.Art == ArtUnbekannt {
		p.bericht.Uebersprungen++
		p.s.prot.Fehler(l.ID, l.Lesernummer,
			"weder Schüler noch Lehrkraft (Praktikant, Sammelkonto oder unklare Gruppe) – nicht übernommen")
		return nil
	}

	var neu Entleiher
	erg, err := uebernahme.ImSavepoint(ctx, tx, "littera_id="+l.ID, func(sp pgx.Tx) error {
		var innerErr error
		neu, innerErr = p.schreibePerson(ctx, sp, l)
		return innerErr
	})
	if err != nil {
		return err
	}
	if !erg.Uebernommen {
		p.bericht.Uebersprungen++
		p.s.prot.Fehler(l.ID, l.Lesernummer,
			"Person übersprungen – "+uebernahme.BeschreibeFehler(erg.Zurueckgerollt))
		return nil
	}

	p.bericht.EntleiherIDs[l.ID] = neu
	if neu.SchuelerID != "" {
		p.bericht.Schueler++
	} else {
		p.bericht.Lehrkraefte++
	}
	return nil
}

func (p *personenlauf) schreibePerson(ctx context.Context, tx pgx.Tx, l Leser) (Entleiher, error) {
	if l.Art == ArtLehrkraft {
		return p.schreibeLehrkraft(ctx, tx, l)
	}
	return p.schreibeSchueler(ctx, tx, l)
}

func (p *personenlauf) schreibeSchueler(ctx context.Context, tx pgx.Tx, l Leser) (Entleiher, error) {
	jahr, ok := p.abgangsjahr(l)
	if !ok {
		return Entleiher{}, fmt.Errorf(
			"%w: aus der Klasse %q lässt sich kein Abgangsjahr ableiten, schueler.abgaenger_jahr ist NOT NULL",
			uebernahme.ErrZeile, l.Klasse)
	}

	var geburtsdatum *time.Time
	if g, ok := GeburtsdatumAus(l.Geburtsdatum, p.s.opt.Jetzt); ok {
		geburtsdatum = &g
	}

	// lusd_id trägt die Herkunft: schueler hat keine JSONB-Spalte, und ohne eine
	// wiedererkennbare Marke wäre nach dem Import nicht mehr feststellbar, welche Zeilen
	// aus Littera stammen — die Wiederholungssperre in PruefeZielbestand hängt daran.
	herkunft := "littera:" + l.ID

	var id string
	err := tx.QueryRow(ctx, sqlSchuelerEinfuegen,
		p.ausweis(l), p.kuerze(l, "vorname", l.Vorname, uebernahme.MaxMedientyp),
		p.kuerze(l, "nachname", l.Nachname, uebernahme.MaxMedientyp),
		p.kuerze(l, "klasse", l.Klasse, uebernahme.MaxKlasse),
		geburtsdatum, jahr, l.Art == ArtAbgegangen, herkunft, p.s.opt.Jetzt,
	).Scan(&id)
	if err != nil {
		return Entleiher{}, fmt.Errorf("beim Schüler %s %s: %w", l.Vorname, l.Nachname, err)
	}
	return Entleiher{SchuelerID: id}, nil
}

// abgangsjahr rechnet das Abgangsjahr aus der Klasse.
//
// Die Gruppe „Abgegangen" trägt als Klassenbezeichnung nur „Ab" — daraus ist nichts
// abzuleiten. Für sie gilt das laufende Schuljahr: Sie sind bereits weg, das Jahr steuert
// nur noch die Stapel-Archivierung, und ist_abgaenger sagt die Wahrheit ohnehin.
func (p *personenlauf) abgangsjahr(l Leser) (int, bool) {
	if jahr, ok := AbgaengerJahr(l.Klasse, p.s.opt.SchuljahrEnde); ok {
		return jahr, true
	}
	if l.Art == ArtAbgegangen {
		return p.s.opt.SchuljahrEnde, true
	}
	return 0, false
}

func (p *personenlauf) schreibeLehrkraft(ctx context.Context, tx pgx.Tx, l Leser) (Entleiher, error) {
	var id string
	err := tx.QueryRow(ctx, sqlBenutzerEinfuegen,
		uebernahme.Nullbar(p.ausweis(l)),
		p.kuerze(l, "vorname", l.Vorname, uebernahme.MaxMedientyp),
		p.kuerze(l, "nachname", l.Nachname, uebernahme.MaxMedientyp),
		p.mailadresse(l), !p.s.opt.LehrerInaktiv, p.s.opt.Jetzt,
	).Scan(&id)
	if err != nil {
		return Entleiher{}, fmt.Errorf("bei der Lehrkraft %s %s: %w", l.Vorname, l.Nachname, err)
	}
	return Entleiher{BenutzerID: id}, nil
}

// ausweis liefert die Ausweisnummer und weicht bei Kollision auf die Littera-interne
// Nummer aus. schueler.barcode_id ist unter aktiven Zeilen eindeutig; im Altbestand
// kollidieren zwei Leser über dieselbe Lesernummer.
func (p *personenlauf) ausweis(l Leser) string {
	nummer := l.Lesernummer
	if nummer != "" && !p.belegteAusweise[nummer] {
		p.belegteAusweise[nummer] = true
		return nummer
	}
	ersatz := "L-" + l.ID
	grund := "Ausweisnummer bereits vergeben"
	if nummer == "" {
		grund = "keine Lesernummer im Altbestand"
	}
	p.s.prot.Warnung(l.ID, nummer, grund+" – Ausweis "+ersatz+" vergeben, Karte muss neu gedruckt werden")
	p.belegteAusweise[ersatz] = true
	return ersatz
}

// mailadresse liefert die echte Adresse oder einen unzustellbaren Platzhalter.
//
// Die kollidierende Adresse steht bewusst NICHT im Protokoll — dort stand sie bis zum
// 23.08.2026 als Kennung. `littera_import.log` ist eine unverschlüsselte Datei im
// Arbeitsverzeichnis ohne Frist und ohne Löschregel; bei einer Leser-Übernahme
// kollidieren Adressen reihenweise (Familien mit einer gemeinsamen Adresse, Sammelkonten),
// und jede Kollision hätte eine echte Adresse dort abgelegt. Wer sie braucht, findet sie
// in der Quelldatei neben der Lesernummer — die Zeile bleibt damit reparierbar.
func (p *personenlauf) mailadresse(l Leser) string {
	echt := lowerTrim(l.EMail)
	if echt != "" && !p.belegteMails[echt] {
		p.belegteMails[echt] = true
		return echt
	}
	ersatz := "littera-" + l.ID + platzhalterDomain
	if echt != "" {
		p.s.prot.Warnung(l.ID, l.Lesernummer, "E-Mail-Adresse bereits vergeben – Platzhalter "+ersatz+" eingesetzt")
	} else {
		p.s.prot.Warnung(l.ID, "", "keine E-Mail im Altbestand – Platzhalter "+ersatz+
			" eingesetzt (benutzer.email ist NOT NULL); vor dem ersten Mailversand nachtragen")
	}
	p.belegteMails[ersatz] = true
	return ersatz
}

func (p *personenlauf) kuerze(l Leser, feld, wert string, max int) string {
	return uebernahme.Kuerze(uebernahme.Kuerzung{Protokoll: p.s.prot, QuellID: l.ID, Kennung: l.Lesernummer, Feld: feld, Wert: wert, Max: max})
}

// lowerTrim bringt eine Adresse auf die Form, in der benutzer.email verglichen wird.
func lowerTrim(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}
