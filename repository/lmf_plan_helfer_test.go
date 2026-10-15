package repository

import (
	"context"

	"bibliothek/db"
	"bibliothek/pkg/lmfplan"
)

// speichereLmfPlanImTest schreibt einen Plan so, wie es die Tür tut: in einer Transaktion, die
// der Rufer hält (SchreibePlan). Die Tests laufen damit durch denselben Einstieg wie der
// Betrieb.
func speichereLmfPlanImTest(ctx context.Context, repo *LMFAbfragen, plan LMFPlan, zeilen []LMFPlanZeile, plaetze []lmfplan.Platz, ausgelassen []string) (LMFPlanStand, error) {
	tx, err := repo.db.Begin(ctx)
	if err != nil {
		return LMFPlanStand{}, err
	}
	defer db.Verwerfe(ctx, tx)
	st, err := repo.SchreibePlan(ctx, tx, plan, zeilen, plaetze, ausgelassen)
	if err != nil {
		return st, err
	}
	return st, tx.Commit(ctx)
}
