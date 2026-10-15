package auth

import (
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v5"
)

// Ohne jti wären zwei Anmeldungen desselben Kontos in derselben Sekunde Byte für Byte gleich,
// und die Abmeldung der einen widerriefe über den Hash in der Sperrliste auch die andere.
func TestErzeugeToken_JedesTokenIstEinmalig(t *testing.T) {
	a, _ := newTestAnmeldung(t, 12*time.Hour)

	t1, err := a.ErzeugeToken("user-1", "B-1", RolleAdmin, "")
	if err != nil {
		t.Fatalf("ErzeugeToken 1: %v", err)
	}
	t2, err := a.ErzeugeToken("user-1", "B-1", RolleAdmin, "")
	if err != nil {
		t.Fatalf("ErzeugeToken 2: %v", err)
	}

	if t1 == t2 {
		t.Fatalf("zwei Tokens mit identischen Claims dürfen nie byte-identisch sein (jti fehlt?)")
	}
}

// Ein gültig signiertes Token, das nicht auf der Sperrliste steht, wird trotzdem abgelehnt,
// sobald das Konto in der Datenbank deaktiviert (aktiv=false) oder gelöscht ist (keine Zeile).
// Ohne die Prüfung behielte, wer gehen musste, bis zum Ablauf des Tokens (zwölf Stunden) vollen
// Zugriff.
func TestPruefeToken_WiderrufWirktSofort(t *testing.T) {
	newToken := func(t *testing.T, a *Anmeldung) string {
		t.Helper()
		tok, err := a.ErzeugeToken("user-1", "B-1", RolleAdmin, "")
		if err != nil {
			t.Fatalf("ErzeugeToken: %v", err)
		}
		return tok
	}

	t.Run("aktives Konto wird akzeptiert", func(t *testing.T) {
		a, mock := newTestAnmeldung(t, 12*time.Hour)
		token := newToken(t, a)
		erwarteNichtWiderrufen(mock)
		erwarteKontoAktiv(mock, true)

		if _, err := a.PruefeToken(token); err != nil {
			t.Errorf("aktives Konto: unerwarteter Fehler: %v", err)
		}
	})

	t.Run("deaktiviertes Konto wird abgelehnt", func(t *testing.T) {
		a, mock := newTestAnmeldung(t, 12*time.Hour)
		token := newToken(t, a)
		erwarteNichtWiderrufen(mock)
		erwarteKontoAktiv(mock, false)

		if _, err := a.PruefeToken(token); err == nil {
			t.Error("deaktiviertes Konto muss abgelehnt werden")
		}
	})

	t.Run("geloeschtes Konto (keine Zeile) wird abgelehnt", func(t *testing.T) {
		a, mock := newTestAnmeldung(t, 12*time.Hour)
		token := newToken(t, a)
		erwarteNichtWiderrufen(mock)
		mock.ExpectQuery(`SELECT aktiv, rolle FROM benutzer`).
			WithArgs(pgxmock.AnyArg()).
			WillReturnError(pgx.ErrNoRows)

		if _, err := a.PruefeToken(token); err == nil {
			t.Error("gelöschtes Konto muss abgelehnt werden")
		}
	})
}
