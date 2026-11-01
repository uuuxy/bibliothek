import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, fireEvent, screen, waitFor } from '@testing-library/svelte';
import PortalLmfPlan from './PortalLmfPlan.svelte';
import { ladePdf, ladePlan } from '../lmfplan/lmfplanDienst.js';
import { snackbars } from '../../stores/snackbars.svelte.js';

vi.mock('../lmfplan/lmfplanDienst.js', async (importOriginal) => ({
	.../** @type {any} */ (await importOriginal()),
	ladePlan: vi.fn(),
	ladePdf: vi.fn()
}));
const leitung = vi.hoisted(() => ({
	/** @type {Map<string, () => void>} */
	hoerer: new Map()
}));
vi.mock('../../liveEvents.js', () => ({
	abonniereNeuladen: (/** @type {string} */ name, /** @type {() => void} */ holeNeu) => {
		leitung.hoerer.set(name, holeNeu);
		return () => leitung.hoerer.delete(name);
	}
}));

/** @type {{ ab: string, termine: import('../lmfplan/lmfplanDienst.js').LmfTermin[], eingangsjahrgaenge: number[] }} */
const PLAN = {
	ab: '2026-08-01',
	termine: [
		{ id: 'a', datum: '2027-06-28', stunde: 3, art: 'rueckgabe', klassen: ['09H1'], vermerk: '' }
	],
	eingangsjahrgaenge: []
};

beforeEach(() => {
	vi.mocked(ladePlan).mockReset().mockResolvedValue(PLAN);
	vi.mocked(ladePdf).mockReset();
	for (const t of [...snackbars.toasts]) snackbars.removeToast(t.id);
});

// Scheitert das PDF, sagt eine Meldung warum; vorher geschah nach dem Klick nichts. Dieselbe
// Meldung wie am Knopf des Planers in der Verwaltung (lmfplanPlaner.svelte.js).
describe('LMF-Plan im Portal: „Als PDF" scheitert', () => {
	it('nennt den Grund, den der Server angibt', async () => {
		vi.mocked(ladePdf).mockRejectedValue(new Error('Keine Berechtigung für diese Aktion'));
		render(PortalLmfPlan);
		await screen.findByText('09H1');

		await fireEvent.click(screen.getByRole('button', { name: 'Als PDF' }));

		await waitFor(() =>
			expect(snackbars.toasts.map((t) => [t.message, t.type])).toEqual([
				['Keine Berechtigung für diese Aktion', 'error']
			])
		);
	});
});

// Die Seite läuft im Lehrerzimmer nebenbei: Eine verschobene Stunde kommt ohne F5, auch wenn der
// Rechner dazwischen geschlafen hat (den Aufbau der Leitung deckt liveEvents.test.js).
describe('PortalLmfPlan: der Plan bleibt aktuell', () => {
	it('holt den Plan neu, wenn sich etwas ändert, und hört nach dem Verlassen auf', async () => {
		vi.mocked(ladePlan).mockResolvedValue(
			/** @type {any} */ ({ termine: [], eingangsjahrgaenge: [] })
		);
		const seite = render(PortalLmfPlan);
		await waitFor(() => expect(ladePlan).toHaveBeenCalled());
		const vorher = vi.mocked(ladePlan).mock.calls.length;

		leitung.hoerer.get('lmf-plan')?.();
		await waitFor(() => expect(vi.mocked(ladePlan).mock.calls.length).toBe(vorher + 1));

		seite.unmount();
		expect(leitung.hoerer.has('lmf-plan')).toBe(false);
	});
});
