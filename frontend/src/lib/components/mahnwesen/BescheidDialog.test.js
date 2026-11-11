import { describe, it, expect, vi } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';

vi.mock('../../apiFetch.js', () => ({ apiGet: vi.fn(), apiPost: vi.fn() }));

import { apiGet, apiPost } from '../../apiFetch.js';
import BescheidDialog from './BescheidDialog.svelte';
import { snackbars } from '../../stores/snackbars.svelte.js';
import { PDF_BLOCKIERT } from '../../utils/pdfImFenster.js';

// Kommt der Vorschlag nicht, sagte der Dialog „Für dieses Kind gibt es weder ein überfälliges
// Buch noch eine offene Forderung": eine Entwarnung, während nur der Abruf scheiterte. Den Grund
// zeigte apiGet als Meldung, die nach Sekunden verschwindet.
describe('BescheidDialog: der Vorschlag kommt nicht', () => {
	it('nennt den Grund statt der Entwarnung und lädt auf Wunsch neu', async () => {
		vi.mocked(apiGet).mockRejectedValue(new Error('Die Datenbank antwortet nicht'));
		const screen = render(BescheidDialog, {
			schuelerId: 's1',
			onclose: vi.fn(),
			onErstellt: vi.fn()
		});

		await screen.findByText('Bücher und Forderungen nicht geladen');
		expect(screen.getByText('Die Datenbank antwortet nicht')).toBeTruthy();
		expect(screen.queryByText(/weder ein überfälliges Buch/)).toBeNull();

		vi.mocked(apiGet).mockResolvedValue({ fehlende_angaben: [], positionen: [], ausleihen: [] });
		await fireEvent.click(screen.getByRole('button', { name: 'Erneut versuchen' }));
		await screen.findByText(/weder ein überfälliges Buch/);
		expect(vi.mocked(apiGet)).toHaveBeenCalledTimes(2);
	});
});

// Verweigerte der Browser den Tab, geschah auf „Brief öffnen" nichts, und niemand erfuhr, warum.
describe('BescheidDialog: Brief öffnen', () => {
	it('meldet, wenn der Browser den Tab verweigert', async () => {
		vi.mocked(apiGet).mockResolvedValue({
			frist_bis: '2026-12-01',
			fehlende_angaben: [],
			positionen: [
				{ schadensfall_id: 'f1', art: 'beschaedigt', titel: 'Mathe 7', betrag: 12, topf: 'land' }
			],
			ausleihen: []
		});
		vi.mocked(apiPost).mockResolvedValue({ id: 'b1', referenznummer: '5830 2026 1234 0001' });
		const meldung = vi.spyOn(snackbars, 'addToast');
		const screen = render(BescheidDialog, {
			schuelerId: 's1',
			onclose: vi.fn(),
			onErstellt: vi.fn()
		});

		await screen.findByText('Mathe 7');
		await fireEvent.click(screen.getByRole('button', { name: 'Bescheid erstellen' }));
		await vi.waitFor(() =>
			expect(meldung).toHaveBeenCalledWith(
				expect.stringContaining('erstellt'),
				'success',
				expect.anything()
			)
		);
		const aktion = /** @type {any} */ (meldung.mock.calls.find(([, art]) => art === 'success'))[2];

		vi.stubGlobal(
			'open',
			vi.fn(() => null)
		);
		aktion.onClick();
		expect(meldung).toHaveBeenLastCalledWith(PDF_BLOCKIERT, 'warning');
		vi.unstubAllGlobals();
	});
});

// Ein leeres Betragsfeld zählte als 0 €. Jetzt ist das Feld eines gewählten Buchs ohne Betrag
// als Fehler markiert (M3: Fehlerfarbe, Symbol, Fehlertext statt Hilfetext), und „Bescheid
// erstellen" bleibt gesperrt.
describe('BescheidDialog: Betrag', () => {
	it('markiert ein gewähltes Buch ohne Betrag und sperrt den Brief', async () => {
		vi.mocked(apiGet).mockResolvedValue({
			frist_bis: '2026-12-01',
			fehlende_angaben: [],
			positionen: [],
			ausleihen: [
				{
					ausleihe_id: 'a1',
					titel: 'Physik 8',
					betrag: 24.9,
					topf: 'land',
					faellig_seit: '2026-08-15'
				}
			]
		});
		const screen = render(BescheidDialog, {
			schuelerId: 's1',
			onclose: vi.fn(),
			onErstellt: vi.fn()
		});
		const feld = /** @type {HTMLInputElement} */ (
			await screen.findByLabelText('Betrag für Physik 8')
		);
		const knopf = /** @type {HTMLButtonElement} */ (
			screen.getByRole('button', { name: 'Bescheid erstellen' })
		);
		expect(feld.getAttribute('aria-invalid')).toBeNull();
		expect(knopf.disabled).toBe(false);

		await fireEvent.input(feld, { target: { value: '' } });
		expect(feld.getAttribute('aria-invalid')).toBe('true');
		expect(screen.getByText('Betrag eintragen')).toBeTruthy();
		expect(knopf.disabled).toBe(true);

		await fireEvent.input(feld, { target: { value: '7.5' } });
		expect(feld.getAttribute('aria-invalid')).toBeNull();
		expect(screen.queryByText('Betrag eintragen')).toBeNull();
		expect(knopf.disabled).toBe(false);
	});
});
