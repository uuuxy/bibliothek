import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';

// Die Bestätigung der Zubehör-Liste schickt den Scan erneut, mit confirmed_checklist — und
// mit override_block, wenn der Scan schon ein Übergehen war (seit dem 24.09.2026 lässt sich
// eine offene Forderung auch am Gerät übergehen). Ohne das hielte die Sperre den zweiten
// Versand wieder auf.
vi.mock('../apiFetch.js', () => ({
	apiFetch: vi.fn(),
	apiClient: { post: vi.fn(), patch: vi.fn() },
	registriereSitzungAbgelaufenHandler: vi.fn()
}));
vi.mock('../audio.js', () => ({
	playSoundSuccess: vi.fn(),
	playSoundError: vi.fn(),
	playSuccessBeep: vi.fn(),
	playErrorBeep: vi.fn()
}));
vi.mock('../../inventur/lib/store.svelte.js', () => ({ showToast: vi.fn(), appState: {} }));

import { omniboxStore } from '../stores/omnibox.svelte.js';
import { apiClient } from '../apiFetch.js';
import OmniboxChecklistDialog from './OmniboxChecklistDialog.svelte';

// Die Zeitgeber der Theke stoppen (stores/omniboxZeitgeber.test.js).
afterEach(() => omniboxStore.stoppeZeitgeber());

describe('OmniboxChecklistDialog', () => {
	beforeEach(() => {
		vi.clearAllMocks();
		omniboxStore.activeStudent = { id: 'schueler-7', vorname: 'Anna', nachname: 'Müller' };
		vi.mocked(apiClient.post).mockResolvedValue(
			/** @type {any} */ ({ ok: true, json: async () => ({ type: 'info' }) })
		);
	});

	for (const overrideBlock of [true, false]) {
		it(`schickt override_block=${overrideBlock} mit der Bestätigung`, async () => {
			omniboxStore.checklistAnfrage = {
				query: 'G-1',
				geraet: { modellname: 'Tablet', barcode_id: 'G-1', zubehoer: 'Kabel' },
				overrideBlock
			};
			const screen = render(OmniboxChecklistDialog, { onReload: () => {} });
			await fireEvent.click(screen.getByRole('button', { name: /Alles vollständig/ }));

			await vi.waitFor(() => expect(apiClient.post).toHaveBeenCalled());
			expect(apiClient.post).toHaveBeenCalledWith(
				'/api/action',
				expect.objectContaining({
					query: 'G-1',
					confirmed_checklist: true,
					override_block: overrideBlock
				})
			);
		});
	}
});
