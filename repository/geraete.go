package repository

// geraete.go — die Verwaltungsseite der Geräteausleihe.
//
// Der Ausleihweg (Kiosk, G--Präfix → device_service) existierte lange vor der
// Verwaltung: Geräte ließen sich nur per Hand-SQL anlegen, FACHKONZEPT §5 stand
// als „angelegt, nicht in Betrieb". Diese Datei liefert die fehlende Tür:
// anlegen, auflisten (mit aktuellem Ausleiher), Status pflegen.

import (
	"context"
	"errors"

	"bibliothek/db"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// ErrGeraetBarcodeVergeben meldet, dass der Barcode bereits auf einem anderen Gerät klebt.
// Nutzer-sichtbare Meldung (409), daher deutsche Großschreibung.
//
//nolint:staticcheck // ST1005: bewusst großgeschrieben, Endnutzer-Meldung
var ErrGeraetBarcodeVergeben = errors.New("Barcode ist bereits an ein anderes Gerät vergeben")

// ErrGeraetSeriennummerVergeben meldet, dass die Seriennummer schon an einem anderen Gerät steht.
//
//nolint:staticcheck // ST1005: bewusst großgeschrieben, Endnutzer-Meldung
var ErrGeraetSeriennummerVergeben = errors.New("Seriennummer ist bereits an einem anderen Gerät eingetragen")

// geraetEindeutigkeit übersetzt eine verletzte Eindeutigkeit in den Fehler, der das Feld
// nennt. Die Tabelle hat zwei: Barcode und Seriennummer.
func geraetEindeutigkeit(err error) error {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "23505" {
		return err
	}
	if pgErr.ConstraintName == "geraete_seriennummer_key" {
		return ErrGeraetSeriennummerVergeben
	}
	return ErrGeraetBarcodeVergeben
}

// GeraetMitStatus ist eine Zeile der Geräte-Verwaltung: die Stammdaten plus der
// aktuelle Ausleiher (nil = im Schrank).
type GeraetMitStatus struct {
	Geraet
	AusgeliehenAn *string `json:"ausgeliehen_an,omitempty"`
}

// GeraeteRepository kapselt die Datenbankzugriffe der Geräte-Verwaltung.
type GeraeteRepository interface {
	ListGeraete(ctx context.Context) ([]GeraetMitStatus, error)
	CreateGeraet(ctx context.Context, modellname string, seriennummer *string, barcode, zubehoer string) (string, error)
	// UpdateGeraet pflegt Stammdaten und Ausleihstatus (ist_ausleihbar=false = defekt/gesperrt).
	UpdateGeraet(ctx context.Context, id, modellname, zubehoer string, zustandNotiz, seriennummer *string, istAusleihbar *bool) error
}

type pgGeraeteRepository struct {
	db db.PgxPoolIface
}

// NewGeraeteRepository bindet die Geräte-Verwaltung an einen Pool.
func NewGeraeteRepository(pool db.PgxPoolIface) GeraeteRepository {
	return &pgGeraeteRepository{db: pool}
}

// ListGeraete liefert alle nicht ausgesonderten Geräte, neueste zuerst, mit dem
// Namen des aktuellen Ausleihers (Schüler ODER Lehrkraft — check_loan_borrower
// garantiert höchstens einen).
func (r *pgGeraeteRepository) ListGeraete(ctx context.Context) ([]GeraetMitStatus, error) {
	rows, err := r.db.Query(ctx, `
		SELECT g.id, g.modellname, g.seriennummer, g.barcode_id, g.zubehoer,
		       g.ist_ausleihbar, g.ist_ausgesondert, g.zustand_notiz,
		       g.erstellt_am, g.aktualisiert_am,
		       btrim(l.vorname || ' ' || l.nachname ||
		             coalesce(' (' || nullif(l.klasse, '') || ')', '')) AS ausgeliehen_an
		FROM geraete g
		LEFT JOIN ausleihen a ON a.geraet_id = g.id AND a.rueckgabe_am IS NULL
		-- leser, nicht die Sicht schueler: Sonst stünde bei einem Gerät in der Hand
		-- eines Kollegen kein Name (Migration 125).
		LEFT JOIN leser l ON l.id = a.schueler_id
		WHERE g.ist_ausgesondert = false
		ORDER BY g.erstellt_am DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	geraete := []GeraetMitStatus{}
	for rows.Next() {
		var g GeraetMitStatus
		if err := rows.Scan(
			&g.ID, &g.Modellname, &g.Seriennummer, &g.BarcodeID, &g.Zubehoer,
			&g.IstAusleihbar, &g.IstAusgesondert, &g.ZustandNotiz,
			&g.ErstelltAm, &g.AktualisiertAm, &g.AusgeliehenAn,
		); err != nil {
			return nil, err
		}
		geraete = append(geraete, g)
	}
	return geraete, rows.Err()
}

func (r *pgGeraeteRepository) CreateGeraet(ctx context.Context, modellname string, seriennummer *string, barcode, zubehoer string) (string, error) {
	var id string
	err := r.db.QueryRow(ctx, `
		INSERT INTO geraete (modellname, seriennummer, barcode_id, zubehoer)
		VALUES ($1, NULLIF($2, ''), $3, $4)
		RETURNING id
	`, modellname, seriennummer, barcode, zubehoer).Scan(&id)
	if err != nil {
		return "", geraetEindeutigkeit(err)
	}
	return id, nil
}

// UpdateGeraet schreibt die Stammdaten. seriennummer und istAusleihbar sind Zeiger: nil heißt
// „nicht mitgeschickt" und lässt die Spalte in Ruhe. Der Bearbeiten-Dialog schickt das
// Defekt-Kennzeichen nie (es liegt auf einem eigenen Knopf); eine Vorgabe dafür gäbe ein
// defektes Gerät bei jeder Korrektur wieder frei. Eine leere Seriennummer heißt „keine" und
// wird wie beim Anlegen NULL: Als leerer Text stieße das zweite Gerät ohne Seriennummer an
// die Eindeutigkeit.
func (r *pgGeraeteRepository) UpdateGeraet(ctx context.Context, id, modellname, zubehoer string, zustandNotiz, seriennummer *string, istAusleihbar *bool) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE geraete
		SET modellname = $1, zubehoer = $2, zustand_notiz = $3,
		    seriennummer = CASE WHEN $4::text IS NULL THEN seriennummer ELSE NULLIF($4, '') END,
		    ist_ausleihbar = COALESCE($5, ist_ausleihbar),
		    aktualisiert_am = CURRENT_TIMESTAMP
		WHERE id = $6 AND ist_ausgesondert = false
	`, modellname, zubehoer, zustandNotiz, seriennummer, istAusleihbar, id)
	if err != nil {
		return geraetEindeutigkeit(err)
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}
