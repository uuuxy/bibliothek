package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"bibliothek/db"
	"bibliothek/repository"

	"github.com/google/go-cmp/cmp"
)

// festerLeserAttrappe liefert einen festen Leser für die Ausweis-Stufe der Theke; Finde bleibt
// ungenutzt (eingebettetes nil).
type festerLeserAttrappe struct {
	leserErmittlung
	leser *repository.Leser
	err   error
}

func (m *festerLeserAttrappe) FindeZuBarcode(ctx context.Context, barcode string) (*repository.Leser, error) {
	return m.leser, m.err
}

// TestVerarbeiteAusweis: Ein gescannter Ausweis liefert einen Leser —
// Schüler wie Kollegium, unterscheidbar an der Art. Eine unbekannte Nummer bleibt ein
// lauter Fehler und verschwindet nicht in der Volltextsuche; ein Datenbankfehler kommt als
// solcher durch, statt sich als „nicht registriert" zu tarnen.
func TestVerarbeiteAusweis(t *testing.T) {
	tests := []struct {
		name        string
		leser       *repository.Leser
		err         error
		expectError bool
	}{
		{
			name:  "Kollege",
			leser: &repository.Leser{ID: "leser-123", Art: "lehrkraft"},
		},
		{
			name:  "Schüler",
			leser: &repository.Leser{ID: "leser-456", Art: "schueler"},
		},
		{
			name:        "Not Found",
			expectError: true,
		},
		{
			name:        "Database Error",
			err:         errors.New("db connection failed"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &OmniboxDienst{
				leserAbfragen: &festerLeserAttrappe{leser: tt.leser, err: tt.err},
			}

			resp := &OmniboxErgebnis{}
			err := svc.verarbeiteAusweis(context.Background(), "L-123", resp)

			if tt.expectError {
				if err == nil {
					t.Fatalf("verarbeiteAusweis(…) = nil, erwartet einen Fehler")
				}
				return
			}
			if err != nil {
				t.Fatalf("verarbeiteAusweis(…) = %v, erwartet nil", err)
			}
			if resp.Art != "student" {
				t.Errorf("verarbeiteAusweis(…): Art = %q, erwartet student", resp.Art)
			}
			if resp.Leser != tt.leser {
				t.Errorf("verarbeiteAusweis(…): Leser = %v, erwartet %v", resp.Leser, tt.leser)
			}
		})
	}
}

func TestNewOmniboxDienst(t *testing.T) {
	// Nullwerte genügen: Geprüft wird nur, dass der Erzeuger jede Abhängigkeit in ihr Feld
	// legt.
	var pool db.Pool
	var studentRepo leserErmittlung
	var bookRepo repository.KatalogAbfragen
	var loanRepo repository.AusleihAbfragen
	var loanSvc AusleihDienst
	var deviceSvc *GeraeteDienst

	svc := NewOmniboxDienst(
		pool,
		studentRepo,
		bookRepo,
		loanRepo,
		loanSvc,
		deviceSvc,
	)

	if svc == nil {
		t.Fatalf("NewOmniboxDienst(…) = nil")
	}

	if svc.pool != pool {
		t.Errorf("NewOmniboxDienst(…).pool ist nicht der übergebene Pool")
	}
	if svc.leserAbfragen != studentRepo {
		t.Errorf("NewOmniboxDienst(…).leserAbfragen sind nicht die übergebenen")
	}
	if svc.katalogAbfragen != bookRepo {
		t.Errorf("NewOmniboxDienst(…).katalogAbfragen sind nicht die übergebenen")
	}
	if svc.ausleihAbfragen != loanRepo {
		t.Errorf("NewOmniboxDienst(…).ausleihAbfragen sind nicht die übergebenen")
	}
	if svc.ausleihe != loanSvc {
		t.Errorf("NewOmniboxDienst(…).ausleihe ist nicht der übergebene Dienst")
	}
	if svc.geraete != deviceSvc {
		t.Errorf("NewOmniboxDienst(…).geraete ist nicht der übergebene Dienst")
	}
}

// katalogAttrappe erfüllt von repository.KatalogAbfragen nur FindeExemplar.
type katalogAttrappe struct {
	repository.KatalogAbfragen
	findeExemplar func(ctx context.Context, barcode string) (*repository.Exemplar, error)
}

func (m *katalogAttrappe) FindeExemplar(ctx context.Context, barcode string) (*repository.Exemplar, error) {
	if m.findeExemplar != nil {
		return m.findeExemplar(ctx, barcode)
	}
	return nil, nil
}

// ausleihDienstAttrappe erfüllt AusleihDienst mit Funktionen, die jeder Test selbst setzt.
type ausleihDienstAttrappe struct {
	AusleihDienst
	bucheAmLeser func(ctx context.Context, exemplar *repository.Exemplar, activeLeserID *string, staffID string, overrideBlock bool) (*AusleihErgebnis, error)
	nimmZurueck  func(ctx context.Context, exemplar *repository.Exemplar, staffID string) (*AusleihErgebnis, error)
}

