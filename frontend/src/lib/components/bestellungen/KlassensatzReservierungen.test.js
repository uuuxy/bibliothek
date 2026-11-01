import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, fireEvent, waitFor } from '@testing-library/svelte';
import KlassensatzReservierungen from './KlassensatzReservierungen.svelte';
import { apiFetch } from '../../apiFetch.js';
import { snackbars } from '../../stores/snackbars.svelte.js';

vi.mock('../../apiFetch.js', () => ({
	apiFetch: vi.fn(),
	FRIST_MAILVERSAND_MS: 1000,
	leseApiFehler: vi.fn(async (/** @type {any} */ res) => JSON.parse(await res.text()).error)
}));
vi.mock('../../stores/snackbars.svelte.js', () => ({ snackbars: { addToast: vi.fn() } }));

// Dieselbe Klasse wie bei den Anliegen: Eine leere Warteschlange heißt „niemand wartet",
// und genau das stand bis zum 12.09.2026 auch dann da, wenn der Abruf gescheitert war.
// Klassensätze sind Unterrichtsvorbereitung — wer hier wartet, wartet auf Bücher für
// eine Stunde mit Termin.
describe('KlassensatzReservierungen', () => {
	beforeEach(() => {
		vi.mocked(apiFetch).mockReset();
	});

	it('unterscheidet „niemand wartet" von „nicht geladen"', async () => {
		vi.mocked(apiFetch).mockResolvedValue(
			/** @type {any} */ ({
				ok: false,
				status: 500,
				text: async () => JSON.stringify({ error: 'Datenbank nicht erreichbar' })
			})
		);
		const { findByText, queryByText } = render(KlassensatzReservierungen);

		expect(await findByText('Datenbank nicht erreichbar')).toBeTruthy();
		expect(queryByText('Keine offenen Klassensatz-Reservierungen.')).toBeNull();
	});

	it('leer bleibt leer', async () => {
		vi.mocked(apiFetch).mockResolvedValue(/** @type {any} */ ({ ok: true, json: async () => [] }));
		const { findByText } = render(KlassensatzReservierungen);
		expect(await findByText('Keine offenen Klassensatz-Reservierungen.')).toBeTruthy();
	});
});

// Beim Abschließen schickt der Server die Bereit-Mail. Scheitert sie, muss die Theke es erfahren,
// sonst wartet die Lehrkraft vergeblich. Die Rückfrage zur Zeile schließt sich auf jedem Weg.
describe('KlassensatzReservierungen: Abschließen', () => {
	const ZEILE = {
		id: 'r1',
		titel_name: 'Natura 7',
		klasse: '07A',
		anzahl: 25,
		verfuegbar: 30,
		erledigt: false,
		erstellt_am: '2026-10-01T08:00:00Z'
	};

	/** @param {any} antwort */
	async function schliesseAb(antwort) {
		vi.mocked(snackbars.addToast).mockReset();
		vi.mocked(apiFetch).mockReset();
		vi.mocked(apiFetch).mockImplementation(
			/** @type {any} */ (
				async (/** @type {string} */ url) =>
					String(url).endsWith('/erledigen')
						? antwort
						: { ok: true, status: 200, json: async () => [ZEILE] }
			)
		);
		const screen = render(KlassensatzReservierungen);
		await fireEvent.click(await screen.findByRole('button', { name: 'Abschließen' }));
		await fireEvent.click(screen.getByRole('button', { name: /Abschließen & Mail senden/ }));
		await waitFor(() => expect(snackbars.addToast).toHaveBeenCalled());
		return screen;
	}

	it('warnt, wenn die Bereit-Mail nicht rausging', async () => {
		await schliesseAb({ ok: true, status: 200, json: async () => ({ mail: 'fehlgeschlagen' }) });
		expect(snackbars.addToast).toHaveBeenCalledWith(
			expect.stringContaining('Bereit-Mail konnte nicht versendet werden'),
			'warning'
		);
	});

	it('meldet Erfolg, wenn die Mail rausging, und schließt die Rückfrage', async () => {
		const screen = await schliesseAb({
			ok: true,
			status: 200,
			json: async () => ({ mail: 'versendet' })
		});
		expect(snackbars.addToast).toHaveBeenCalledWith('Reservierung abgeschlossen.', 'success');
		expect(screen.queryByRole('button', { name: /Abschließen & Mail senden/ })).toBeNull();
	});

	it('schließt die Rückfrage auch, wenn jemand anderes schon abgeschlossen hat', async () => {
		const screen = await schliesseAb({ ok: false, status: 404, json: async () => ({}) });
		expect(snackbars.addToast).toHaveBeenCalledWith(
			'Reservierung war bereits abgeschlossen.',
			'success'
		);
		expect(screen.queryByRole('button', { name: /Abschließen & Mail senden/ })).toBeNull();
	});

	// Während die Antwort zur ersten Zeile unterwegs ist, kann schon die zweite gefragt sein. Die
	// späte Antwort schließt nur die Rückfrage ihrer eigenen Zeile.
	it.each([
		['Erfolg', { ok: true, status: 200, json: async () => ({ mail: 'versendet' }) }],
		['schon abgeschlossen', { ok: false, status: 404, json: async () => ({}) }]
	])('lässt die Rückfrage einer anderen Zeile stehen: %s', async (_, antwort) => {
		/** @type {(wert: any) => void} */
		let antworte = () => {};
		vi.mocked(snackbars.addToast).mockReset();
		vi.mocked(apiFetch).mockReset();
		vi.mocked(apiFetch).mockImplementation(
			/** @type {any} */ (
				(/** @type {string} */ url) =>
					String(url).endsWith('/erledigen')
						? new Promise((fertig) => (antworte = fertig))
						: Promise.resolve({
								ok: true,
								status: 200,
								json: async () => [ZEILE, { ...ZEILE, id: 'r2', klasse: '08B' }]
							})
			)
		);
		const screen = render(KlassensatzReservierungen);
		const [erste] = await screen.findAllByRole('button', { name: 'Abschließen' });
		await fireEvent.click(erste);
		await fireEvent.click(screen.getByRole('button', { name: /Abschließen & Mail senden/ }));

		const buttons = screen.getAllByRole('button', { name: 'Abschließen' });
		await fireEvent.click(buttons[buttons.length - 1]);
		antworte(antwort);

		await waitFor(() => expect(snackbars.addToast).toHaveBeenCalled());
		expect(screen.getByRole('button', { name: /Abschließen & Mail senden/ })).toBeTruthy();
	});
});
