package service

import (
	"context"
	"errors"
	"testing"

	"bibliothek/db"
	"bibliothek/repository"
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

// stubStudentRepoSperre liefert einen festen Schüler — nur GetByID wird von ladeAkteur
// aufgerufen, die übrigen Interface-Methoden bleiben ungenutzt (eingebettetes nil).
type stubStudentRepoSperre struct {
	repository.StudentRepository
	student *repository.Student
}

func (s stubStudentRepoSperre) GetByID(context.Context, string) (*repository.Student, error) {
	return s.student, nil
}

// TestGeraeteAusleiheRespektiertManuelleSperre belegt die Lücke, die der Nebenläufigkeits-
// Audit nebenbei fand (19.08.2026): Der Geräte-Pfad prüfte nur ist_gesperrt. Ein von der
// Bibliothek MANUELL gesperrter Schüler (is_manually_blocked, z. B. unbezahlte Schäden)
// konnte trotzdem ein Gerät ausleihen — obwohl er kein Buch bekäme. Jetzt blockieren
// BEIDE Flags, wie im Buch-Pfad.
func TestGeraeteAusleiheRespektiertManuelleSperre(t *testing.T) {
	sid := "s1"

	// Nur manuell gesperrt → muss auch fürs Gerät blockieren.
	svc := &defaultDeviceService{studentRepo: stubStudentRepoSperre{
		student: &repository.Student{ID: sid, IstGesperrt: false, IsManuallyBlocked: true}}}
	if _, _, err := svc.ladeAkteur(context.Background(), &sid, nil); !errors.Is(err, ErrBlocked) {
		t.Fatalf("manuell gesperrter Schüler muss auch fürs Gerät blockiert sein, err=%v", err)
	}

	// Gegenprobe: gar nicht gesperrt → geht durch.
	svcOK := &defaultDeviceService{studentRepo: stubStudentRepoSperre{
		student: &repository.Student{ID: sid}}}
	if _, _, err := svcOK.ladeAkteur(context.Background(), &sid, nil); err != nil {
		t.Fatalf("ungesperrter Schüler darf nicht blockiert werden: %v", err)
	}
}
