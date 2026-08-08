package service

import (
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
