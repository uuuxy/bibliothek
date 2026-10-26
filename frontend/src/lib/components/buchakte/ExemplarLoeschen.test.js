import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, fireEvent, waitFor } from '@testing-library/svelte';

vi.mock('../../../inventur/lib/store.svelte.js', () => ({ melde: vi.fn() }));
vi.mock('../../stores/bestaetigung.svelte.js', () => ({
	loeschenBestaetigen: vi.fn(async () => true)
}));
vi.mock('../../apiFetch.js', async (original) => ({
	.../** @type {any} */ (await original()),
	apiFetch: vi.fn()
}));

import { apiFetch } from '../../apiFetch.js';
import BuchakteExemplare from './BuchakteExemplare.svelte';
import { authStore } from '../../stores/authStore.svelte.js';
import { melde } from '../../../inventur/lib/store.svelte.js';
import { serverAntwort } from '../../serverAntwort.js';

// Der Kopf der Buchakte nennt „1 von 2 verfügbar". Nach „Exemplar löschen" führt der Reiter
// die zwei Zahlen nach: Ein bestelltes Exemplar, das nie eintraf, stand nie im Bestand, sein
// Löschen ändert ihn nicht.
describe('Buchakte: Exemplar löschen führt den Bestand im Kopf nach', () => {
	const exemplare = () => [
		{ id: 'e1', barcode_id: 'B-1', ist_ausleihbar: true, ist_verfuegbar: true, im_bestand: true },
		{ id: 'e2', barcode_id: 'B-2', ist_ausleihbar: true, ist_verfuegbar: false, im_bestand: true },
		{ id: 'e3', barcode_id: 'B-3', ist_ausleihbar: false, ist_verfuegbar: true, im_bestand: false }
	];

	beforeEach(() => {
		vi.mocked(apiFetch).mockReset();
		vi.mocked(apiFetch).mockResolvedValue(/** @type {any} */ ({ ok: true, status: 200 }));
		authStore.currentUser = /** @type {any} */ ({ rolle: 'admin', permissions: ['*'] });
	});

	/** @param {number} karte Stelle der Karte in der Liste */
	async function loesche(karte) {
		const book = { id: 't-1', title: 'Momo', gesamt: 2, verfuegbar: 1 };
		const screen = render(BuchakteExemplare, {
			props: { exemplare: exemplare(), titel: book, ladeAlles: () => {} }
		});
		await fireEvent.click(screen.getAllByRole('button', { name: 'Exemplar löschen' })[karte]);
		await waitFor(() => expect(apiFetch).toHaveBeenCalledTimes(1));
		await waitFor(() =>
			expect(screen.getAllByRole('button', { name: 'Exemplar löschen' })).toHaveLength(2)
		);
		return book;
	}

	it('lässt den Bestand stehen, wenn ein bestelltes Exemplar gelöscht wird', async () => {
		const book = await loesche(2);
		expect({ gesamt: book.gesamt, verfuegbar: book.verfuegbar }).toEqual({
			gesamt: 2,
			verfuegbar: 1
		});
	});

	it('zieht ein verfügbares Exemplar des Bestands von beiden Zahlen ab', async () => {
		const book = await loesche(0);
		expect({ gesamt: book.gesamt, verfuegbar: book.verfuegbar }).toEqual({
			gesamt: 1,
			verfuegbar: 0
		});
	});
});

// Scheitert beim Löschen mehrerer Exemplare eines, nennt die Meldung den Grund des Servers, und
// das Exemplar bleibt markiert. Vorher stand der Fehlschlag nur in der Konsole.
describe('Buchakte: markierte Exemplare löschen', () => {
	beforeEach(() => {
		vi.mocked(melde).mockReset();
		vi.mocked(apiFetch).mockReset();
		vi.mocked(apiFetch).mockImplementation(async (url) =>
			String(url).endsWith('/e2')
				? serverAntwort(409, { error: 'Das Exemplar ist ausgeliehen.' })
				: serverAntwort(200)
		);
		authStore.currentUser = /** @type {any} */ ({ rolle: 'admin', permissions: ['*'] });
	});

	it('meldet, was nicht gelöscht wurde, und lässt es markiert', async () => {
		const exemplare = ['e1', 'e2', 'e3'].map((id, i) => ({
			id,
			barcode_id: `B-${i + 1}`,
			ist_ausleihbar: true,
			ist_verfuegbar: true,
			im_bestand: true
		}));
		const screen = render(BuchakteExemplare, {
			props: { exemplare, titel: { id: 't-1', title: 'Momo' }, ladeAlles: () => {} }
		});
		for (const nummer of ['B-1', 'B-2', 'B-3']) {
			await fireEvent.click(screen.getByLabelText(`Exemplar ${nummer} auswählen`));
		}
		await fireEvent.click(screen.getByRole('button', { name: 'Löschen' }));

		await waitFor(() =>
			expect(melde).toHaveBeenCalledWith(
				'1 von 3 Exemplaren nicht gelöscht: Das Exemplar ist ausgeliehen.',
				'error'
			)
		);
		expect(melde).toHaveBeenCalledWith('2 Exemplare erfolgreich gelöscht', 'success');
		const bleibt = /** @type {HTMLInputElement} */ (
			screen.getByLabelText('Exemplar B-2 auswählen')
		);
		expect(bleibt.checked).toBe(true);
		expect(screen.queryByLabelText('Exemplar B-1 auswählen')).toBeNull();
		expect(screen.getByText('1 markiert')).toBeTruthy();
	});
});
