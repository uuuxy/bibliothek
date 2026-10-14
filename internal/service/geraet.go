package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"bibliothek/db"
	"bibliothek/pkg/leserart"
	"bibliothek/pkg/lmfplan"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// GeraeteErgebnis ist das Ergebnis eines Scans an einem Gerät: Ausleihe, Rückgabe oder die
// Bitte, die Checkliste des Zubehörs zu bestätigen.
type GeraeteErgebnis struct {
	// Art ist "ausleihe", "rueckgabe" oder "geraet_check", wenn erst die Checkliste bestätigt
	// werden muss.
	Art string
	// Geraet ist das gescannte Gerät.
	Geraet *repository.Geraet
	// Leser ist der Leser, an den das Gerät ausgeliehen wurde.
	Leser *repository.Leser
	// Frist ist die Rückgabefrist einer Ausleihe.
	Frist *time.Time
	// AusleiheID ist die Kennung der gebuchten Ausleihe.
	AusleiheID *string
	// Fremdrueckgabe sagt, ob jemand anderes als der Ausleiher das Gerät zurückgebracht hat.
	Fremdrueckgabe bool
	// Vorbesitzer ist bei einer Fremdrückgabe der Leser, der das Gerät ausgeliehen hatte.
	Vorbesitzer *repository.Leser
}

// GeraeteDienst bucht den Scan eines Geräts (Laptop, Tablet).
type GeraeteDienst interface {
	// VerarbeiteScan leiht ein freies Gerät dem geöffneten Leser aus und nimmt ein verliehenes
	// zurück. Hat das Gerät Zubehör, muss vor der Ausleihe die Checkliste bestätigt sein.
	// uebergehen übergeht die Hinweise (offene Forderung, Überfällig-Automatik), nicht die
	// Sperre am Leser, wie beim Buch (pruefeAusleihSperren).
	VerarbeiteScan(ctx context.Context, nummer string, aktiverLeserID *string, checklisteBestaetigt, uebergehen bool, bearbeiterID string) (*GeraeteErgebnis, error)
}

// geraeteDienst erfüllt GeraeteDienst.
type geraeteDienst struct {
	pool              db.PgxPoolIface
	leserAbfragen     repository.LeserAbfragen
	ausleihAbfragen   repository.AusleihAbfragen
	protokollAbfragen repository.ProtokollAbfragen
}

// NewGeraeteDienst baut den Dienst für Geräte.
func NewGeraeteDienst(pool db.PgxPoolIface, leserAbfragen repository.LeserAbfragen, ausleihAbfragen repository.AusleihAbfragen, protokollAbfragen repository.ProtokollAbfragen) GeraeteDienst {
	return &geraeteDienst{
		pool:              pool,
		leserAbfragen:     leserAbfragen,
		ausleihAbfragen:   ausleihAbfragen,
		protokollAbfragen: protokollAbfragen,
	}
}

// ladeGeraet lädt das Gerät zu seiner Nummer. Ob es verliehen werden darf, prüft erst der Zweig
// der Ausleihe (pruefeGeraetAusleihbar): Ein verliehenes Gerät, das danach als defekt gemeldet
// wurde, muss sich zurückgeben lassen (geraet_rueckgabe_sperre_pg_test.go).
func (d *geraeteDienst) ladeGeraet(ctx context.Context, nummer string) (repository.Geraet, error) {
	g, err := repository.LadeGeraetZuNummer(ctx, d.pool, nummer)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return g, meldung(ErrNichtGefunden, "Gerät mit Barcode %s nicht gefunden", nummer)
		}
		return g, err
	}
	return g, nil
}

// pruefeGeraetAusleihbar hält ausgesonderte und gesperrte Geräte (z. B. defekt) von der
// Ausleihe fern. Die Rückgabe prüft sie nicht: Eine offene Ausleihe zu schließen ist immer
// richtig, auch für ein Gerät, das inzwischen defekt oder ausgesondert ist.
func pruefeGeraetAusleihbar(g repository.Geraet) error {
	if g.IstAusgesondert {
		return meldung(ErrUngueltigerZustand, "Gerät ist ausgesondert")
	}
	if !g.IstAusleihbar {
		return fmt.Errorf("%w: Gerät ist aktuell gesperrt", ErrGesperrt)
	}
	return nil
}

