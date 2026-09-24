package service

// Der eine Prüfweg für Buch und Gerät (pruefeAusleihSperren, entschieden am 24.09.2026).
// Die Mocks erwarten jede Abfrage einzeln: Läuft eine Regel, die nicht laufen dürfte, meldet
// pgxmock die unerwartete Query als Fehler; fehlt eine, meldet es ExpectationsWereMet.
// Den Live-Pfad der Theke gegen echtes Postgres hält api/ausleih_sperren_pg_test.go; die
// Rückbau-Proben stehen dort je Test.

import (
	"context"
	"errors"
	"testing"

	"bibliothek/repository"

	"github.com/pashagolub/pgxmock/v5"
)

// pruefeMitMock ruft den Prüfweg mit einem Mock-Pool, auf dem vorbereiten die erwarteten
// Abfragen einträgt, und prüft danach, dass genau diese liefen.
func pruefeMitMock(t *testing.T, leser *repository.Student, lernmittel, uebergehen bool, vorbereiten func(pgxmock.PgxPoolIface)) (sperrLage, error) {
	t.Helper()
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock init: %v", err)
	}
	defer mock.Close()
	vorbereiten(mock)
	lage, err := pruefeAusleihSperren(context.Background(), mock, leser, lernmittel, uebergehen)
	if merr := mock.ExpectationsWereMet(); merr != nil {
		t.Errorf("Abfragen: %v", merr)
	}
	return lage, err
}

func keineAbfrage(pgxmock.PgxPoolIface) {}

func schueler(mod func(*repository.Student)) *repository.Student {
	s := &repository.Student{ID: "s1", Art: "schueler", Klasse: "5a"}
	if mod != nil {
		mod(s)
	}
	return s
}

// Eine Sperre am Leser lässt nur die Rückgabe zu — auch mit override_block. Sie trägt das
// Merkmal „leser" (die Theke bietet an, sie aufzuheben), nicht „uebergehbar". Wie Littera:
// „Für eine Ausleihe muss jedoch die Sperre zunächst in den Leserstammdaten aufgehoben
// werden!"
func TestAusleihSperren_SperreAmLeserLaesstSichNichtUebergehen(t *testing.T) {
	faelle := map[string]*repository.Student{
		"von Hand": schueler(func(s *repository.Student) {
			s.IsManuallyBlocked, s.BlockReason = true, strPtr("Ausweis verloren")
		}),
		"vom Programm (Ehemalige)": schueler(func(s *repository.Student) {
			s.IstGesperrt, s.BlockReason = true, strPtr("Automatisierte Abgänger-Sperre (offene Vorgänge)")
		}),
	}
	for name, leser := range faelle {
		t.Run(name, func(t *testing.T) {
			for _, uebergehen := range []bool{false, true} {
				_, err := pruefeMitMock(t, leser, false, uebergehen, keineAbfrage)
				if !errors.Is(err, ErrBlocked) {
					t.Fatalf("uebergehen=%v: erwartet ErrBlocked, bekam %v", uebergehen, err)
				}
				if !IstSperreAmLeser(err) || IstUebergehbareSperre(err) {
					t.Errorf("uebergehen=%v: erwartet Merkmal „leser“, bekam leser=%v uebergehbar=%v",
						uebergehen, IstSperreAmLeser(err), IstUebergehbareSperre(err))
				}
				var sg *SperrGrundFehler
				if !errors.As(err, &sg) || sg.Grund != *leser.BlockReason {
					t.Errorf("der Grund fehlt in der Meldung: %v", err)
				}
			}
		})
	}
}

// Beim Lernmittel gilt nur die Sperre von Hand (Antwort der Schule vom 22.09.2026: keine
// automatische Sperrung, „auch nicht eine Sperrung, die bestimmte Personen aufheben
// können"). Die Sperre der Ehemaligen setzt das Programm — sie zählt dort nicht, und keine
// der zwei Automatiken fragt die Datenbank.
func TestAusleihSperren_LernmittelNurSperreVonHand(t *testing.T) {
	t.Run("Ehemalige, offene Forderung, überfällig: keine Abfrage, keine Sperre", func(t *testing.T) {
		leser := schueler(func(s *repository.Student) {
			s.IstGesperrt, s.BlockReason = true, strPtr("Automatisierte Abgänger-Sperre (Schuljahreswechsel)")
		})
		lage, err := pruefeMitMock(t, leser, true, false, keineAbfrage)
		if err != nil {
			t.Fatalf("Lernmittel an einen Ehemaligen abgewiesen: %v", err)
		}
		if len(lage.uebergangen) != 0 {
			t.Errorf("nichts wurde übergangen, trotzdem %v", lage.uebergangen)
		}
	})
	t.Run("Gegenprobe: die Sperre von Hand hält auch das Lernmittel auf", func(t *testing.T) {
		leser := schueler(func(s *repository.Student) {
			s.IsManuallyBlocked, s.BlockReason = true, strPtr("Hausverbot")
		})
		if _, err := pruefeMitMock(t, leser, true, false, keineAbfrage); !IstSperreAmLeser(err) {
			t.Errorf("Sperre von Hand beim Lernmittel: erwartet Sperre am Leser, bekam %v", err)
		}
	})
}

