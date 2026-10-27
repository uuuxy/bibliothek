import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, fireEvent, waitFor } from '@testing-library/svelte';

vi.mock('../../stores/snackbars.svelte.js', () => ({ snackbars: { addToast: vi.fn() } }));
// leseApiFehler bleibt echt (ARCHITEKTUR.md 8.14).
vi.mock('../../apiFetch.js', async (importOriginal) => ({
	.../** @type {any} */ (await importOriginal()),
	apiGet: vi.fn(),
	apiPost: vi.fn()
}));

import { apiGet } from '../../apiFetch.js';
import AltbestandDialog from './AltbestandDialog.svelte';

const ENTWARNUNG = /kein Exemplar mehr offen/;

/** Öffnet den Dialog und setzt den Stichtag, wie die Person ihn wählt. @param {string} tag */
async function waehleTag(/** @type {any} */ k, tag) {
	const feld = /** @type {HTMLInputElement} */ (k.getByLabelText('Beklebt bis'));
	await fireEvent.input(feld, { target: { value: tag } });
	await fireEvent.change(feld);
}

const dialog = () => render(AltbestandDialog, { open: true, onclose: vi.fn(), onfertig: vi.fn() });
const knopf = (/** @type {any} */ k) =>
	/** @type {HTMLButtonElement} */ (k.getByRole('button', { name: 'Als erledigt vermerken' }));

// Der Dialog vermerkt alles bis zum Stichtag als beklebt, unumkehrbar. Scheiterte die Zählung
// davor, stand „Bis zu diesem Tag steht kein Exemplar mehr offen", in Chromium, Firefox und WebKit
// (OFFEN.md 5.106).
describe('AltbestandDialog: die Zahl vor dem Vermerk', () => {
	beforeEach(() => vi.clearAllMocks());

	it('nennt den Grund, wenn die Zählung scheitert, statt „kein Exemplar mehr offen"', async () => {
		vi.mocked(apiGet).mockImplementationOnce(() =>
			Promise.reject(new Error('Keine Verbindung zum Server.'))
		);
		const k = dialog();

		await waehleTag(k, '2020-01-01');

		await waitFor(() =>
			expect(k.getByText(/Nicht gezählt: Keine Verbindung zum Server\./)).toBeTruthy()
		);
		expect(k.queryByText(ENTWARNUNG)).toBeNull();
		expect(knopf(k).disabled).toBe(true);
	});

	it('sagt während der Zählung, dass gezählt wird, und lässt nichts vermerken', async () => {
		/** @type {(antwort: any) => void} */
		let antworte = () => {};
		vi.mocked(apiGet).mockImplementationOnce(() => new Promise((fertig) => (antworte = fertig)));
		const k = dialog();

		await waehleTag(k, '2020-01-01');

		expect(k.getByText(/Wird gezählt/)).toBeTruthy();
		expect(k.queryByText(ENTWARNUNG)).toBeNull();
		expect(knopf(k).disabled).toBe(true);
		antworte({ anzahl: 12 });
		await waitFor(() => expect(knopf(k).disabled).toBe(false));
	});

	// Wer den Tag schnell wechselt: Die Zahl muss zum Tag gehören, bis zu dem der Knopf vermerkt.
	it('zeigt die Zahl zum zuletzt gewählten Tag, auch wenn eine ältere Antwort zuletzt kommt', async () => {
		/** @type {((antwort: any) => void)[]} */
		const antworten = [];
		vi.mocked(apiGet).mockImplementation(() => new Promise((fertig) => antworten.push(fertig)));
		const k = dialog();

		await waehleTag(k, '2020-01-01');
		await waehleTag(k, '2024-01-01');
		antworten[1]({ anzahl: 900 });
		antworten[0]({ anzahl: 12 });

		await waitFor(() => expect(k.getByText(/900 Exemplare/)).toBeTruthy());
		expect(k.queryByText(/12 Exemplare/)).toBeNull();
	});
});