// ladeAkteur lädt den geöffneten Leser und prüft die Sperren der Ausleihe auf demselben Weg wie
// beim Buch (pruefeAusleihSperren). Ein Gerät ist kein Lernmittel: Es gelten die Sperre am
// Leser und beide Hinweise. Die Rückgabe nimmt ladeRueckgeber, weil ein gesperrter Leser sein
// Gerät zurückgeben können muss. Die Lage geht mit zurück: leiheGeraetAus rechnet aus ihren
// Einstellungen die Frist und protokolliert, was übergangen wurde.
func (d *geraeteDienst) ladeAkteur(ctx context.Context, aktiverLeserID *string, uebergehen bool) (*repository.Leser, sperrLage, error) {
	leser, err := d.ladeRueckgeber(ctx, aktiverLeserID)
	if err != nil || leser == nil {
		return nil, sperrLage{}, err
	}
	lage, err := pruefeAusleihSperren(ctx, d.pool, leser, false, uebergehen)
	if err != nil {
		return nil, sperrLage{}, err
	}
	if lage.einst == nil {
		// Die Prüfung liest die Einstellungen nur für die Überfällig-Automatik, und die läuft
		// bei einem Kollegen nie. Die Frist braucht sie trotzdem: die Sommerferien der Schule.
		if lage.einst, err = ladeAusleihEinstellungen(ctx, d.pool); err != nil {
			return nil, sperrLage{}, err
		}
	}
	return leser, lage, nil
}

// ladeRueckgeber lädt den geöffneten Leser ohne Prüfung der Sperren. Die Rückgabe braucht ihn
// nur, um eine Fremdrückgabe zu erkennen.
func (d *geraeteDienst) ladeRueckgeber(ctx context.Context, aktiverLeserID *string) (*repository.Leser, error) {
	if aktiverLeserID == nil || *aktiverLeserID == "" {
		return nil, nil
	}
	leser, err := d.leserAbfragen.Finde(ctx, *aktiverLeserID)
	if err != nil {
		return nil, err
	}
	if leser == nil {
		return nil, meldung(ErrNichtGefunden, "Aktiver Leser nicht gefunden")
	}
	return leser, nil
}

// geraeteLeihfristTage ist die Leihfrist eines Geräts in Tagen.
const geraeteLeihfristTage = 14

// geraeteRueckgabeFrist ist die Leihfrist eines Geräts als Tagesfrist wie die der Bücher
// (frist.go): Tagesende in der Zone der Schule, und fällt der Tag auf ein Wochenende, einen
// Feiertag oder in die Ferien, der nächste Schultag. Ohne das Tagesende fiele die Frist auf die
// Sekunde genau 14 Tage später in der Zone des Servers, und Mahnlauf und die Anzeige „heute
// fällig" lägen daneben.
func geraeteRueckgabeFrist(ab time.Time, kalender lmfplan.Ferientabelle) time.Time {
	return kalender.Tagesfrist(ab, geraeteLeihfristTage)
}

// leiheGeraetAus leiht das freie Gerät dem geöffneten Leser aus.
func (d *geraeteDienst) leiheGeraetAus(ctx context.Context, tx pgx.Tx, g *repository.Geraet, leser *repository.Leser, lage sperrLage, bearbeiterID string) (*GeraeteErgebnis, error) {
	if leser == nil {
		return nil, meldung(ErrUngueltigerZustand, "Bitte scannen Sie zuerst einen Ausweis")
	}

	sommerferien := ""
	if lage.einst != nil {
		sommerferien = lage.einst.Sommerferien
	}
	frist := geraeteRueckgabeFrist(time.Now(), lmfplan.FerientabelleAus(sommerferien))
	erg := &GeraeteErgebnis{Leser: leser}
	// Wer kein Schüler ist, bekommt das Gerät als Dauerleihe, dieselbe Regel wie beim Buch
	// (legeAusleiheAn).
	ausleiheID, err := repository.LeiheGeraetAus(ctx, tx, repository.GeraeteAusleihe{
		GeraetID:       g.ID,
		LeserID:        leser.ID,
		BearbeiterID:   bearbeiterID,
		RueckgabeFrist: frist,
		IstDauerleihe:  !leserart.IstSchueler(leser.Art),
	})
	if err != nil {
		// Zwei Plätze leihen dasselbe Gerät zugleich aus: Der zweite verletzt
		// uniq_ausleihen_aktiv_geraet (23505). Es entsteht nur eine Ausleihe; ohne die
		// Übersetzung bekäme der zweite Platz statt des Konflikts einen Serverfehler.
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, meldung(ErrKonflikt, "Gerät ist bereits ausgeliehen (woanders verbucht)")
		}
		return nil, err
	}

	// Der Eintrag ins Protokoll steht in der Transaktion der Ausleihe.
	if err := d.protokollAbfragen.ProtokolliereAusleihe(ctx, tx, g.ID, leser.ID, "", bearbeiterID); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	protokolliereUebergangen(ctx, d.protokollAbfragen, bearbeiterID, leser.ID, lage.uebergangen)

	erg.Art = "ausleihe"
	erg.Geraet = g
	erg.Frist = &frist
	erg.AusleiheID = &ausleiheID
	return erg, nil
}

