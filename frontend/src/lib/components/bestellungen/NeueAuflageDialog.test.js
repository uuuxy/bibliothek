import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';

vi.mock('../../apiFetch.js', () => ({
	apiClient: { post: vi.fn(), put: vi.fn() },
	extractApiError: vi.fn(async (/** @type {any} */ res) => `Fehler ${res.status}`)
}));
vi.mock('../../stores/toastStore.svelte.js', () => ({ toastStore: { addToast: vi.fn() } }));

import { apiClient } from '../../apiFetch.js';
import NeueAuflageDialog from './NeueAuflageDialog.svelte';

// „Neue Auflage bestellen" (docs/OFFEN.md 4.18, Stufe 4): Die ISBN holt den Titel über
// POST /api/buecher/aus-isbn, der Dialog zeigt ihn als Vorschlag, und erst die Bestätigung
// fasst zusammen (POST …/neue-auflage) und reicht ihn an den Warenkorb weiter. Ein eben aus
// der DNB angelegter Titel ist noch kein Lernmittel — vorher wird er eins, sonst wiese der
// Server das Zusammenfassen ab.

const zeile = { id: 'alt', titel: 'Lambacher Schweizer 7', auflagen: undefined };
const antwort = (/** @type {any} */ body, ok = true, status = 200) => ({
	ok,
	status,
	json: async () => body
});
const neu = {
	titel_id: 'neu',
	titel: 'Lambacher Schweizer 7 (Neubearbeitung)',
	isbn: '9783120000026',
	verlag: 'Klett',
	ist_lernmittel: false
};

async function bisZumVorschlag(/** @type {any} */ gefunden = neu) {
	vi.mocked(apiClient.post).mockResolvedValueOnce(/** @type {any} */ (antwort(gefunden)));
	const onbestellt = vi.fn();
	const screen = render(NeueAuflageDialog, { zeile, onschliessen: vi.fn(), onbestellt });
	await fireEvent.input(screen.getByLabelText('ISBN der neuen Auflage'), {
		target: { value: '9783120000026' }
	});
	await fireEvent.click(screen.getByRole('button', { name: 'Suchen' }));
	await screen.findByText(gefunden.titel);
	return { screen, onbestellt };
}

beforeEach(() => {
	vi.mocked(apiClient.post).mockReset();
	vi.mocked(apiClient.put).mockReset();
});

describe('NeueAuflageDialog', () => {
	it('sperrt „Suchen" ohne ISBN, „Abbrechen" nie', () => {
		const screen = render(NeueAuflageDialog, { zeile, onschliessen: vi.fn(), onbestellt: vi.fn() });
		expect(screen.getByRole('button', { name: 'Suchen' }).hasAttribute('disabled')).toBe(true);
		expect(screen.getByRole('button', { name: 'Abbrechen' }).hasAttribute('disabled')).toBe(false);
	});

	it('macht den Titel zum Lernmittel, fasst zusammen und reicht die neue Auflage weiter', async () => {
		const { screen, onbestellt } = await bisZumVorschlag();
		expect(apiClient.post).toHaveBeenCalledWith('/api/buecher/aus-isbn', { isbn: '9783120000026' });
		expect(screen.getByText(/als Lernmittel geführt/)).toBeTruthy();

		vi.mocked(apiClient.put).mockResolvedValueOnce(/** @type {any} */ (antwort({})));
		vi.mocked(apiClient.post).mockResolvedValueOnce(/** @type {any} */ (antwort({ auflagen: [] })));
		await fireEvent.click(screen.getByRole('button', { name: 'Zuordnen und bestellen' }));
		await vi.waitFor(() => expect(onbestellt).toHaveBeenCalled());

		expect(apiClient.put).toHaveBeenCalledWith('/api/buecher/titel/neu/lernmittel', {
			ist_lernmittel: true
		});
		expect(apiClient.post).toHaveBeenLastCalledWith('/api/buecher/titel/alt/neue-auflage', {
			titel_id: 'neu'
		});
		expect(onbestellt.mock.calls[0][0]).toMatchObject({ id: 'neu', ist_lernmittel: true });
	});

	it('schlägt die Auflage aus der Zeile nicht als neue vor', async () => {
		const { screen } = await bisZumVorschlag({ ...neu, titel_id: 'alt', ist_lernmittel: true });
		expect(screen.getByText(/gehört schon zu diesem Buch/)).toBeTruthy();
		expect(
			screen.getByRole('button', { name: 'Zuordnen und bestellen' }).hasAttribute('disabled')
		).toBe(true);
	});

	it('meldet eine Abweisung im Dialog und legt nichts in den Warenkorb', async () => {
		const { screen, onbestellt } = await bisZumVorschlag({ ...neu, ist_lernmittel: true });
		vi.mocked(apiClient.post).mockResolvedValueOnce(
			/** @type {any} */ (antwort({ error: 'nein' }, false, 400))
		);
		await fireEvent.click(screen.getByRole('button', { name: 'Zuordnen und bestellen' }));
		await vi.waitFor(() => expect(screen.getByRole('alert').textContent).toContain('Fehler 400'));
		expect(apiClient.put).not.toHaveBeenCalled();
		expect(onbestellt).not.toHaveBeenCalled();
	});
});
