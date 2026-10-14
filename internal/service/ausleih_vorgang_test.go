package service

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"

	"bibliothek/repository"

	"github.com/pashagolub/pgxmock/v5"
)

// --- Mocks: nur die im Checkout-Pfad genutzten Methoden tragen Logik,
// der Rest erfüllt das Interface als No-op. ---

type leserAttrappe struct {
	leser *repository.Leser
	err   error
}

// Finde ist die Abfrage des Ausleihpfads: Sie liest die Tabelle leser, nicht die Sicht
// schueler, und findet damit auch einen Kollegen.
func (m *leserAttrappe) Finde(ctx context.Context, id string) (*repository.Leser, error) {
	return m.leser, m.err
}
func (m *leserAttrappe) FindeZuBarcode(ctx context.Context, barcode string) (*repository.Leser, error) {
	return nil, nil
}
func (m *leserAttrappe) FindeUnscharf(ctx context.Context, q string, limit int) ([]repository.Leser, int, error) {
	return nil, 0, nil
}
func (m *leserAttrappe) HatFoto(ctx context.Context, id string) (bool, error) { return false, nil }
func (m *leserAttrappe) ZaehleOffeneSchaeden(ctx context.Context, id string) (int, float64, error) {
	return 0, 0, nil
}
func (m *leserAttrappe) LadeLaufendeAusleihen(ctx context.Context, id string) ([]repository.LaufendeAusleihe, error) {
	return nil, nil
}
func (m *leserAttrappe) LadeKlassen(ctx context.Context) ([]string, error) { return nil, nil }
func (m *leserAttrappe) LadeEtikettZeilen(ctx context.Context, ids []string) ([]repository.LeserEtikettZeile, error) {
	return nil, nil
}
func (m *leserAttrappe) LadeSchuelerListe(ctx context.Context, klassen []string, suche string, sortierung repository.LeserSortierung) ([]repository.LeserListenZeile, error) {
	return nil, nil
}
func (m *leserAttrappe) LadeLeserListe(ctx context.Context, klassen []string, suche string, sortierung repository.LeserSortierung) ([]repository.LeserListenZeile, error) {
	return nil, nil
}
func (m *leserAttrappe) LadeEhemaligenListe(ctx context.Context, suche string, sortierung repository.LeserSortierung) ([]repository.LeserListenZeile, error) {
	return nil, nil
}

type protokollAttrappe struct {
	adminAktionen int
}

func (m *protokollAttrappe) ProtokolliereAdminAktion(ctx context.Context, adminID, aktion, ip string, details map[string]any) error {
	m.adminAktionen++
	return nil
}
func (m *protokollAttrappe) LoescheTitel(ctx context.Context, t, b string) error       { return nil }
func (m *protokollAttrappe) SondereExemplarAus(ctx context.Context, c, b string) error { return nil }
func (m *protokollAttrappe) LoescheKonto(ctx context.Context, u, b string) error       { return nil }
func (m *protokollAttrappe) LegeLeserInPapierkorb(ctx context.Context, s, b, g string) error {
	return nil
}
func (m *protokollAttrappe) LoescheLeserEndgueltig(ctx context.Context, s, b string) error {
	return nil
}
func (m *protokollAttrappe) LoescheAbgaengerEndgueltig(ctx context.Context, s, b string) error {
	return nil
}
func (m *protokollAttrappe) StorniereForderung(ctx context.Context, s, b, g string) error { return nil }
func (m *protokollAttrappe) VerbucheZahlung(ctx context.Context, s, b string) error       { return nil }
func (m *protokollAttrappe) ProtokolliereAusleihe(ctx context.Context, tx pgx.Tx, e, s, bu, b string) error {
	return nil
}
func (m *protokollAttrappe) ProtokolliereRueckgabe(ctx context.Context, tx pgx.Tx, e, s, bu, b string) error {
	return nil
}
func (m *protokollAttrappe) ProtokolliereSystemAktion(ctx context.Context, tabelle, aktion, kontext string, details map[string]any) error {
	return nil
}

func strPtr(s string) *string { return &s }

