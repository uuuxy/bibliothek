package repository

import (
	"context"

	"bibliothek/db"
	"bibliothek/pkg/lmfplan"
)

// speichereLmfPlanImTest schreibt einen Plan so, wie es der Handler tut: in einer
// Transaktion, die der Aufrufer hält (SaveLmfPlanIn). Bis 07.09.2026 gab es dafür eine
// Hülle SaveLmfPlan in der Produktion — die rief nach der Transaktionsklammer
// (api/lmf_plan.go) niemand mehr, und das deadcode-Gate wies sie als tote Tür aus. Die
// Tests laufen jetzt durch denselben Einstieg wie der Betrieb.
func speichereLmfPlanImTest(ctx context.Context, repo *LmfTerminRepository, plan LmfPlan, zeilen []LmfPlanZeile, plaetze []lmfplan.Platz, ausgelassen []string) (LmfPlanStand, error) {
	tx, err := repo.db.Begin(ctx)
	if err != nil {
		return LmfPlanStand{}, err
	}
	defer db.SafeRollback(ctx, tx)
	st, err := repo.SaveLmfPlanIn(ctx, tx, plan, zeilen, plaetze, ausgelassen)
	if err != nil {
		return st, err
	}
	return st, tx.Commit(ctx)
}