// Ein Kollege wird nie gesperrt (16.09. und 24.09.2026) — auch nicht, wenn an seinem Konto
// noch eine Sperre steht, die der Knopf bis zum 24.09.2026 setzen konnte. Keine Abfrage:
// Forderung und Überfällig zählen bei ihm nicht.
func TestAusleihSperren_KollegeWirdNieGesperrt(t *testing.T) {
	for _, art := range []string{"lehrkraft", "liv"} {
		leser := &repository.Student{ID: "k1", Art: art, IsManuallyBlocked: true, IstGesperrt: true, BlockReason: strPtr("alt")}
		if _, err := pruefeMitMock(t, leser, false, false, keineAbfrage); err != nil {
			t.Errorf("%s abgewiesen: %v", art, err)
		}
	}
	t.Run("leere Art heißt Schüler (Sicht schueler) — die Sperre gilt", func(t *testing.T) {
		leser := &repository.Student{ID: "s1", IsManuallyBlocked: true, BlockReason: strPtr("Test")}
		if _, err := pruefeMitMock(t, leser, false, false, keineAbfrage); !IstSperreAmLeser(err) {
			t.Errorf("erwartet Sperre am Leser, bekam %v", err)
		}
	})
}

// Ein anonymisierter Datensatz ist keine Person mehr: nichts, auch kein Lernmittel. Ohne
// Merkmal — aufheben lässt sich diese Sperre nicht (api/student_lock.go lehnt ab).
func TestAusleihSperren_AnonymisiertBekommtNichts(t *testing.T) {
	leser := schueler(func(s *repository.Student) {
		s.IstAnonymisiert, s.IstGesperrt, s.BlockReason = true, true, strPtr("Abgänger anonymisiert")
	})
	for _, lernmittel := range []bool{false, true} {
		_, err := pruefeMitMock(t, leser, lernmittel, true, keineAbfrage)
		if !errors.Is(err, ErrBlocked) || IstSperreAmLeser(err) || IstUebergehbareSperre(err) {
			t.Errorf("lernmittel=%v: erwartet ErrBlocked ohne Merkmal, bekam %v", lernmittel, err)
		}
	}
}

// Offene Forderung und Überfällig-Automatik sind Hinweise: ohne override_block gesperrt, mit
// ihm durch — und in sperrLage.uebergangen fürs Protokoll. Bis zum 13.09.2026 entschied die
// Theke am Wortlaut, und die Schadens-Sperre bekam keinen Dialog; das Merkmal hält es.
func TestAusleihSperren_HinweiseSindUebergehbar(t *testing.T) {
	forderung := func(mock pgxmock.PgxPoolIface) {
		mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM schadensfaelle").
			WithArgs("s1").WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(1))
	}
	ueberfaellig := func(mock pgxmock.PgxPoolIface) { expectSettingsAndOverdue(mock, 2) }

	t.Run("offene Forderung ohne override_block", func(t *testing.T) {
		_, err := pruefeMitMock(t, schueler(nil), false, false, forderung)
		if !errors.Is(err, ErrBlocked) || !IstUebergehbareSperre(err) {
			t.Errorf("erwartet übergehbare Sperre, bekam %v", err)
		}
	})
	t.Run("überfällig ohne override_block", func(t *testing.T) {
		_, err := pruefeMitMock(t, schueler(nil), false, false, ueberfaellig)
		if !errors.Is(err, ErrBlocked) || !IstUebergehbareSperre(err) {
			t.Errorf("erwartet übergehbare Sperre, bekam %v", err)
		}
	})
	t.Run("beide übergangen", func(t *testing.T) {
		lage, err := pruefeMitMock(t, schueler(nil), false, true, func(mock pgxmock.PgxPoolIface) {
			forderung(mock)
			mock.ExpectQuery("SELECT schluessel, coalesce\\(wert, ''\\) FROM system_einstellungen").
				WillReturnRows(pgxmock.NewRows([]string{"schluessel", "wert"}).
					AddRow("max_overdue_items", "1").AddRow("max_overdue_days", "14"))
			mock.ExpectQuery("SELECT COUNT").
				WithArgs("s1", 14).WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(3))
		})
		if err != nil {
			t.Fatalf("override_block soll beide Hinweise übergehen: %v", err)
		}
		if len(lage.uebergangen) != 2 || lage.einst == nil {
			t.Errorf("erwartet zwei Einträge fürs Protokoll und die Einstellungen, bekam %v / %v", lage.uebergangen, lage.einst)
		}
	})
	t.Run("ohne Hinweis: nichts übergangen", func(t *testing.T) {
		lage, err := pruefeMitMock(t, schueler(nil), false, true, func(mock pgxmock.PgxPoolIface) {
			expectSettingsAndOverdue(mock, 0)
		})
		if err != nil || len(lage.uebergangen) != 0 {
			t.Errorf("erwartet freie Ausleihe ohne Protokolleintrag, bekam err=%v uebergangen=%v", err, lage.uebergangen)
		}
	})
}

// Das Protokoll schreibt je übergangenen Hinweis einen Eintrag — und keinen, wenn nichts
// übergangen wurde.
func TestAusleihSperren_ProtokollJeHinweis(t *testing.T) {
	audit := &mockAuditRepo{}
	protokolliereUebergangen(context.Background(), audit, "staff1", "s1", nil)
	if audit.adminAktionCalls != 0 {
		t.Fatalf("ohne Übergehen %d Einträge", audit.adminAktionCalls)
	}
	protokolliereUebergangen(context.Background(), audit, "staff1", "s1", []string{"a", "b"})
	if audit.adminAktionCalls != 2 {
		t.Errorf("erwartet 2 Einträge, bekam %d", audit.adminAktionCalls)
	}
}

// Gegenprobe zum Merkmal: Ein defektes Gerät ist keine Sperre am Leser und kein Hinweis —
// die Theke meldet es nur.
func TestAusleihSperren_GesperrtesGeraetOhneMerkmal(t *testing.T) {
	err := pruefeGeraetAusleihbar(repository.Geraet{IstAusleihbar: false})
	if !errors.Is(err, ErrBlocked) || IstUebergehbareSperre(err) || IstSperreAmLeser(err) {
		t.Errorf("Geräte-Sperre: erwartet ErrBlocked ohne Merkmal, bekam %v", err)
	}
}
