package testweg

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"bibliothek/repository"
)

// LUSDSchuelerNeu sind die Angaben für LegeLUSDSchuelerAn. Ein Abgänger entsteht gesperrt, mit
// dem Sperrgrund der Abgänger und einer Karenz, die seit zehn Tagen läuft.
type LUSDSchuelerNeu struct {
	Vorname, Nachname, Klasse, Barcode string
	Geburtsdatum, SchulEintrittAm      *time.Time
	IstAbgaenger                       bool
}

// LegeLUSDSchuelerAn legt einen Schüler an, wie ihn der LUSD-Import hinterlässt: mit
// Geburtsdatum und Schuleintritt, vor einem Jahr bestätigt.
func LegeLUSDSchuelerAn(t *testing.T, pool *pgxpool.Pool, s LUSDSchuelerNeu) string {
	t.Helper()
	var id string
	if err := pool.QueryRow(context.Background(), `
		INSERT INTO schueler (vorname, nachname, klasse, barcode_id, abgaenger_jahr, geburtsdatum,
		                      schul_eintritt_am, ist_abgaenger, lusd_bestaetigt_am, ist_gesperrt, block_reason, abgaenger_seit)
		VALUES ($1,$2,$3,$4,2031,$5,$6,$7,NOW() - interval '1 year',$7,
		        CASE WHEN $7 THEN '`+repository.AbgaengerSperrgrundKarenz+`' END,
		        CASE WHEN $7 THEN NOW() - interval '10 days' END)
		RETURNING id`,
		s.Vorname, s.Nachname, s.Klasse, s.Barcode, s.Geburtsdatum, s.SchulEintrittAm, s.IstAbgaenger).Scan(&id); err != nil {
		t.Fatalf("Schüler %s %s anlegen: %v", s.Vorname, s.Nachname, err)
	}
	return id
}
