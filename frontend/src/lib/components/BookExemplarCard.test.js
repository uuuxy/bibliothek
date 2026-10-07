import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, fireEvent, waitFor } from '@testing-library/svelte';

vi.mock('../apiFetch.js', async (original) => ({
	...(await original()),
	apiFetch: vi.fn(),
	apiClient: { put: vi.fn() }
}));

import { apiClient } from '../apiFetch.js';
import BookExemplarCard from './BookExemplarCard.svelte';

// Was nicht zum Bestand zählt, heißt nicht „Gesperrt". Ein ausgesondertes Exemplar hat keinen
// Papierkorb: Die Tür weist es als schon ausgebucht ab.
describe('Exemplarkarte: Zustand', () => {
	/** @param {Record<string, any>} ex */
	const karte = (ex) =>
		render(BookExemplarCard, {
			ex: { id: 'e1', barcode_id: 'B-0041873', ist_verfuegbar: true, ...ex },
			selected: false,
			darfBearbeiten: true,
			onToggleSelect: () => {},
			onDelete: () => {}
		});
	/** @param {ReturnType<typeof karte>} k */
	const wort = (k) => k.container.querySelector('[data-chip]')?.textContent?.trim();

	it('nennt ein bestelltes Exemplar bestellt und bietet das Löschen an', () => {
		const k = karte({ ist_ausleihbar: false, ist_ausgesondert: false, im_bestand: false });

		expect(wort(k)).toBe('Bestellt');
		expect(k.queryByRole('button', { name: 'Exemplar löschen' })).not.toBeNull();
	});

	it('nennt ein ausgesondertes Exemplar ausgesondert und bietet kein Löschen an', () => {
		const k = karte({ ist_ausleihbar: false, ist_ausgesondert: true, im_bestand: false });

		expect(wort(k)).toBe('Ausgesondert');
		expect(k.queryByRole('button', { name: 'Exemplar löschen' })).toBeNull();
		expect(k.queryByRole('button', { name: 'Status ändern' })).not.toBeNull();
	});

	it('nennt ein nicht ausleihbares Exemplar im Bestand gesperrt', () => {
		const k = karte({ ist_ausleihbar: false, ist_ausgesondert: false, im_bestand: true });

		expect(wort(k)).toBe('Gesperrt');
	});
});

// Lehnt der Server einen Barcode ab, steht sein Grund ungekürzt am Feld und ist dessen
// Beschreibung: Ein Screenreader liest ihn mit dem Feld vor.
describe('Exemplarkarte: abgelehnter Barcode', () => {
	const grund = 'dieser Barcode wird bereits von einem anderen Exemplar verwendet';

	const karte = () =>
		render(BookExemplarCard, {
			ex: { id: 'e1', barcode_id: 'AUTO-000123', ist_ausleihbar: true, ist_verfuegbar: true },
			selected: false,
			darfBearbeiten: true,
			onToggleSelect: () => {},
			onDelete: () => {}
		});

	/** @param {ReturnType<typeof karte>} k */
	async function trageEin(k) {
		await fireEvent.click(k.getByRole('button', { name: 'Barcode scannen' }));
		const feld = /** @type {HTMLInputElement} */ (k.getByLabelText('Barcode des Exemplars'));
		await fireEvent.input(feld, { target: { value: 'B-0041873' } });
		await fireEvent.keyDown(feld, { key: 'Enter' });
		return feld;
	}

	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('nennt den Grund des Servers als Beschreibung des Felds', async () => {
		vi.mocked(apiClient.put).mockResolvedValue(
			/** @type {any} */ ({ ok: false, json: async () => ({ error: grund }) })
		);
		const feld = await trageEin(karte());

		await waitFor(() => expect(feld.getAttribute('aria-invalid')).toBe('true'));
		const beschreibung = document.getElementById(feld.getAttribute('aria-describedby') ?? '');
		expect(beschreibung?.textContent).toBe(grund);
		expect(beschreibung?.hasAttribute('title'), 'der Grund steht nur in einer Sprechblase').toBe(
			false
		);
	});

	// Die Gegenprobe: Ein angenommener Barcode schließt das Feld und hinterlässt keinen Fehler.
	it('zeigt nach einem angenommenen Barcode die Nummer und keinen Fehler', async () => {
		vi.mocked(apiClient.put).mockResolvedValue(/** @type {any} */ ({ ok: true }));
		const k = karte();
		await trageEin(k);

		await waitFor(() => expect(k.queryByLabelText('Barcode des Exemplars')).toBeNull());
		expect(k.container.textContent).toContain('B-0041873');
		expect(k.container.textContent).not.toContain(grund);
	});
});
