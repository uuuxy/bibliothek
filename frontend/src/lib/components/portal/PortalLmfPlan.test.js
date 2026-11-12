import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
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

/** @type {{ termine: import('../lmfplan/lmfplanDienst.js').LmfTermin[], eingangsjahrgaenge: number[] }} */
const PLAN = {
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

// Über dem Plan steht kein Erklärsatz: Die Überschrift nennt den Plan, der Satz darunter sagt,
// was geschieht (M3, UX writing: „Use headings and subheads to prioritize and group
// information"). „Als PDF" steht in der Zeile der Überschrift statt allein in einer Zeile darüber.
describe('PortalLmfPlan: Kopf des Plans', () => {
	it('zeigt Überschrift und „Als PDF" in einer Zeile, ohne Erklärsatz darüber', async () => {
		render(PortalLmfPlan);
		const titel = await screen.findByRole('heading', {
			name: 'Büchertausch vor den Sommerferien'
		});
		const knopf = screen.getByRole('button', { name: 'Als PDF' });
		expect(knopf.parentElement).toBe(titel.parentElement);
		expect(screen.queryByText(/als nächster dran/)).toBeNull();
	});

	it('zeigt ohne Plan keinen PDF-Knopf', async () => {
		vi.mocked(ladePlan).mockResolvedValue({ termine: [], eingangsjahrgaenge: [] });
		render(PortalLmfPlan);
		await screen.findByText('Der nächste Plan folgt');
		expect(screen.queryByRole('button', { name: 'Als PDF' })).toBeNull();
	});
});

// Die Seite läuft im Lehrerzimmer nebenbei: Eine verschobene Stunde kommt ohne F5, auch wenn der
// Rechner dazwischen geschlafen hat (den Aufbau der Leitung deckt liveEvents.test.js).
describe('PortalLmfPlan: der Plan bleibt aktuell', () => {
	/** @param {string} klasse */
	const planMit = (klasse) =>
		/** @type {any} */ ({ ...PLAN, termine: [{ ...PLAN.termine[0], klassen: [klasse] }] });

	// Der alte Plan stand sonst wieder da, wenn seine Antwort zuletzt kam.
	it('eine ältere Antwort überschreibt die neuere nicht', async () => {
		render(PortalLmfPlan);
		await screen.findByText('09H1');

		/** @type {(plan: any) => void} */
		let aeltereAntwort = () => {};
		vi.mocked(ladePlan).mockReturnValueOnce(new Promise((fertig) => (aeltereAntwort = fertig)));
		leitung.hoerer.get('lmf-plan')?.();
		vi.mocked(ladePlan).mockResolvedValueOnce(planMit('NEU'));
		leitung.hoerer.get('lmf-plan')?.();
		await screen.findByText('NEU');

		aeltereAntwort(planMit('ALT'));
		await new Promise((fertig) => setTimeout(fertig));

		expect(screen.queryByText('ALT')).toBeNull();
		expect(screen.getByText('NEU')).toBeTruthy();
	});

	it('eine späte Absage nimmt den neueren Plan nicht weg', async () => {
		render(PortalLmfPlan);
		await screen.findByText('09H1');

		/** @type {(grund: Error) => void} */
		let aeltereScheitert = () => {};
		vi.mocked(ladePlan).mockReturnValueOnce(new Promise((_, nein) => (aeltereScheitert = nein)));
		leitung.hoerer.get('lmf-plan')?.();
		vi.mocked(ladePlan).mockResolvedValueOnce(planMit('NEU'));
		leitung.hoerer.get('lmf-plan')?.();
		await screen.findByText('NEU');

		aeltereScheitert(new Error('Probe'));
		await new Promise((fertig) => setTimeout(fertig));

		expect(screen.getByText('NEU')).toBeTruthy();
		expect(snackbars.toasts).toHaveLength(0);
	});

	// Die Seite läuft nebenbei im Lehrerzimmer. Der Plan stimmt meistens noch; eine rote Zeile ohne
	// Knopf anstelle des Plans nähme ihn weg, bis jemand neu lädt.
	it('scheitert das Auffrischen, bleibt der Plan stehen, und die Leitung holt nach', async () => {
		render(PortalLmfPlan);
		await screen.findByText('09H1');

		vi.mocked(ladePlan).mockRejectedValueOnce(new Error('Datenbank kurz weg'));
		expect(await leitung.hoerer.get('lmf-plan')?.()).toBe(false);

		await waitFor(() =>
			expect(snackbars.toasts.map((t) => [t.message, t.type])).toEqual([
				['Der Plan wurde nicht aktualisiert: Datenbank kurz weg', 'error']
			])
		);
		expect(screen.getByText('09H1')).toBeTruthy();
		expect(screen.queryByRole('alert')).toBeNull();

		vi.mocked(ladePlan).mockResolvedValueOnce(planMit('NEU'));
		expect(await leitung.hoerer.get('lmf-plan')?.()).toBe(true);
		expect(await screen.findByText('NEU')).toBeTruthy();
	});

	it('scheitert der erste Abruf, nennt die Seite den Grund und bietet den Weg zurück', async () => {
		vi.mocked(ladePlan).mockRejectedValueOnce(new Error('Datenbank kurz weg'));
		render(PortalLmfPlan);

		const meldung = await screen.findByRole('alert');
		expect(meldung.textContent).toContain('LMF-Plan nicht geladen');
		expect(meldung.textContent).toContain('Datenbank kurz weg');
		expect(snackbars.toasts).toHaveLength(0);

		await fireEvent.click(screen.getByRole('button', { name: 'Erneut versuchen' }));
		expect(await screen.findByText('09H1')).toBeTruthy();
		expect(screen.queryByRole('alert')).toBeNull();
	});

	it('holt den Plan neu, wenn sich etwas ändert, und hört nach dem Verlassen auf', async () => {
		vi.mocked(ladePlan).mockResolvedValue(
			/** @type {any} */ ({ termine: [], eingangsjahrgaenge: [] })
		);
		const seite = render(PortalLmfPlan);
		await waitFor(() => expect(ladePlan).toHaveBeenCalled());
		const vorher = vi.mocked(ladePlan).mock.calls.length;

		leitung.hoerer.get('lmf-plan')?.();
		await waitFor(() => expect(vi.mocked(ladePlan).mock.calls).toHaveLength(vorher + 1));

		seite.unmount();
		expect(leitung.hoerer.has('lmf-plan')).toBe(false);
	});
});

// Im Lehrerzimmer läuft die Seite über Nacht. Am Morgen sind die Zeilen von gestern nicht mehr
// hervorgehoben, und der Plan kommt neu, denn nach dem letzten Tag des Büchertauschs gilt die Ausgabe.
describe('PortalLmfPlan: heute', () => {
	afterEach(() => vi.useRealTimers());

	it('hebt die Zeilen von heute hervor; nach Mitternacht wandert die Hervorhebung, und der Plan kommt neu', async () => {
		vi.useFakeTimers({ toFake: ['Date', 'setInterval', 'clearInterval'] });
		vi.setSystemTime(new Date(2027, 5, 28, 23, 59, 30));
		const { container } = render(PortalLmfPlan);
		const heutige = () => container.querySelector('tr[aria-current="date"]');
		await waitFor(() => expect(heutige()?.textContent).toContain('09H1'));
		const vorher = vi.mocked(ladePlan).mock.calls.length;

		await vi.advanceTimersByTimeAsync(60_000);
		await waitFor(() => expect(heutige()).toBeNull());
		expect(vi.mocked(ladePlan).mock.calls).toHaveLength(vorher + 1);
	});
});
