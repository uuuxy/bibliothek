import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';

vi.mock('../../apiFetch.js', async (importOriginal) => ({
	.../** @type {any} */ (await importOriginal()),
	apiFetch: vi.fn()
}));

import { apiFetch } from '../../apiFetch.js';
import Bestandsbuch from './Bestandsbuch.svelte';

const antwort = /** @type {any} */ ({
	ok: true,
	json: async () => ({ von: '2026-03-16', bis: '2026-09-15', abschnitte: [] })
});

/** Die erste Antwort kommt erst, wenn der Test sie freigibt; die Rückgabe gibt sie frei. */
function zeigeMitSpaeterAntwort() {
	/** @type {(wert: any) => void} */
	let freigeben = () => {};
	vi.mocked(apiFetch).mockReturnValue(new Promise((r) => (freigeben = r)));
	const screen = render(Bestandsbuch, {
		pfad: 'zugangsbuch',
		buchname: 'Zugangsbuch',
		wortSingular: 'Zugang',
		spalten: [{ kopf: 'Titel', feld: 'titel' }]
	});
	const feld = (/** @type {string} */ name) =>
		/** @type {HTMLInputElement} */ (screen.getByLabelText(name));
	return { screen, von: feld('Von'), bis: feld('Bis'), freigeben: () => freigeben(antwort) };
}

beforeEach(() => {
	vi.clearAllMocks();
	vi.useFakeTimers({ toFake: ['Date'] });
	vi.setSystemTime(new Date(2026, 7, 10, 9, 0));
});
afterEach(() => vi.useRealTimers());

// Wer den Zeitraum eingibt, während das Buch noch lädt, verliert die Eingabe nicht an die
// Vorgabe des Servers; das Feld, das niemand angefasst hat, zeigt den geladenen Zeitraum.
describe('Bestandsbuch: Zeitraum eingeben, während das Buch lädt', () => {
	it('lässt ein angefangenes Datum stehen und füllt das andere Feld', async () => {
		const { screen, von, bis, freigeben } = zeigeMitSpaeterAntwort();

		await fireEvent.input(von, { target: { value: '01.0' } });
		freigeben();

		await screen.findByText('Kein Zugang in diesem Zeitraum');
		expect(von.value).toBe('01.0');
		expect(bis.value).toBe('15.09.2026');

		await fireEvent.input(von, { target: { value: '01.04.2026' } });
		await fireEvent.click(screen.getByRole('button', { name: 'Anzeigen' }));
		expect(apiFetch).toHaveBeenLastCalledWith(
			'/api/bestand/zugangsbuch?von=2026-04-01&bis=2026-09-15'
		);
	});

	it('lässt ein im Kalender gewähltes Datum stehen', async () => {
		const { screen, von, bis, freigeben } = zeigeMitSpaeterAntwort();

		await fireEvent.click(screen.getAllByRole('button', { name: 'Kalender öffnen' })[1]);
		await fireEvent.click(screen.getByRole('button', { name: 'Montag, 3. August 2026' }));
		await fireEvent.click(screen.getByRole('button', { name: 'OK' }));
		freigeben();

		await screen.findByText('Kein Zugang in diesem Zeitraum');
		expect(von.value).toBe('16.03.2026');
		expect(bis.value).toBe('03.08.2026');
	});
});
