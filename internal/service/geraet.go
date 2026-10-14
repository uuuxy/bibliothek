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

// GeraeteErgebnis beschreibt das Ergebnis einer Geräte-Aktion wie Ausleihe, Rückgabe oder Checklisten-Anforderung.
type GeraeteErgebnis struct {
	// Art definiert den Typ des Ergebnisses (z. B. "ausleihe", "rueckgabe", "geraet_check").
	Art string
	// Geraet enthält die Stammdaten des betroffenen Geräts.
	Geraet *repository.Geraet
	// Leser ist der Leser, für den die Aktion durchgeführt wurde.
	Leser *repository.Leser
	// Frist gibt die berechnete Rückgabefrist für das Gerät an (nur bei Ausleihe).
	Frist *time.Time
	// AusleiheID ist die ID des verknüpften Ausleihdatensatzes.
	AusleiheID *string
	// Fremdrueckgabe ist wahr, wenn das Gerät von einer anderen Person als dem Ausleiher zurückgegeben wurde.
	Fremdrueckgabe bool
	// Vorbesitzer ist der Leser, der das Gerät zuletzt ausgeliehen hatte (bei Fremdrückgabe).
	Vorbesitzer *repository.Leser
}

// GeraeteDienst definiert die Schnittstelle für alle Aktionen rund um Hardware-Geräte (z. B. Laptops, Tablets).
type GeraeteDienst interface {
	// VerarbeiteScan verarbeitet das Scannen eines Geräts. Je nach Zustand wird das Gerät
	// entweder ausgeliehen (falls frei) oder zurückgegeben (falls aktuell ausgeliehen).
	// Zudem wird geprüft, ob eine Zubehör-Checkliste vor der Ausleihe bestätigt werden muss.
	// overrideBlock übergeht die Hinweise (offene Forderung, Überfällig-Automatik), nicht
	// die Sperre am Leser — wie beim Buch (pruefeAusleihSperren).
	VerarbeiteScan(ctx context.Context, query string, aktiverLeserID *string, checklisteBestaetigt, uebergehen bool, bearbeiterID string) (*GeraeteErgebnis, error)
}

// geraeteDienst ist die Standard-Implementierung des GeraeteDienst.
type geraeteDienst struct {
	pool              db.PgxPoolIface
	leserAbfragen     repository.LeserAbfragen
	ausleihAbfragen   repository.AusleihAbfragen
	protokollAbfragen repository.ProtokollAbfragen
}

// NewGeraeteDienst erstellt eine neue Instanz des Standard-Geräteservice.
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

// ladeAkteur ermittelt den aktiven Leser und prüft die Sperren der Ausleihe auf demselben Weg
// wie beim Buch (pruefeAusleihSperren). Ein Gerät ist kein Lernmittel: Es gelten die Sperre
// am Leser und beide Hinweise. Die Rückgabe nimmt ladeRueckgeber, ein gesperrter Leser muss
// sein Gerät zurückgeben können. Die Lage geht mit zurück: leiheGeraetAus rechnet aus ihren
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

// ladeRueckgeber ermittelt den aktiven Leser aus dem gescannten Ausweis, ohne
// Sperrprüfung. Die Rückgabe braucht ihn nur, um eine Fremdrückgabe zu erkennen.
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

// leiheGeraetAus behandelt Fall A: das freie Gerät wird an den aktiven Leser ausgeliehen.
// geraeteLeihfristTage ist die Standard-Leihfrist für Hardware (2 Wochen).
const geraeteLeihfristTage = 14

// geraeteRueckgabeFrist ist die Geräte-Leihfrist als Tagesfrist wie die der Bücher
// (frist.go): Tagesende in der Schul-Zeitzone (Europe/Berlin), und fällt der Tag auf ein
// Wochenende, einen Feiertag oder in die Ferien, der nächste Schultag. Ohne die
// Normalisierung fiel die Frist auf die Sekunde genau N Tage später in der Server-Zeitzone
// (im Docker-Container UTC); ein um 10:00 MESZ geliehenes Gerät wäre 08:00 UTC fällig, was
// Mahnläufe und die "heute/morgen fällig"-Anzeige verschob.
func geraeteRueckgabeFrist(ab time.Time, kalender lmfplan.Ferientabelle) time.Time {
	return kalender.Tagesfrist(ab, geraeteLeihfristTage)
}

func (d *geraeteDienst) leiheGeraetAus(ctx context.Context, tx pgx.Tx, g *repository.Geraet, leser *repository.Leser, lage sperrLage, bearbeiterID string) (*GeraeteErgebnis, error) {
	if leser == nil {
		return nil, meldung(ErrUngueltigerZustand, "Bitte scannen Sie zuerst einen Ausweis")
	}

	// Standard-Hardware-Leihfrist beträgt 14 Tage (2 Wochen), auf das Tagesende in der
	// Schul-Zeitzone normalisiert (analog zu den Buch-Fristen).
	sommerferien := ""
	if lage.einst != nil {
		sommerferien = lage.einst.Sommerferien
	}
	frist := geraeteRueckgabeFrist(time.Now(), lmfplan.FerientabelleAus(sommerferien))
	erg := &GeraeteErgebnis{Leser: leser}
	// Ein Schreiber für jeden Leser. Nicht-Schüler bekommen das Gerät als Dauerleihe —
	// dieselbe Regel wie beim Buch (legeAusleiheAn).
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

	// Revisionssicheres Audit-Log schreiben.
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

// findeVorbesitzer stellt fest, ob eine Fremdrückgabe vorliegt (Rückgeber ≠ Ausleiher)
// und lädt in diesem Fall die Stammdaten des ursprünglichen Ausleihers in resp.
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

// gibGeraetZurueck behandelt Fall B: das ausgeliehene Gerät wird zurückgegeben.
func (d *geraeteDienst) gibGeraetZurueck(ctx context.Context, tx pgx.Tx, g *repository.Geraet, offeneAusleihe *repository.Ausleihe, leser *repository.Leser, bearbeiterID string) (*GeraeteErgebnis, error) {
	erg := &GeraeteErgebnis{}

	istFremd, err := d.findeVorbesitzer(ctx, offeneAusleihe, leser, erg)
	if err != nil {
		return nil, err
	}

	if err := repository.BucheGeraeteRueckgabe(ctx, tx, offeneAusleihe.ID, bearbeiterID, istFremd); err != nil {
		return nil, err
	}

	// Revisionssicheres Audit-Log für die Rückgabe schreiben.
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

// VerarbeiteScan implementiert die Geschäftslogik für die Ausleihe und Rückgabe von Geräten.
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

	// Transaktion starten, um den Ausleihprozess atomar und thread-sicher zu gestalten.
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

	// Checklisten-Regel: Hat das Gerät Zubehör und der Benutzer hat es noch nicht
	// bestätigt, unterbrechen wir und fordern die Bestätigung an. Das Übergehen einer Sperre
	// geht hier nicht verloren: Die Theke schickt override_block mit der Bestätigung erneut
	// (OmniboxChecklistDialog.svelte), und protokolliert wird erst in leiheGeraetAus.
	if g.Zubehoer != "" && !checklisteBestaetigt {
		return &GeraeteErgebnis{Art: "geraet_check", Geraet: &g}, nil
	}

	if !istVerliehen {
		return d.leiheGeraetAus(ctx, tx, &g, leser, lage, bearbeiterID)
	}
	return d.gibGeraetZurueck(ctx, tx, &g, &offeneAusleihe, leser, bearbeiterID)
}
