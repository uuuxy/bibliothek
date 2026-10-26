import { describe, it, expect, vi } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';

vi.mock('../../apiFetch.js', () => ({ apiGet: vi.fn(), apiPost: vi.fn() }));

import { apiGet } from '../../apiFetch.js';
import BescheidDialog from './BescheidDialog.svelte';

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
