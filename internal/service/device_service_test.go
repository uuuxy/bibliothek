package service

import (
	"context"
	"errors"
	"testing"

	"bibliothek/db"
	"bibliothek/repository"

	"github.com/pashagolub/pgxmock/v5"
)

func TestNewDeviceService(t *testing.T) {
	var pool db.PgxPoolIface = nil
	var studentRepo repository.StudentRepository = nil
	var loanRepo repository.LoanRepository = nil
	var auditRepo repository.AuditRepository = nil

	service := NewDeviceService(pool, studentRepo, loanRepo, auditRepo)
	if service == nil {
		t.Fatal("erwartete DeviceService-Instanz, bekam nil")
	}

	ds, ok := service.(*defaultDeviceService)
	if !ok {
		t.Fatalf("erwartete Typ *defaultDeviceService, bekam %T", service)
	}

	if ds.pool != pool {
		t.Errorf("erwartete pool=%v, bekam %v", pool, ds.pool)
	}
	if ds.studentRepo != studentRepo {
		t.Errorf("erwartete studentRepo=%v, bekam %v", studentRepo, ds.studentRepo)
	}
	if ds.loanRepo != loanRepo {
		t.Errorf("erwartete loanRepo=%v, bekam %v", loanRepo, ds.loanRepo)
	}
	if ds.auditRepo != auditRepo {
		t.Errorf("erwartete auditRepo=%v, bekam %v", auditRepo, ds.auditRepo)
	}
}

// stubStudentRepoSperre liefert einen festen Leser — nur GetLeserByID wird von ladeAkteur
// aufgerufen, die übrigen Interface-Methoden bleiben ungenutzt (eingebettetes nil).
type stubStudentRepoSperre struct {
	repository.StudentRepository
	student *repository.Student
}

func (s stubStudentRepoSperre) GetLeserByID(context.Context, string) (*repository.Student, error) {
	return s.student, nil
}

// TestGeraeteAusleiheRespektiertManuelleSperre belegt die Lücke, die der Nebenläufigkeits-
// Audit nebenbei fand (19.08.2026): Der Geräte-Pfad prüfte nur ist_gesperrt. Ein von der
// Bibliothek MANUELL gesperrter Schüler (is_manually_blocked, z. B. unbezahlte Schäden)
// konnte trotzdem ein Gerät ausleihen — obwohl er kein Buch bekäme. Seit dem 24.09.2026
// prüfen Buch und Gerät über denselben Weg (pruefeAusleihSperren): Die Sperre am Leser hält
// auch mit override_block — aufgehoben wird sie in der Akte.
func TestGeraeteAusleiheRespektiertManuelleSperre(t *testing.T) {
	sid := "s1"
	// Nur manuell gesperrt → blockiert VOR jeder Pool-Nutzung (Flag-Check zuerst).
	svc := &defaultDeviceService{studentRepo: stubStudentRepoSperre{
		student: &repository.Student{ID: sid, Art: "schueler", IsManuallyBlocked: true}}}
	for _, uebergehen := range []bool{false, true} {
		_, _, err := svc.ladeAkteur(context.Background(), &sid, uebergehen)
		if !errors.Is(err, ErrBlocked) || !IstSperreAmLeser(err) {
			t.Fatalf("override_block=%v: manuell gesperrter Schüler muss auch fürs Gerät blockiert sein, err=%v", uebergehen, err)
		}
	}
}

// TestGeraeteAusleiheRespektiertAutomatikSperren belegt die Betreiber-Entscheidung
// (19.08.2026): Die Geräte-Ausleihe wendet dieselben AUTOMATIK-Sperren an wie der
// Buch-Pfad (unbezahlte Schäden, Überfällig-Automatik). Seit dem 24.09.2026 lassen sie sich
// am Gerät übergehen wie am Buch (docs/OFFEN.md 4.4).
func TestGeraeteAusleiheRespektiertAutomatikSperren(t *testing.T) {
	sid := "s1"
	neu := func(t *testing.T, vorbereiten func(pgxmock.PgxPoolIface)) (*defaultDeviceService, pgxmock.PgxPoolIface) {
		t.Helper()
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(mock.Close)
		vorbereiten(mock)
		return &defaultDeviceService{pool: mock, studentRepo: stubStudentRepoSperre{
			student: &repository.Student{ID: sid, Art: "schueler"}}}, mock
	}
	schaden := func(mock pgxmock.PgxPoolIface) {
		mock.ExpectQuery(`FROM schadensfaelle WHERE schueler_id`).
			WithArgs(sid).WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(1))
	}

	t.Run("unbezahlter Schaden blockiert das Gerät", func(t *testing.T) {
		svc, _ := neu(t, schaden)
		_, _, err := svc.ladeAkteur(context.Background(), &sid, false)
		if !errors.Is(err, ErrBlocked) || !IstUebergehbareSperre(err) {
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
			t.Fatalf("erwartet Ausleihe mit einem Protokolleintrag, war err=%v uebergangen=%v", err, lage.uebergangen)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Error(err)
		}
	})

	t.Run("ohne Sperren geht das Gerät durch", func(t *testing.T) {
		svc, mock := neu(t, func(mock pgxmock.PgxPoolIface) {
			mock.ExpectQuery(`FROM schadensfaelle WHERE schueler_id`).
				WithArgs(sid).WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(0))
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

// Ein Kollege wird nie gesperrt (16.09. und 24.09.2026): keine Sperre, keine Zählung. Die
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
		WillReturnRows(pgxmock.NewRows([]string{"schluessel", "wert"}).AddRow("sommerferien", ""))
	svc := &defaultDeviceService{pool: mock, studentRepo: stubStudentRepoSperre{
		student: &repository.Student{ID: kid, Art: "lehrkraft", IsManuallyBlocked: true, BlockReason: strPtr("alt")}}}

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
