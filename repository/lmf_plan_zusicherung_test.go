package repository

import (
	"context"
	"testing"

	"bibliothek/pkg/lmfplan"
)

// SchreibePlan indiziert plaetze[i] über die Zeilen. Dass beide Scheiben gleich lang
// sind, sichert auch der eine Aufrufer zu (api/lernmittel/lmf_plan.go, nach
// pruefeLMFPlan). Diese Zusicherung gehört an die Tür, die sie braucht: Ein zweiter
// Aufrufer — oder ein Umbau der Prüfung — bekommt sonst keinen Fehler, sondern einen
// Indexfehler mitten in einer offenen Transaktion.
//
// Der Test kommt ohne Datenbank aus: Die Prüfung steht vor jedem Zugriff auf die
// Transaktion, deshalb ist tx hier nil. Genau das ist die Aussage — vor dem ersten
// Schreiben, nicht irgendwo in der Schleife. Ohne die Prüfung endet derselbe Aufruf
// nicht mit einem Fehler, sondern mit einem Panic (am Rückbau gesehen).
func TestSchreibeLMFPlan_PlatzJeZeile(t *testing.T) {
	repo := NewLMFAbfragen(nil)
	plan := LMFPlan{Art: LMFTerminAusgabe, ErsterTag: "2026-09-10", Startstunde: 1, StundenJeTag: 6}
	zeilen := []LMFPlanZeile{{Klassen: []string{"05A"}}, {Klassen: []string{"05B"}}}

	_, err := repo.SchreibePlan(context.Background(), nil, plan, zeilen, []lmfplan.Platz{{}}, nil)
	if err == nil {
		t.Fatal("ein Platz für zwei Zeilen wurde angenommen")
	}
}
