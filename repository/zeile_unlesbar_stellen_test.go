package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pashagolub/pgxmock/v5"
)

// Eine Zeile, die sich nicht lesen lässt, bricht das Lesen ab und meldet ErrZeileUnlesbar. Ließe
// die Abfrage sie aus, käme die Liste kürzer an, und niemand erführe es. Nachgestellt wird der
// häufigste Fall: NULL in einer Spalte, die in einen Go-Typ ohne Zeiger gelesen wird. Jede Zeile
// trägt so viele Spalten, wie die Abfrage liest; sonst scheiterte das Lesen an der Zahl.
func TestUnlesbareZeileBrichtAb(t *testing.T) {
	jetzt := time.Now()
	type lesen func(ctx context.Context, mock pgxmock.PgxPoolIface) error
	faelle := []struct {
		name string
		lies lesen
	}{
		{"Klassenleitungen", func(ctx context.Context, mock pgxmock.PgxPoolIface) error {
			mock.ExpectQuery(`FROM klassen_lehrer_mapping ORDER BY klasse`).WillReturnRows(
				pgxmock.NewRows([]string{"klasse", "lehrer_email", "erstellt_am"}).AddRow(nil, "a@schule.de", jetzt))
			_, err := LadeKlassenleitungen(ctx, mock)
			return err
		}},
		{"Kontoauszug", func(ctx context.Context, mock pgxmock.PgxPoolIface) error {
			mock.ExpectQuery(`WHERE a.schueler_id = \$1 AND a.rueckgabe_am IS NULL`).WithArgs(pgxmock.AnyArg()).WillReturnRows(
				pgxmock.NewRows([]string{"titel", "barcode_id", "ausgeliehen_am", "rueckgabe_frist"}).AddRow(nil, "B-1", jetzt, jetzt))
			_, err := LadeKontoauszugBuecher(ctx, mock, uuid.New())
			return err
		}},
		{"Protokoll der Verwaltung", func(ctx context.Context, mock pgxmock.PgxPoolIface) error {
			mock.ExpectQuery(`FROM audit_logs a`).WillReturnRows(
				pgxmock.NewRows([]string{"id", "admin_id", "admin_name", "aktion", "details", "ip", "zeitstempel"}).
					AddRow(nil, nil, "Nina", "LOGIN", "{}", "10.0.0.1", jetzt))
			_, err := LadeAdminProtokoll(ctx, mock)
			return err
		}},
		{"Renner", func(ctx context.Context, mock pgxmock.PgxPoolIface) error {
			mock.ExpectQuery(`COUNT\(a.id\) AS count`).WillReturnRows(
				pgxmock.NewRows([]string{"id", "titel", "autor", "cover", "fach", "signatur", "jahr", "count"}).
					AddRow(nil, "Momo", "", "", "", "", 0, 3))
			_, err := LadeRenner(ctx, mock, "", "", 10)
			return err
		}},
		{"Ladenhüter", func(ctx context.Context, mock pgxmock.PgxPoolIface) error {
			mock.ExpectQuery(`AS last_loan`).WillReturnRows(
				pgxmock.NewRows([]string{"id", "titel", "autor", "isbn", "fach", "signatur", "jahr", "last_loan"}).
					AddRow(nil, "Momo", "", "", "", "", 0, nil))
			_, err := LadeLadenhueter(ctx, mock, "", 10)
			return err
		}},
		{"Monatstrend", func(ctx context.Context, mock pgxmock.PgxPoolIface) error {
			mock.ExpectQuery(`WITH monate AS`).WillReturnRows(
				pgxmock.NewRows([]string{"monat", "ausleihen", "rueckgaben"}).AddRow(nil, 1, 1))
			_, err := LadeMonatsTrend(ctx, mock, "")
			return err
		}},
		{"Felder der Etiketten je Barcode", func(ctx context.Context, mock pgxmock.PgxPoolIface) error {
			mock.ExpectQuery(`barcode_id = ANY`).WithArgs(pgxmock.AnyArg()).WillReturnRows(
				pgxmock.NewRows([]string{"barcode_id", "jahr", "signatur", "topf"}).AddRow(nil, "2024", "", "land"))
			_, err := LadeEtikettServerfelderZuBarcodes(ctx, mock, []string{"B-1"})
			return err
		}},
		{"Klassen der Leserdatei", func(ctx context.Context, mock pgxmock.PgxPoolIface) error {
			mock.ExpectQuery(`SELECT klasse FROM schueler`).WillReturnRows(
				pgxmock.NewRows([]string{"klasse"}).AddRow(nil))
			_, err := NewLeserAbfragen(mock).LadeKlassen(ctx)
			return err
		}},
		{"Adressen der Klassenleitungen im Mahnwesen", func(ctx context.Context, mock pgxmock.PgxPoolIface) error {
			mock.ExpectQuery(`FROM klassen_lehrer_mapping`).WillReturnRows(
				pgxmock.NewRows([]string{"klasse", "lehrer_email"}).AddRow(nil, "a@schule.de"))
			return NewMahnwesenAbfragen(mock).reichereLehrerEmailsAn(ctx, []MahnwesenKlasse{{Klasse: "5a"}})
		}},
		{"Mail-Vorlagen", func(ctx context.Context, mock pgxmock.PgxPoolIface) error {
			mock.ExpectQuery(`FROM mail_vorlagen`).WillReturnRows(
				pgxmock.NewRows([]string{"id", "typ", "betreff", "text_body", "updated_at"}).AddRow(nil, "mahnung", "", "", jetzt))
			_, err := LadeMailVorlagen(ctx, mock)
			return err
		}},
		{"Sachgruppen", func(ctx context.Context, mock pgxmock.PgxPoolIface) error {
			mock.ExpectQuery(`FROM systematik_kategorien`).WillReturnRows(
				pgxmock.NewRows([]string{"id", "kuerzel", "bezeichnung"}).AddRow(nil, "Ma", "Mathematik"))
			_, err := LadeSystematikKategorien(ctx, mock)
			return err
		}},
		{"Fächer", func(ctx context.Context, mock pgxmock.PgxPoolIface) error {
			mock.ExpectQuery(`WHERE subject IS NOT NULL`).WillReturnRows(
				pgxmock.NewRows([]string{"subject"}).AddRow(nil))
			_, err := LadeFaecher(ctx, mock)
			return err
		}},
		{"Rechte-Matrix", func(ctx context.Context, mock pgxmock.PgxPoolIface) error {
			mock.ExpectQuery(`FROM role_permissions`).WillReturnRows(
				pgxmock.NewRows([]string{"role", "permission", "allowed"}).AddRow(nil, "view_books", true))
			_, err := LadeRollenRechte(ctx, mock)
			return err
		}},
		{"Titel für den Abgleich der Cover", func(ctx context.Context, mock pgxmock.PgxPoolIface) error {
			mock.ExpectQuery(`SELECT id, isbn FROM buecher_titel`).WillReturnRows(
				pgxmock.NewRows([]string{"id", "isbn"}).AddRow(nil, "9783161484100"))
			_, err := LadeTitelFuerCoverAbgleich(ctx, mock)
			return err
		}},
		{"Einstellungen", func(ctx context.Context, mock pgxmock.PgxPoolIface) error {
			mock.ExpectQuery(`FROM system_einstellungen`).WillReturnRows(
				pgxmock.NewRows([]string{"schluessel", "wert"}).AddRow(nil, nil))
			_, err := LadeEinstellungen(ctx, mock)
			return err
		}},
		{"Reservierungen", func(ctx context.Context, mock pgxmock.PgxPoolIface) error {
			mock.ExpectQuery(`AS angefordert_von`).WillReturnRows(
				pgxmock.NewRows([]string{"id", "titel_id", "titel", "cover_url", "klasse", "anzahl", "notiz", "erledigt",
					"erstellt_am", "angefordert_von", "verfuegbar", "erledigt_notiz", "erledigt_am"}).
					AddRow(nil, "t-1", "Momo", "", "5a", 25, nil, false, jetzt, nil, 30, nil, nil))
			_, err := NewReservierungsAbfragen(mock).LadeAlle(ctx)
			return err
		}},
		{"offene Ausleihen vor dem Löschen eines Titels", func(ctx context.Context, mock pgxmock.PgxPoolIface) error {
			const titelID = "33333333-3333-3333-3333-333333333333"
			mock.ExpectBegin()
			erwarteAuflagenSperre(mock, []string{titelID})
			mock.ExpectQuery("FROM buecher_titel WHERE id").WithArgs(titelID).WillReturnRows(snapshotTitelRows())
			mock.ExpectQuery("FROM ausleihen a").WithArgs(titelID).WillReturnRows(
				pgxmock.NewRows([]string{"barcode_id"}).AddRow(nil))
			// Kein ExpectExec: Bei unklarer Lage wird nichts gelöscht.
			mock.ExpectRollback()
			return NewKatalogAbfragen(mock).LoescheTitel(ctx, titelID, "admin-1")
		}},
	}
	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			mock, err := pgxmock.NewPool()
			if err != nil {
				t.Fatalf("pgxmock: %v", err)
			}
			defer mock.Close()
			if err := f.lies(context.Background(), mock); !errors.Is(err, ErrZeileUnlesbar) {
				t.Errorf("Fehler = %v, erwartet ErrZeileUnlesbar", err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("Erwartungen an die Datenbank: %v", err)
			}
		})
	}
}
