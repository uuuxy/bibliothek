package api

import (
	"testing"

	"bibliothek/db"
	"bibliothek/mailservice"
)

// NewServer richtet den Versand über die gespeicherte Konfiguration ein. Fehlte er, lehnte der
// Betrieb jede Mail als „nicht konfiguriert" ab, und kein Test einer Tür fiele darüber: Die
// Tests geben ihren Türen einen eigenen Versand.
func TestNewServer_RichtetDenMailversandEin(t *testing.T) {
	srv := NewServer(&db.Verbindung{}, nil, nil, false)
	if _, ok := srv.Mail.(*mailservice.Postausgang); !ok {
		t.Errorf("NewServer(…).Mail = %T, erwartet *mailservice.Postausgang", srv.Mail)
	}
}
