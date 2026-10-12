package smtptest

import (
	"testing"
	"time"
)

// WarteAufMail holt die abgefangene Nachricht. Die Frist macht aus einem ausbleibenden Versand
// einen Fehlschlag statt eines hängenden Tests.
func WarteAufMail(t *testing.T, sitzungen <-chan Sitzung) string {
	t.Helper()
	select {
	case s := <-sitzungen:
		return s.Nachricht
	case <-time.After(10 * time.Second):
		t.Fatal("es wurde gar keine Mail verschickt")
		return ""
	}
}

// Kopf kürzt eine Nachricht für eine Fehlermeldung: Die base64-kodierten Anhänge machten sie
// sonst unlesbar.
func Kopf(nachricht string) string {
	if len(nachricht) > 1200 {
		return nachricht[:1200] + "\n… (gekürzt)"
	}
	return nachricht
}