func (m *ausleihDienstAttrappe) BucheAmLeser(ctx context.Context, exemplar *repository.Exemplar, activeLeserID *string, staffID string, overrideBlock bool) (*AusleihErgebnis, error) {
	if m.bucheAmLeser != nil {
		return m.bucheAmLeser(ctx, exemplar, activeLeserID, staffID, overrideBlock)
	}
	return nil, nil
}

func (m *ausleihDienstAttrappe) NimmZurueck(ctx context.Context, exemplar *repository.Exemplar, staffID string) (*AusleihErgebnis, error) {
	if m.nimmZurueck != nil {
		return m.nimmZurueck(ctx, exemplar, staffID)
	}
	return nil, nil
}

func TestVerarbeiteExemplar(t *testing.T) {
	ctx := context.Background()

	t.Run("FindeExemplar returns error", func(t *testing.T) {
		expectedErr := errors.New("db error")
		bookRepo := &katalogAttrappe{
			findeExemplar: func(ctx context.Context, barcode string) (*repository.Exemplar, error) {
				return nil, expectedErr
			},
		}

		svc := &OmniboxDienst{
			katalogAbfragen: bookRepo,
		}

		q := OmniboxEingabe{Text: "123"}
		resp := &OmniboxErgebnis{}
		err := svc.verarbeiteExemplar(ctx, q, resp)

		if err != expectedErr {
			t.Errorf("verarbeiteExemplar(…) = %v, erwartet %v", err, expectedErr)
		}
	})

	t.Run("FindeExemplar returns nil (not found)", func(t *testing.T) {
		bookRepo := &katalogAttrappe{
			findeExemplar: func(ctx context.Context, barcode string) (*repository.Exemplar, error) {
				return nil, nil
			},
		}

		svc := &OmniboxDienst{
			katalogAbfragen: bookRepo,
		}

		q := OmniboxEingabe{Text: "123"}
		resp := &OmniboxErgebnis{}
		err := svc.verarbeiteExemplar(ctx, q, resp)

		if err == nil {
			t.Errorf("verarbeiteExemplar(…) ohne Exemplar = nil, erwartet einen Fehler")
		} else if !errors.Is(err, ErrNichtGefunden) {
			t.Errorf("svc.verarbeiteExemplar(ctx, q, resp) = %v, erwartet ErrNichtGefunden", err)
		}
	})

	t.Run("Successful Checkout with active student", func(t *testing.T) {
		studentID := "student-123"
		bookCopy := &repository.Exemplar{ID: "copy-123", IstAusleihbar: true, IstAusgesondert: false}
		expectedLoanResult := &AusleihErgebnis{Art: "ausleihe"}

		bookRepo := &katalogAttrappe{
			findeExemplar: func(ctx context.Context, barcode string) (*repository.Exemplar, error) {
				return bookCopy, nil
			},
		}

		loanSvc := &ausleihDienstAttrappe{
			bucheAmLeser: func(ctx context.Context, exemplar *repository.Exemplar, activeLeserID *string, staffID string, overrideBlock bool) (*AusleihErgebnis, error) {
				if exemplar != bookCopy || activeLeserID == nil || *activeLeserID != studentID {
					t.Errorf("BucheAmLeser called with wrong arguments")
				}
				return expectedLoanResult, nil
			},
		}

		svc := &OmniboxDienst{
			katalogAbfragen: bookRepo,
			ausleihe:        loanSvc,
		}

		q := OmniboxEingabe{Text: "123", AktiverLeserID: &studentID}
		resp := &OmniboxErgebnis{}
		err := svc.verarbeiteExemplar(ctx, q, resp)

		if err != nil {
			t.Fatalf("verarbeiteExemplar(…) = %v, erwartet nil", err)
		}
		if resp.Art != "ausleihe" {
			t.Errorf("verarbeiteExemplar(…): Art = %s, erwartet ausleihe", resp.Art)
		}
	})

	t.Run("Successful Return with no active borrower", func(t *testing.T) {
		bookCopy := &repository.Exemplar{ID: "copy-123", IstAusleihbar: true, IstAusgesondert: false}
		expectedLoanResult := &AusleihErgebnis{Art: "rueckgabe"}

		bookRepo := &katalogAttrappe{
			findeExemplar: func(ctx context.Context, barcode string) (*repository.Exemplar, error) {
				return bookCopy, nil
			},
		}

		loanSvc := &ausleihDienstAttrappe{
			nimmZurueck: func(ctx context.Context, exemplar *repository.Exemplar, staffID string) (*AusleihErgebnis, error) {
				if exemplar != bookCopy {
					t.Errorf("NimmZurueck called with wrong arguments")
				}
				return expectedLoanResult, nil
			},
		}

		svc := &OmniboxDienst{
			katalogAbfragen: bookRepo,
			ausleihe:        loanSvc,
		}

		q := OmniboxEingabe{Text: "123"}
		resp := &OmniboxErgebnis{}
		err := svc.verarbeiteExemplar(ctx, q, resp)

		if err != nil {
			t.Fatalf("verarbeiteExemplar(…) = %v, erwartet nil", err)
		}
		if resp.Art != "rueckgabe" {
			t.Errorf("verarbeiteExemplar(…): Art = %s, erwartet rueckgabe", resp.Art)
		}
	})
}

