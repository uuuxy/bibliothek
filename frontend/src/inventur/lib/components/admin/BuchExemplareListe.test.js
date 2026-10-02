import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';

vi.mock('../../../../lib/apiFetch.js', () => ({ apiFetch: vi.fn() }));
vi.mock('../../../../lib/stores/bestaetigung.svelte.js', () => ({
	loeschenBestaetigen: vi.fn()
}));
vi.mock('$lib/store.svelte.js', () => ({ showToast: vi.fn() }));

import { apiFetch } from '../../../../lib/apiFetch.js';
import { loeschenBestaetigen } from '../../../../lib/stores/bestaetigung.svelte.js';
import BuchExemplareListe from './BuchExemplareListe.svelte';

/** @param {any} koerper */
const antwort = (koerper) => /** @type {any} */ ({ ok: true, json: async () => koerper });
/** @param {string} nummer @param {any} [zusatz] */
const exemplar = (nummer, zusatz = {}) => ({
	id: `e-${nummer}`,
	barcode_id: `B-${nummer}`,
	ist_ausleihbar: true,
	ist_verfuegbar: true,
	ist_ausgesondert: false,
	im_bestand: true,
	...zusatz
});

beforeEach(() => vi.clearAllMocks());

// GET /api/buecher/titel/{id}/exemplare antwortet mit dem nackten Array (RespondJSON,
// api/copy_admin.go) — so lesen es die Buchakte (useBookAkte) und das Druck-Center
// (labels.svelte.js). Die Maske las einmal `json.data`, bekam undefined und zeigte bei jedem
// Titel „Exemplare (0)".
describe('BuchExemplareListe', () => {
	it('zeigt die Exemplare, die der Server liefert', async () => {
		vi.mocked(apiFetch).mockResolvedValue(
			antwort([exemplar('000101'), exemplar('000102', { ist_ausleihbar: false })])
		);
		const screen = render(BuchExemplareListe, { formular: { id: 't-1', stock: 2 } });

		expect(await screen.findByText('B-000101')).toBeTruthy();
		expect(screen.getByText('B-000102')).toBeTruthy();
		expect(screen.getByRole('heading', { name: 'Exemplare (2)' })).toBeTruthy();
		expect(screen.queryByText(/Nicht im Bestand/)).toBeNull();
		expect(apiFetch).toHaveBeenCalledWith('/api/buecher/titel/t-1/exemplare', expect.anything());
	});

	// Über der Liste stand „Exemplare (5)" neben dem Bestand 2: Ausgesonderte und bestellte
	// Exemplare standen mit darin, beide als „Gesperrt", und „Exemplar löschen" an einem
	// ausgesonderten endete mit „nicht gefunden".
	it('listet nur den Bestand und nennt, was nicht dazu zählt', async () => {
		const draussen = { ist_ausleihbar: false, im_bestand: false };
		vi.mocked(apiFetch).mockResolvedValue(
			antwort([
				exemplar('1'),
				exemplar('2'),
				exemplar('3', { ...draussen, ist_ausgesondert: true }),
				exemplar('4', { ...draussen, ist_ausgesondert: true }),
				exemplar('5', draussen)
			])
		);
		const screen = render(BuchExemplareListe, { formular: { id: 't-1', stock: 2 } });

		expect(await screen.findByRole('heading', { name: 'Exemplare (2)' })).toBeTruthy();
		expect(screen.queryByText('B-3')).toBeNull();
		expect(screen.queryByText('B-5')).toBeNull();
		expect(
			screen.getByText('Nicht im Bestand: 2 ausgesondert, 1 bestellt. Sie stehen in der Buchakte.')
		).toBeTruthy();
		expect(screen.getAllByRole('button', { name: 'Exemplar löschen' })).toHaveLength(2);
	});

	it('ein gelöschtes Exemplar verlässt die Liste und zählt als ausgesondert', async () => {
		vi.mocked(apiFetch).mockImplementation(async (url, optionen) => {
			if (optionen?.method === 'DELETE') return antwort({});
			if (String(url).endsWith('/exemplare')) return antwort([exemplar('1'), exemplar('2')]);
			return antwort({ stock: 1 });
		});
		vi.mocked(loeschenBestaetigen).mockResolvedValue(true);
		const formular = { id: 't-1', stock: 2, stockGesehen: 2 };
		const screen = render(BuchExemplareListe, { formular });
		await screen.findByText('B-1');

		await fireEvent.click(screen.getAllByRole('button', { name: 'Exemplar löschen' })[0]);

		expect(await screen.findByRole('heading', { name: 'Exemplare (1)' })).toBeTruthy();
		expect(screen.getByText(/Nicht im Bestand: 1 ausgesondert\./)).toBeTruthy();
		expect(formular.stock, 'die Zahl im Feld kommt vom Server').toBe(1);
	});

	it('ohne Exemplar im Bestand sagt die Liste das', async () => {
		vi.mocked(apiFetch).mockResolvedValue(
			antwort([exemplar('1', { im_bestand: false, ist_ausgesondert: true })])
		);
		const screen = render(BuchExemplareListe, { formular: { id: 't-1', stock: 0 } });

		expect(await screen.findByText('Kein Exemplar im Bestand.')).toBeTruthy();
		expect(screen.getByText(/Nicht im Bestand: 1 ausgesondert\./)).toBeTruthy();
	});
});
