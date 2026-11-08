import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, fireEvent, waitFor } from '@testing-library/svelte';

// leseApiFehler bleibt echt (ARCHITEKTUR.md 8.14).
vi.mock('../../apiFetch.js', async (importOriginal) => ({
	.../** @type {any} */ (await importOriginal()),
	apiFetch: vi.fn()
}));

// Der Druck selbst gehört dem Baustein der Leserdatei; hier zählt, wen der Reiter ihm gibt.
const { drucke } = vi.hoisted(() => ({ drucke: vi.fn() }));
vi.mock('./ausweisdruck.svelte.js', async (importOriginal) => ({
	.../** @type {any} */ (await importOriginal()),
	erzeugeAusweisdruck: () => ({ etikettModus: true, maxPosition: 21, startPosition: 1, drucke })
}));

import { apiFetch } from '../../apiFetch.js';
import { serverAntwort } from '../../serverAntwort.js';
import AusweiseJeKlasse from './AusweiseJeKlasse.svelte';

const KLASSE_05A = [
	{
		id: 's1',
		vorname: 'Ada',
		nachname: 'A',
		klasse: '05A',
		barcode_id: 'A-1',
		ausweis_gueltig_bis: '2031-07-31'
	},
	{
		id: 's2',
		vorname: 'Ben',
		nachname: 'B',
		klasse: '05A',
		barcode_id: 'A-2',
		ausweis_gueltig_bis: '2031-07-31'
	},
	{
		id: 's3',
		vorname: 'Cem',
		nachname: 'C',
		klasse: '05A',
		barcode_id: '',
		ausweis_gueltig_bis: null
	}
];

/** Antwortet nach der Adresse: Klassen, dann die Schüler der gewählten Klasse. */
function server(schueler = KLASSE_05A) {
	vi.mocked(apiFetch).mockImplementation(async (/** @type {any} */ adresse) =>
		adresse === '/api/klassen' ? serverAntwort(200, ['05A', '05B']) : serverAntwort(200, schueler)
	);
}

/** @param {import('@testing-library/svelte').RenderResult<any>} k @param {string} klasse */
async function waehle(k, klasse) {
	await fireEvent.click(await k.findByRole('combobox', { name: 'Klasse' }));
	await fireEvent.click(await k.findByRole('option', { name: klasse }));
}

describe('AusweiseJeKlasse: der Reiter druckt die Ausweise einer ganzen Klasse', () => {
	beforeEach(() => vi.clearAllMocks());

	it('lädt die Schüler der gewählten Klasse und nennt, wem die Ausweisnummer fehlt', async () => {
		server();
		const k = render(AusweiseJeKlasse);
		await waehle(k, '05A');

		await k.findByText('3 Schüler, 1 ohne Ausweisnummer');
		expect(apiFetch).toHaveBeenCalledWith('/api/schueler?klasse=05A');
	});

	it('gibt dem Druck alle Schüler der Klasse; wer keine Nummer hat, sortiert der Baustein aus', async () => {
		server();
		const k = render(AusweiseJeKlasse);
		await waehle(k, '05A');

		await fireEvent.click(await k.findByRole('button', { name: 'Etiketten drucken' }));
		expect(drucke).toHaveBeenCalledWith(KLASSE_05A);
	});

	it('eine leere Klasse: ein Satz statt eines Knopfs, der nichts druckt', async () => {
		server([]);
		const k = render(AusweiseJeKlasse);
		await waehle(k, '05B');

		await k.findByText('In dieser Klasse steht kein Schüler.');
		expect(
			/** @type {HTMLButtonElement} */ (k.getByRole('button', { name: 'Etiketten drucken' }))
				.disabled
		).toBe(true);
	});

	it('kommen die Schüler nicht, nennt er den Grund', async () => {
		vi.mocked(apiFetch).mockImplementation(async (/** @type {any} */ adresse) =>
			adresse === '/api/klassen'
				? serverAntwort(200, ['05A'])
				: serverAntwort(503, { error: 'Die Datenbank antwortet nicht' })
		);
		const k = render(AusweiseJeKlasse);
		await waehle(k, '05A');

		const kasten = await k.findByRole('alert');
		expect(kasten.textContent).toContain('Schüler nicht geladen');
		expect(kasten.textContent).toContain('Die Datenbank antwortet nicht');
	});

	// Wer schnell die Klasse wechselt, druckte sonst die Karten der vorigen Klasse.
	it('kommt die Antwort der vorigen Klasse zu spät, bleibt die zuletzt gewählte', async () => {
		/** @type {(antwort: any) => void} */
		let liefere = () => {};
		vi.mocked(apiFetch).mockImplementation(async (/** @type {any} */ adresse) => {
			if (adresse === '/api/klassen') return serverAntwort(200, ['05A', '05B']);
			if (adresse === '/api/schueler?klasse=05A')
				return new Promise((fertig) => (liefere = fertig));
			return serverAntwort(200, [KLASSE_05A[0]]);
		});
		const k = render(AusweiseJeKlasse);
		await waehle(k, '05A');
		await waehle(k, '05B');
		await k.findByText('1 Schüler');

		liefere(serverAntwort(200, KLASSE_05A));
		await new Promise((r) => setTimeout(r, 0));

		expect(k.getByText('1 Schüler')).toBeTruthy();
		await fireEvent.click(k.getByRole('button', { name: 'Etiketten drucken' }));
		expect(drucke).toHaveBeenCalledWith([KLASSE_05A[0]]);
	});
});

// Die Klasse wird gewählt, nicht getippt. Kam die Klassenliste nicht, blieb die Auswahl leer,
// ohne ein Wort, in Chromium, Firefox und WebKit (OFFEN.md 5.106).
describe('AusweiseJeKlasse: die Klassenliste kommt nicht', () => {
	beforeEach(() => vi.clearAllMocks());

	it.each([
		[
			'bei einer Ablehnung',
			async () => serverAntwort(503, { error: 'Die Datenbank antwortet nicht' }),
			'Die Datenbank antwortet nicht'
		],
		[
			'ohne Antwort',
			() => Promise.reject(new Error('Keine Verbindung zum Server.')),
			'Keine Verbindung zum Server.'
		]
	])('nennt %s den Grund und lädt auf Wunsch neu', async (_, antwort, grund) => {
		vi.mocked(apiFetch).mockImplementationOnce(/** @type {any} */ (antwort));
		const k = render(AusweiseJeKlasse);

		const kasten = await k.findByRole('alert');
		expect(kasten.textContent).toContain('Klassen nicht geladen');
		expect(kasten.textContent).toContain(grund);

		vi.mocked(apiFetch).mockResolvedValueOnce(serverAntwort(200, ['05A', '05B']));
		await fireEvent.click(k.getByRole('button', { name: 'Erneut versuchen' }));
		await waitFor(() => expect(k.queryByRole('alert')).toBeNull());
		expect(k.getByRole('combobox', { name: 'Klasse' })).toBeTruthy();
	});
});