type sucheKatalogAttrappe struct {
	repository.KatalogAbfragen
	titel []repository.Titel
	err   error
}

func (r *sucheKatalogAttrappe) FindeTitel(_ context.Context, _ string) ([]repository.Titel, error) {
	return r.titel, r.err
}

func TestVerarbeiteSuche(t *testing.T) {
	tests := []struct {
		name          string
		query         string
		mockTitles    []repository.Titel
		mockErr       error
		expectedType  string
		expectedError bool
	}{
		{
			name:          "Success - returns titles",
			query:         "hobbit",
			mockTitles:    []repository.Titel{{Titel: "The Hobbit"}},
			mockErr:       nil,
			expectedType:  "search_results",
			expectedError: false,
		},
		{
			name:          "Success - returns empty list",
			query:         "unknown",
			mockTitles:    []repository.Titel{},
			mockErr:       nil,
			expectedType:  "search_results",
			expectedError: false,
		},
		{
			name:          "Error - repository error",
			query:         "error",
			mockTitles:    nil,
			mockErr:       errors.New("db error"),
			expectedType:  "",
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &sucheKatalogAttrappe{
				titel: tt.mockTitles,
				err:   tt.mockErr,
			}
			svc := &OmniboxDienst{
				katalogAbfragen: repo,
			}

			resp := &OmniboxErgebnis{}
			err := svc.verarbeiteSuche(context.Background(), tt.query, resp)

			if tt.expectedError {
				if err == nil {
					t.Errorf("verarbeiteSuche(…) = nil, erwartet einen Fehler")
				}
				if err != nil && err.Error() != tt.mockErr.Error() {
					t.Errorf("verarbeiteSuche(…) = %v, erwartet %v", err, tt.mockErr)
				}
			} else {
				if err != nil {
					t.Errorf("verarbeiteSuche(…) = %v, erwartet nil", err)
				}
				if resp.Art != tt.expectedType {
					t.Errorf("verarbeiteSuche(…): Art = %q, erwartet %q", resp.Art, tt.expectedType)
				}
				if len(resp.Treffer) != len(tt.mockTitles) {
					t.Errorf("verarbeiteSuche(…): %d Treffer, erwartet %d", len(resp.Treffer), len(tt.mockTitles))
				}
			}
		})
	}
}

// TestVerarbeiteEingabe_ZweiterVersuchOhnePruefzeichen: Bleibt ein Scan ohne Treffer und endet er
// auf ein gültiges Prüfzeichen, fragt die Theke ein zweites Mal ohne dieses Zeichen
// (TestAlterAufdruckMitPruefzeichenWirdGelesen). Zwei Zusagen daran: Bleibt auch der zweite
// Versuch leer, nennt die Meldung die Nummer, die gescannt wurde. Und ein Fehler, der kein
// „nicht gefunden" ist, löst keinen zweiten Versuch aus.
func TestVerarbeiteEingabe_ZweiterVersuchOhnePruefzeichen(t *testing.T) {
	// „B-100016" ist „B-10001" mit dem Prüfzeichen 6.
	const gescannt, gekuerzt = "B-100016", "B-10001"

	t.Run("beide Versuche leer: die Meldung nennt die gescannte Nummer", func(t *testing.T) {
		var gefragt []string
		svc := &OmniboxDienst{katalogAbfragen: &katalogAttrappe{
			findeExemplar: func(_ context.Context, barcode string) (*repository.Exemplar, error) {
				gefragt = append(gefragt, barcode)
				return nil, nil
			},
		}}

		_, err := svc.VerarbeiteEingabe(context.Background(), OmniboxEingabe{Text: gescannt})
		if !errors.Is(err, ErrNichtGefunden) {
			t.Fatalf("Fehler %v, erwartet ErrNichtGefunden", err)
		}
		if !strings.Contains(err.Error(), gescannt) {
			t.Errorf("Meldung %q nennt nicht die gescannte Nummer %s", err.Error(), gescannt)
		}
		if soll := []string{gescannt, gekuerzt}; !cmp.Equal(gefragt, soll) {
			t.Errorf("gefragt wurde nach %v, erwartet %v", gefragt, soll)
		}
	})

	t.Run("ein Fehler der Datenbank löst keinen zweiten Versuch aus", func(t *testing.T) {
		dbFehler := errors.New("verbindung weg")
		abfragen := 0
		svc := &OmniboxDienst{katalogAbfragen: &katalogAttrappe{
			findeExemplar: func(context.Context, string) (*repository.Exemplar, error) {
				abfragen++
				return nil, dbFehler
			},
		}}

		_, err := svc.VerarbeiteEingabe(context.Background(), OmniboxEingabe{Text: gescannt})
		if !errors.Is(err, dbFehler) {
			t.Fatalf("Fehler %v, erwartet %v", err, dbFehler)
		}
		if abfragen != 1 {
			t.Errorf("%d Abfragen, erwartet eine", abfragen)
		}
	})
}
