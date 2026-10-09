package littera

import (
	"context"
	"fmt"
	"strings"
	"time"

	"bibliothek/internal/uebernahme"
	"bibliothek/pkg/leserart"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
)

// PersonenBericht ist die Bilanz des Personenteils.
type PersonenBericht struct {
	QuellLeser    int
	Schueler      int
	Kollegium     int // jede Leserzeile im Kollegium, mit Konto oder ohne (leserart.MitKonto)
	Uebersprungen int // an einem Fehler gescheiterte

	IstSchueler  int
	IstKollegium int
	AbgleichOK   bool

	// EntleiherIDs bildet Littera-Leser → Ziel ab. Der Ausleihteil braucht sie.
	EntleiherIDs map[string]Entleiher
}

// Entleiher sagt, in welcher Tabelle eine Person gelandet ist. Die Ausleihe muss genau
// eine der beiden Spalten setzen (check_loan_borrower).
// Entleiher ist die Leserzeile, an der die Ausleihen dieser Person hängen. Vor
// Migration 125 standen hier zwei IDs — eine für Schüler, eine für Lehrkräfte —, und der
// Ausleihteil musste raten, welche gefüllt ist.
type Entleiher struct {
	LeserID string
	// IstSchueler trennt nur noch die Zählung im Bericht und die Dauerleihe.
	IstSchueler bool
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

// Das Konto trägt seit Migration 125 weder Ausweis noch Personenart; beides gehört zur
// Leserzeile, die der Trigger trg_benutzer_hat_leserzeile mit anlegt und deren ID hier
// zurückkommt.
const sqlBenutzerEinfuegen = `
	INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv, erstellt_am)
	VALUES ($1,$2,$3,'kollegium',$4,$5)
	RETURNING leser_id`

const sqlLeserzeileNachtragen = `
	UPDATE leser SET barcode_id = $1, art = $2 WHERE id = $3`

// Praktikum und Fachbereich bekommen kein Konto (leserart.MitKonto, Entscheidung vom
// 30.09.2026): nur die Leserzeile, an der ihre Ausleihen hängen. Keine Platzhalter-Adresse —
// es gibt nichts, womit sie sich anmelden sollten.
const sqlLeserOhneKontoEinfuegen = `
	INSERT INTO leser (barcode_id, vorname, nachname, art, erstellt_am)
	VALUES ($1,$2,$3,$4,$5)
	RETURNING id`

// SchreibePersonen überträgt jeden Leser: Schüler (auch „Abgegangen" und „Im Ausland") in
// die Schülerdatei, alle anderen ins Kollegium.
//
// Ein Leser ohne Art (OhneZuordnung) hält den Lauf an, bevor die erste Person geschrieben
// ist. Bei Personendaten ist eine falsch einsortierte Zeile schlimmer als eine fehlende —
// aber eine fehlende nimmt ihre Ausleihen mit, und das Buch gilt als verfügbar. Deshalb
// weder raten noch auslassen: zuordnen, dann laufen.
func (s *Schreiber) SchreibePersonen(ctx context.Context, ab *Altbestand) (PersonenBericht, error) {
	bericht := PersonenBericht{
		QuellLeser:   len(ab.Leser),
		EntleiherIDs: make(map[string]Entleiher, len(ab.Leser)),
	}
	if offen := OhneZuordnung(ab); len(offen) > 0 {
		return bericht, fmt.Errorf("Leser ohne Zuordnung, es wurde keine Person geschrieben: %v", offen)
	}

	vorherS, vorherK, err := s.zaehlePersonen(ctx)
	if err != nil {
		return bericht, err
	}

	lauf := &personenlauf{s: s, bericht: &bericht,
		belegteAusweise: map[string]bool{}, verbrauchteAusweise: map[string]bool{},
		belegteMails: map[string]bool{}, buchBarcodes: map[string]bool{},
		karten: ab.Ausweisnummern, mehrereKarten: map[string]bool{}}
	for _, id := range ab.AusweisMehrfach {
		lauf.mehrereKarten[id] = true
	}
	if err := lauf.vorbelegen(ctx); err != nil {
		return bericht, err
	}
	if err := lauf.alleBatches(ctx, ab.Leser); err != nil {
		return bericht, err
	}

	nachherS, nachherK, err := s.zaehlePersonen(ctx)
	if err != nil {
		return bericht, fmt.Errorf("der Abgleich nach der Übernahme schlug fehl: %w", err)
	}
	bericht.IstSchueler, bericht.IstKollegium = nachherS-vorherS, nachherK-vorherK
	bericht.AbgleichOK = bericht.IstSchueler == bericht.Schueler &&
		bericht.IstKollegium == bericht.Kollegium
	return bericht, nil
}

// zaehlePersonen zählt die Leserzeilen, nicht die Konten: Praktikum und Fachbereich kommen
// ohne Konto an (bis zum 30.09.2026 zählte hier benutzer mit der Rolle kollegium).
func (s *Schreiber) zaehlePersonen(ctx context.Context) (schueler, kollegium int, err error) {
	err = s.pool.QueryRow(ctx, `
		SELECT (SELECT count(*) FROM leser WHERE art = 'schueler'),
		       (SELECT count(*) FROM leser WHERE art <> 'schueler')
	`).Scan(&schueler, &kollegium)
	if err != nil {
		return 0, 0, fmt.Errorf("konnte die Personen nicht zählen: %w", err)
	}
	return schueler, kollegium, nil
}

type personenlauf struct {
	s               *Schreiber
	bericht         *PersonenBericht
	belegteAusweise map[string]bool
	// verbrauchteAusweise hält die Ersatzvergabe aus jeder Nummer heraus, die schon einmal an
	// einer Leserzeile stand — auch im Papierkorb, auch ausgeschieden (Migration 146) —, jede
	// A-Nummer zusätzlich ohne führende Nullen (sqlVerbrauchteAusweise). Die eigene Karte einer
	// Person prüft nur belegteAusweise: Sie gehört dieser Person.
	verbrauchteAusweise map[string]bool
	belegteMails        map[string]bool
	// buchBarcodes sind für Ausweise gesperrt: Die Theke löst eine Nummer ohne Vorsilbe
	// zuerst als Buch auf (resolveOhnePraefix).
	buchBarcodes map[string]bool
	// karten bildet Leser → Herstellernummer des Ausweises ab (FremdLeserNummer);
	// mehrereKarten nennt die Leser, für die Littera mehr als eine Karte führt.
	karten        map[string]string
	mehrereKarten map[string]bool
}

// vorbelegen liest die schon vergebenen Ausweisnummern und Adressen ein — dieselbe
// Vorsichtsmaßnahme wie bei den Barcodes: Eine Kollision mit vorhandenen Zeilen kostet
// sonst die ganze Person.
func (p *personenlauf) vorbelegen(ctx context.Context) error {
	// EINE Abfrage über alle Ausweisnummern — sie stehen seit Migration 125 an einem Ort.
	if err := p.lade(ctx, `SELECT barcode_id FROM leser WHERE deleted_at IS NULL AND barcode_id IS NOT NULL`, p.belegteAusweise); err != nil {
		return err
	}
	// Die Ersatznummer erfindet dieser Lauf selbst; sie darf keine sein, die schon einmal einer
	// Person gehörte (docs/OFFEN.md 5.23).
	if err := p.lade(ctx, sqlVerbrauchteAusweise, p.verbrauchteAusweise); err != nil {
		return err
	}
	// Die Bücher stehen zu diesem Zeitpunkt schon da: Der Lauf schreibt den Bestand vor den
	// Personen (cmd/littera-altbestand).
	if err := p.lade(ctx, `SELECT barcode_id FROM buecher_exemplare WHERE barcode_id IS NOT NULL`, p.buchBarcodes); err != nil {
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
	if neu.IstSchueler {
		p.bericht.Schueler++
	} else {
		p.bericht.Kollegium++
	}
	return nil
}

// schreibePerson legt die Person nach ihrer Art an: Schüler in die Schülerdatei, im Kollegium
// mit Konto, wo eines dazugehört (leserart.MitKonto), sonst nur die Leserzeile. Die
// Littera-Gruppe steht seit dem 30.09.2026 als Art an der Leserzeile (ZielArt) und nicht mehr
// nur im Protokoll.
func (p *personenlauf) schreibePerson(ctx context.Context, tx pgx.Tx, l Leser) (Entleiher, error) {
	ziel := l.Art.ZielArt()
	switch {
	case leserart.IstSchueler(ziel):
		return p.schreibeSchueler(ctx, tx, l)
	case leserart.MitKonto(ziel):
		return p.schreibeLehrkraft(ctx, tx, l)
	case ziel != "":
		return p.schreibeOhneKonto(ctx, tx, l)
	}
	// SchreibePersonen lässt keinen Leser ohne Art herein (OhneZuordnung); kommt doch einer
	// an, bricht der Lauf ab, statt ihn irgendwo einzusortieren.
	return Entleiher{}, fmt.Errorf("Leser %s hat keine Art (%d)", l.ID, l.Art)
}

func (p *personenlauf) schreibeSchueler(ctx context.Context, tx pgx.Tx, l Leser) (Entleiher, error) {
	jahr, rueckfall := p.abgangsjahr(l)

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
	if rueckfall {
		p.s.prot.Warnung(l.ID, l.Lesernummer, fmt.Sprintf(
			"Klasse „%s“ nennt keinen Jahrgang – Abgangsjahr %d eingesetzt wie bei der Handanlage; "+
				"das richtige setzt der Abgang (Versetzung, LUSD-Import)", l.Klasse, jahr))
	}
	return Entleiher{LeserID: id, IstSchueler: true}, nil
}

// abgangsjahr rechnet das Abgangsjahr aus der Klasse.
//
// Die Gruppe „Abgegangen" trägt als Klassenbezeichnung nur „Ab" — daraus ist nichts
// abzuleiten. Für sie gilt das laufende Schuljahr: Sie sind bereits weg, ist_abgaenger sagt
// die Wahrheit, und das Jahr bestimmt nur noch, ab wann die Löschung greifen darf
// (repository.PredikatAbgaengerLoeschung).
//
// Jede andere Klasse ohne Jahrgang („AUS" der Gruppe „Im Ausland") bekommt, was die
// Anwendung dafür überall einsetzt, Handanlage wie LUSD-Import (repository.AbgangsjahrOhneKlasse),
// und rueckfall=true für die Warnung. Bis zum 28.09.2026 wurde ein solcher Schüler samt seinen
// Ausleihen ausgelassen — mit der Begründung, ein geratenes Jahr archiviere ihn still; eine
// Archivierung nach dem Abgangsjahr gibt es aber nicht (siehe AbgangsjahrOhneKlasse).
func (p *personenlauf) abgangsjahr(l Leser) (jahr int, rueckfall bool) {
	if jahr, ok := AbgaengerJahr(l.Klasse, p.s.opt.SchuljahrEnde); ok {
		return jahr, false
	}
	if l.Art == ArtAbgegangen {
		return p.s.opt.SchuljahrEnde, false
	}
	return repository.AbgangsjahrOhneKlasse(p.s.opt.Jetzt), true
}

func (p *personenlauf) schreibeLehrkraft(ctx context.Context, tx pgx.Tx, l Leser) (Entleiher, error) {
	art := l.Art.ZielArt()
	var leserID string
	err := tx.QueryRow(ctx, sqlBenutzerEinfuegen,
		p.kuerze(l, "vorname", l.Vorname, uebernahme.MaxMedientyp),
		p.kuerze(l, "nachname", l.Nachname, uebernahme.MaxMedientyp),
		p.mailadresse(l), !p.s.opt.LehrerInaktiv, p.s.opt.Jetzt,
	).Scan(&leserID)
	if err != nil {
		return Entleiher{}, fmt.Errorf("bei der Lehrkraft %s %s: %w", l.Vorname, l.Nachname, err)
	}
	tag, err := tx.Exec(ctx, sqlLeserzeileNachtragen,
		uebernahme.Nullbar(p.ausweis(l)), art, leserID)
	if err != nil {
		return Entleiher{}, fmt.Errorf("bei der Lehrkraft %s %s: %w", l.Vorname, l.Nachname, err)
	}
	// 0 Zeilen hieße: Das Konto kam ohne Leserzeile. Ohne diese Prüfung liefe die
	// Übernahme durch und meldete eine Lehrkraft als geschrieben, deren Ausweis und Art
	// nirgends stehen — an der Theke wäre sie nicht auffindbar.
	if tag.RowsAffected() == 0 {
		return Entleiher{}, fmt.Errorf("bei der Lehrkraft %s %s: die Leserzeile fehlt",
			l.Vorname, l.Nachname)
	}
	return Entleiher{LeserID: leserID}, nil
}

// schreibeOhneKonto legt Praktikum und Fachbereich als Leserzeile ohne Konto an. Ausweis und
// Name wie im Kollegium; kein Geburtsdatum, keine Klasse (chk_leser_schueler_pflichtfelder
// gilt nur für Schüler).
func (p *personenlauf) schreibeOhneKonto(ctx context.Context, tx pgx.Tx, l Leser) (Entleiher, error) {
	var leserID string
	err := tx.QueryRow(ctx, sqlLeserOhneKontoEinfuegen,
		uebernahme.Nullbar(p.ausweis(l)),
		p.kuerze(l, "vorname", l.Vorname, uebernahme.MaxMedientyp),
		p.kuerze(l, "nachname", l.Nachname, uebernahme.MaxMedientyp),
		l.Art.ZielArt(), p.s.opt.Jetzt,
	).Scan(&leserID)
	if err != nil {
		return Entleiher{}, fmt.Errorf("beim Konto %s %s: %w", l.Vorname, l.Nachname, err)
	}
	return Entleiher{LeserID: leserID}, nil
}

// ausweis liefert die Ausweisnummer und weicht bei Kollision auf die Littera-interne
// Nummer aus. schueler.barcode_id ist unter aktiven Zeilen eindeutig; im Altbestand
// kollidieren zwei Leser über dieselbe Lesernummer.
//
// Die Nummer ist die, die der Ausweis beim Scannen liefert: Steht in FremdLeserNummer eine
// Herstellernummer, gewinnt sie gegen die Lesernummer (fremdnummern.go). Bis zum 15.09.2026
// wurde sie eingelesen und nie geschrieben — nach dem Personenlauf hätte kein alter Ausweis
// seine Person gefunden (OFFEN.md 5.15). Vergeben ist außerdem jede Nummer, die schon ein
// Buch trägt.
func (p *personenlauf) ausweis(l Leser) string {
	nummer := l.Lesernummer
	if karte := p.karten[l.ID]; karte != "" {
		nummer = karte
	} else if len(p.karten) > 0 {
		// Die Schule nutzt Herstellerausweise, für diese Person führt Littera aber keinen: Ein
		// Ausweis in ihrer Hand findet sie nicht — das muss vor dem ersten Scan jemand wissen.
		p.s.prot.Warnung(l.ID, nummer,
			"keine Karte in FremdLeserNummer – Ausweis trägt die Lesernummer, ein vorhandener Herstellerausweis findet die Person nicht")
	}
	if p.mehrereKarten[l.ID] {
		p.s.prot.Warnung(l.ID, nummer,
			"mehrere Karten hinterlegt – die zuletzt angelegte gilt, mit einer älteren findet die Theke niemanden")
	}
	if nummer != "" && !p.belegteAusweise[nummer] && !p.buchBarcodes[nummer] {
		p.belegteAusweise[nummer] = true
		return nummer
	}
	// Auch die Ersatznummer kann schon vergeben sein, etwa von Hand an eine Lehrkraft.
	//
	// „A-" wie Ausweis (16.09.2026) statt des früheren „L-": Ein Buchstabe auf dem
	// Ausweis soll nichts über die Person behaupten — wer jemand ist, steht in den
	// Stammdaten. Die Theke liest „L-" weiterhin, damit Nummern aus früheren Läufen
	// scannen; vergeben wird es nicht mehr.
	//
	// Die Ersatznummer liegt im Nummernkreis des Generators (ausweis_nummer_start): Die
	// Littera-Kennungen reichten 2010 bis 6845 und wachsen mit jeder Anlage. Deshalb weicht sie
	// jeder Nummer aus, die schon einmal einer Person gehörte (ersatzFrei).
	ersatz := "A-" + l.ID
	for n := 2; !p.ersatzFrei(ersatz); n++ {
		ersatz = fmt.Sprintf("A-%s-%d", l.ID, n)
	}
	grund := "Ausweisnummer bereits vergeben"
	switch {
	case nummer == "":
		grund = "keine Lesernummer im Altbestand"
	case p.buchBarcodes[nummer]:
		grund = "Ausweisnummer ist schon der Barcode eines Buchs"
	}
	p.s.prot.Warnung(l.ID, nummer, grund+" – Ausweis "+ersatz+" vergeben, Karte muss neu gedruckt werden")
	p.belegteAusweise[ersatz] = true
	return ersatz
}

// sqlVerbrauchteAusweise liest jede Nummer, die je an einer Leserzeile stand, und jede A-Nummer
// als Zahl ohne führende Nullen — aus leser und aus ausweisnummern_ausgeschieden. So zählt sie
// auch der Generator (substr(barcode_id, 3)::bigint): A-04908 steht hier als A-4908 und sperrt
// die Ersatznummer A-4908, die aus der Littera-Kennung 4908 entstünde.
const sqlVerbrauchteAusweise = `
	SELECT barcode_id FROM leser WHERE barcode_id IS NOT NULL
	UNION
	SELECT 'A-' || substr(barcode_id, 3)::bigint FROM leser WHERE barcode_id ~ '^A-[0-9]{1,15}$'
	UNION
	SELECT 'A-' || nummer FROM ausweisnummern_ausgeschieden`

// ersatzFrei sagt, ob eine selbst erfundene Nummer vergeben werden darf: Sie steht weder an
// einer aktiven Leserzeile noch an einem Buch, und sie stand nie an einer Person — auch nicht
// als dieselbe Zahl in anderer Schreibweise (A-04908 und A-4908). Deren Karte buchte sonst an
// der Theke auf die übernommene Person (docs/OFFEN.md 5.23).
func (p *personenlauf) ersatzFrei(nummer string) bool {
	return !p.belegteAusweise[nummer] && !p.buchBarcodes[nummer] && !p.verbrauchteAusweise[nummer]
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
