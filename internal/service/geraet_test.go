package service

import (
	"context"
	"errors"
	"testing"

	"bibliothek/db"
	"bibliothek/repository"

	"github.com/pashagolub/pgxmock/v5"
)

func TestNewGeraeteDienst(t *testing.T) {
	var pool db.Pool = nil
	var studentRepo leserAbruf = nil
	var loanRepo repository.AusleihAbfragen = nil
	var auditRepo repository.ProtokollAbfragen = nil

	ds := NewGeraeteDienst(pool, studentRepo, loanRepo, auditRepo)
	if ds == nil {
		t.Fatal("erwartete GeraeteDienst-Instanz, bekam nil")
	}

	if ds.pool != pool {
		t.Errorf("erwartete pool=%v, bekam %v", pool, ds.pool)
	}
	if ds.leserAbfragen != studentRepo {
		t.Errorf("erwartete studentRepo=%v, bekam %v", studentRepo, ds.leserAbfragen)
	}
	if ds.ausleihAbfragen != loanRepo {
		t.Errorf("erwartete loanRepo=%v, bekam %v", loanRepo, ds.ausleihAbfragen)
	}
	if ds.protokollAbfragen != auditRepo {
		t.Errorf("erwartete auditRepo=%v, bekam %v", auditRepo, ds.protokollAbfragen)
	}
}

// gesperrterLeserAttrappe liefert einen festen Leser; ladeAkteur ruft nur Finde.
type gesperrterLeserAttrappe struct {
	leser *repository.Leser
}

func (s gesperrterLeserAttrappe) Finde(context.Context, string) (*repository.Leser, error) {
	return s.leser, nil
}

// TestGeraeteAusleiheRespektiertManuelleSperre: Ein von der Bibliothek von Hand gesperrter
// Schüler (is_manually_blocked, z. B. unbezahlte Schäden) bekommt kein Gerät, wie er kein Buch
// bekäme; ist_gesperrt allein genügt dem Geräte-Pfad nicht. Buch und Gerät prüfen über
// denselben Weg (pruefeAusleihSperren): Die Sperre am Leser hält auch mit override_block —
// aufgehoben wird sie in der Akte.
func TestGeraeteAusleiheRespektiertManuelleSperre(t *testing.T) {
	sid := "s1"
	// Nur manuell gesperrt → blockiert vor jeder Pool-Nutzung (Flag-Check zuerst).
	svc := &GeraeteDienst{leserAbfragen: gesperrterLeserAttrappe{
		leser: &repository.Leser{ID: sid, Art: "schueler", IstVonHandGesperrt: true}}}
	for _, uebergehen := range []bool{false, true} {
		_, _, err := svc.ladeAkteur(context.Background(), &sid, uebergehen)
		if !errors.Is(err, ErrGesperrt) || !IstSperreAmLeser(err) {
			t.Fatalf("override_block=%v: manuell gesperrter Schüler muss auch fürs Gerät blockiert sein, err=%v", uebergehen, err)
		}
	}
}

// TestGeraeteAusleiheRespektiertAutomatikSperren: Die Geräte-Ausleihe wendet dieselben
// automatischen Sperren an wie der Buch-Pfad (unbezahlte Schäden, Überfällig-Automatik), und
// sie lassen sich am Gerät übergehen wie am Buch.
func TestGeraeteAusleiheRespektiertAutomatikSperren(t *testing.T) {
	sid := "s1"
	neu := func(t *testing.T, vorbereiten func(pgxmock.PgxPoolIface)) (*GeraeteDienst, pgxmock.PgxPoolIface) {
		t.Helper()
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(mock.Close)
		vorbereiten(mock)
		return &GeraeteDienst{pool: mock, leserAbfragen: gesperrterLeserAttrappe{
			leser: &repository.Leser{ID: sid, Art: "schueler"}}}, mock
	}
	schaden := func(mock pgxmock.PgxPoolIface) {
		mock.ExpectQuery(`FROM schadensfaelle WHERE schueler_id`).
			WithArgs(sid).WillReturnRows(pgxmock.NewRows([]string{"count", "summe"}).AddRow(1, 12.0))
	}

	t.Run("unbezahlter Schaden blockiert das Gerät", func(t *testing.T) {
		svc, _ := neu(t, schaden)
		_, _, err := svc.ladeAkteur(context.Background(), &sid, false)
		if !errors.Is(err, ErrGesperrt) || !IstUebergehbareSperre(err) {
			t.Fatalf("Schüler mit unbezahltem Schaden: erwartet übergehbare Sperre, err=%v", err)
		}
	})

	t.Run("override_block übergeht den Schaden", func(t *testing.T) {
		svc, mock := neu(t, func(mock pgxmock.PgxPoolIface) {
			schaden(mock)
			mock.ExpectQuery(`FROM system_einstellungen`).
				WillReturnRows(pgxmock.NewRows([]string{"schluessel", "wert"})) // leer → Defaults
			mock.ExpectQuery(`FROM ausleihen`).
				WithArgs(sid, pgxmock.AnyArg()).WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(0))
		})
		_, lage, err := svc.ladeAkteur(context.Background(), &sid, true)
		if err != nil || len(lage.uebergangen) != 1 || lage.einst == nil {
			t.Fatalf("ladeAkteur(…) = Fehler %v, übergangen %v, erwartet die Ausleihe mit einem Protokolleintrag", err, lage.uebergangen)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Error(err)
		}
	})

	t.Run("ohne Sperren geht das Gerät durch", func(t *testing.T) {
		svc, mock := neu(t, func(mock pgxmock.PgxPoolIface) {
			mock.ExpectQuery(`FROM schadensfaelle WHERE schueler_id`).
				WithArgs(sid).WillReturnRows(pgxmock.NewRows([]string{"count", "summe"}).AddRow(0, 0.0))
			mock.ExpectQuery(`FROM system_einstellungen`).
				WillReturnRows(pgxmock.NewRows([]string{"schluessel", "wert"})) // leer → Defaults
			mock.ExpectQuery(`FROM ausleihen`).
				WithArgs(sid, pgxmock.AnyArg()).WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(0))
		})
		if _, _, err := svc.ladeAkteur(context.Background(), &sid, false); err != nil {
			t.Fatalf("ungesperrter Schüler ohne offene Vorgänge darf nicht blockiert werden: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Error(err)
		}
	})
}

// Ein Kollege wird nie gesperrt: keine Sperre, keine Zählung. Die
// Einstellungen liest ladeAkteur trotzdem — genau einmal, für die Frist (Sommerferien).
// Rot gesehen am Rückbau: ohne das Nachladen in ladeAkteur fehlen sie.
func TestGeraeteAusleiheKollegeOhneSperre(t *testing.T) {
	kid := "k1"
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()
	mock.ExpectQuery(`FROM system_einstellungen`).
		WillReturnRows(pgxmock.NewRows([]string{"schluessel", "wert"}).AddRow("sommerferien", strPtr("")))
	svc := &GeraeteDienst{pool: mock, leserAbfragen: gesperrterLeserAttrappe{
		leser: &repository.Leser{ID: kid, Art: "lehrkraft", IstVonHandGesperrt: true, Sperrgrund: strPtr("alt")}}}

	leser, lage, err := svc.ladeAkteur(context.Background(), &kid, false)
	if err != nil || leser == nil {
		t.Fatalf("Kollege abgewiesen: %v", err)
	}
	if lage.einst == nil {
		t.Error("ohne Einstellungen fehlt der Frist die Ferientabelle der Schule")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Error(err)
	}
}
