package service

// Eine Lehrkraft ohne Ausweisnummer muss ausleihen können: Bis ein Ausweis gedruckt ist, steht
// sie an der Theke über die Namenssuche zur Verfügung, und ihre Kennung geht als aktiver Leser
// in die Buchung.
//
// Der Ausleihpfad liest die Leserzeile (Finde) und damit dieselbe Quelle wie die Theke; die
// NULL-Spalte der Ausweisnummer kommt als coalesce, und die Frist entscheidet die Art. Ein
// Fehler darf dabei nicht zu „nicht gefunden" (404) werden: Sonst sähe ein Datenbank-Aussetzer
// aus wie ein Bedienfehler, und die Lehrkraft ohne Ausweis scheiterte am Scan der NULL-Spalte
// in einen string.

import (
	"context"
	"fmt"
	"testing"
	"time"

	"bibliothek/internal/pgtest"
	"bibliothek/internal/testdaten"
	"bibliothek/repository"
)

func TestLadeVorgang_LehrkraftOhneAusweis(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())

	// Ein aktives Konto bekommt eine Nummer und behält sie (Migrationen 136 und 145). Ohne
	// Nummer ist eine Lehrkraft nur ohne aktives Konto — etwa ohne Schul-Adresse, solange die
	// Akte keine nachgetragen hat. Der Ausleihpfad muss den Zustand tragen.
	leserID := testdaten.LegeLeserAn(t, pool, "", "Ohne", "Ausweis-"+suffix, "lehrkraft")
	t.Cleanup(func() {
		if _, err := pool.Exec(ctx, `DELETE FROM leser WHERE id = $1`, leserID); err != nil {
			t.Errorf("Aufräumen (Leser): %v", err)
		}
	})

	svc := &AusleihDienst{pool: pool, leserAbfragen: repository.NewLeserAbfragen(pool)}
	chk, err := svc.ladeVorgang(ctx, &repository.Exemplar{}, &leserID)
	if err != nil {
		t.Fatalf("eine aktive Lehrkraft ohne Ausweis muss ausleihen können, bekam: %v", err)
	}
	if chk.leserID != leserID || chk.leser == nil || chk.leser.BarcodeID != "" {
		t.Fatalf("ladeVorgang(…) = Leser %q, %+v, erwartet %s ohne Ausweisnummer", chk.leserID, chk.leser, leserID)
	}
	if chk.istSchueler() {
		t.Errorf("eine Lehrkraft ist kein Schüler, Art gelesen: %q", chk.leser.Art)
	}
	// Ohne Klasse gibt es keine Schuljahresfrist: Nicht-Schüler bekommen ein Jahr.
	if chk.frist.Before(time.Now().AddDate(0, 11, 0)) {
		t.Errorf("Frist %s liegt zu früh — erwartet rund ein Jahr", chk.frist)
	}
}