// erwarteEinstellungenUndUeberfaellige richtet die Mock-Erwartungen für einen Durchlauf der
// Sperrprüfung ohne Sperre ein: offene Schäden → ladeEinstellungen → Overdue-Zählung.
func erwarteEinstellungenUndUeberfaellige(mock pgxmock.PgxPoolIface, overdueCount int) {
	mock.ExpectQuery("SELECT COUNT\\(\\*\\), COALESCE\\(SUM\\(betrag\\), 0\\)::float8 FROM schadensfaelle").
		WithArgs("s1").
		WillReturnRows(pgxmock.NewRows([]string{"count", "summe"}).AddRow(0, 0.0))
	mock.ExpectQuery("SELECT schluessel, wert FROM system_einstellungen").
		WillReturnRows(pgxmock.NewRows([]string{"schluessel", "wert"}).
			AddRow("max_overdue_items", strPtr("1")).
			AddRow("max_overdue_days", strPtr("14")))
	mock.ExpectQuery("SELECT COUNT").
		WithArgs("s1", 14).
		WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(overdueCount))
}

// erwarteFristEinstellungen: die eine Einstellungsabfrage, mit der bestimmeFrist die
// Leihfrist bestimmt.
func erwarteFristEinstellungen(mock pgxmock.PgxPoolIface) {
	mock.ExpectQuery("SELECT schluessel, wert FROM system_einstellungen").
		WillReturnRows(pgxmock.NewRows([]string{"schluessel", "wert"}).
			AddRow("standard_ausleihfrist_tage", strPtr("14")))
}

func newVorgangDienst(t *testing.T, student *repository.Leser) (*ausleihDienst, *protokollAttrappe, pgxmock.PgxPoolIface) {
	t.Helper()
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock init: %v", err)
	}
	audit := &protokollAttrappe{}
	svc := &ausleihDienst{
		pool:              mock,
		leserAbfragen:     &leserAttrappe{leser: student},
		protokollAbfragen: audit,
	}
	return svc, audit, mock
}

func aktiverSchueler(id string) *string { return &id }

// --- Tests ---

func TestLadeVorgang_GewoehnlicherFall(t *testing.T) {
	svc, _, mock := newVorgangDienst(t, &repository.Leser{
		ID: "s1", Klasse: "5a", Vorname: "Max", Nachname: "Mustermann", Art: "schueler",
	})
	defer mock.Close()

	erwarteFristEinstellungen(mock)

	copy := &repository.Exemplar{Titel: "Der Hobbit", Medientyp: "Buch", IstAusleihbar: true}
	ctx, err := svc.ladeVorgang(context.Background(), copy, aktiverSchueler("s1"))

	if err != nil {
		t.Fatalf("regulärer Schüler ohne Sperre soll durchgehen, bekam: %v", err)
	}
	if !ctx.istSchueler() || ctx.leserID != "s1" {
		t.Errorf("erwartete den Schüler s1, bekam Art %q / %q", ctx.leser.Art, ctx.leserID)
	}
	if ctx.frist.IsZero() {
		t.Error("erwartete gesetztes Fälligkeitsdatum")
	}
}

// Auflösen sperrt nicht: Ein gesperrter Schüler wird aufgelöst, die Sperre prüft der
// Checkout danach. Sperrte schon das Auflösen, wäre die eigene Rückgabe wieder gesperrt.
func TestLadeVorgang_SperrtNicht(t *testing.T) {
	svc, audit, mock := newVorgangDienst(t, &repository.Leser{
		ID: "s1", Klasse: "5a", Art: "schueler", IstGesperrt: true, Sperrgrund: strPtr("überfällig"),
	})
	defer mock.Close()

	erwarteFristEinstellungen(mock)

	copy := &repository.Exemplar{Titel: "Der Hobbit", Medientyp: "Buch", IstAusleihbar: true}
	ctx, err := svc.ladeVorgang(context.Background(), copy, aktiverSchueler("s1"))

	if err != nil {
		t.Fatalf("Auflösen darf nicht sperren, bekam: %v", err)
	}
	if ctx.leser == nil || !ctx.leser.IstGesperrt {
		t.Error("erwartete den gesperrten Leser im ausleihVorgang")
	}
	if audit.adminAktionen != 0 {
		t.Errorf("Auflösen darf kein Übergehen protokollieren, calls=%d", audit.adminAktionen)
	}
}

func TestLadeVorgang_OhneAktivenLeser(t *testing.T) {
	svc, _, mock := newVorgangDienst(t, nil)
	defer mock.Close()

	copy := &repository.Exemplar{Titel: "Der Hobbit", Medientyp: "Buch", IstAusleihbar: true}
	_, err := svc.ladeVorgang(context.Background(), copy, nil)

	if !errors.Is(err, ErrUngueltigerZustand) {
		t.Errorf("ohne aktiven Leser soll ErrUngueltigerZustand kommen, bekam: %v", err)
	}
}