// findeVorbesitzer sagt, ob es eine Fremdrückgabe ist (der Rückgeber ist nicht der Ausleiher),
// und legt dann den Ausleiher als Vorbesitzer in erg.
func (d *geraeteDienst) findeVorbesitzer(ctx context.Context, offeneAusleihe *repository.Ausleihe, leser *repository.Leser, erg *GeraeteErgebnis) (bool, error) {
	if offeneAusleihe.SchuelerID == nil {
		return false, nil
	}
	if leser != nil && *offeneAusleihe.SchuelerID == leser.ID {
		return false, nil
	}
	// Ein gelöschter Vorbesitzer bleibt ohne Namen; eine Fremdrückgabe ist es trotzdem.
	vorbesitzer, err := repository.LadeLeserNameUndKlasse(ctx, d.pool, *offeneAusleihe.SchuelerID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return false, err
	}
	erg.Vorbesitzer = &vorbesitzer
	return true, nil
}

// gibGeraetZurueck nimmt das verliehene Gerät zurück.
func (d *geraeteDienst) gibGeraetZurueck(ctx context.Context, tx pgx.Tx, g *repository.Geraet, offeneAusleihe *repository.Ausleihe, leser *repository.Leser, bearbeiterID string) (*GeraeteErgebnis, error) {
	erg := &GeraeteErgebnis{}

	istFremd, err := d.findeVorbesitzer(ctx, offeneAusleihe, leser, erg)
	if err != nil {
		return nil, err
	}

	if err := repository.BucheGeraeteRueckgabe(ctx, tx, offeneAusleihe.ID, bearbeiterID, istFremd); err != nil {
		return nil, err
	}

	// Der Eintrag ins Protokoll steht in der Transaktion der Rückgabe.
	if offeneAusleihe.SchuelerID != nil {
		if err := d.protokollAbfragen.ProtokolliereRueckgabe(ctx, tx, g.ID, *offeneAusleihe.SchuelerID, "", bearbeiterID); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	erg.Art = "rueckgabe"
	erg.Geraet = g
	erg.AusleiheID = &offeneAusleihe.ID
	erg.Fremdrueckgabe = istFremd

	return erg, nil
}

// VerarbeiteScan bucht den Scan eines Geräts.
func (d *geraeteDienst) VerarbeiteScan(
	ctx context.Context,
	nummer string,
	aktiverLeserID *string,
	checklisteBestaetigt bool,
	uebergehen bool,
	bearbeiterID string,
) (*GeraeteErgebnis, error) {
	g, err := d.ladeGeraet(ctx, nummer)
	if err != nil {
		return nil, err
	}

	tx, err := d.ausleihAbfragen.BeginneTx(ctx)
	if err != nil {
		return nil, err
	}
	defer db.SafeRollback(ctx, tx)

	offeneAusleihe, istVerliehen, err := repository.SperreOffeneGeraeteAusleihe(ctx, tx, g.ID)
	if err != nil {
		return nil, err
	}

	// Die Sperren (Gerät defekt oder ausgesondert, Leser gesperrt) gelten der Ausleihe, nicht
	// der Rückgabe: Eine offene Ausleihe muss sich schließen lassen
	// (geraet_rueckgabe_sperre_pg_test.go).
	var leser *repository.Leser
	var lage sperrLage
	if istVerliehen {
		leser, err = d.ladeRueckgeber(ctx, aktiverLeserID)
	} else {
		if err = pruefeGeraetAusleihbar(g); err != nil {
			return nil, err
		}
		leser, lage, err = d.ladeAkteur(ctx, aktiverLeserID, uebergehen)
	}
	if err != nil {
		return nil, err
	}

	// Hat das Gerät Zubehör und ist die Checkliste noch nicht bestätigt, endet der Scan mit der
	// Bitte darum. Das Übergehen einer Sperre geht dabei nicht verloren: Die Theke schickt
	// override_block mit der Bestätigung erneut (OmniboxChecklistDialog.svelte), und
	// protokolliert wird erst in leiheGeraetAus.
	if g.Zubehoer != "" && !checklisteBestaetigt {
		return &GeraeteErgebnis{Art: "geraet_check", Geraet: &g}, nil
	}

	if !istVerliehen {
		return d.leiheGeraetAus(ctx, tx, &g, leser, lage, bearbeiterID)
	}
	return d.gibGeraetZurueck(ctx, tx, &g, &offeneAusleihe, leser, bearbeiterID)
}
