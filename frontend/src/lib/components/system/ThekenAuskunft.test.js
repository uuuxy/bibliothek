import { describe, it, expect, vi } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';

vi.mock('../../apiFetch.js', () => ({
	apiFetch: vi.fn(),
	leseApiFehler: vi.fn()
}));

import { apiFetch } from '../../apiFetch.js';
import ThekenAuskunft from './ThekenAuskunft.svelte';

// Ein Handscanner mit amerikanischer Tastatur tippt an einem deutschen Rechner den Bindestrich
// als ß (FACHKONZEPT §1); die Auskunft schlägt die Nummer nach, die die Theke nachschlägt.
describe('Theken-Auskunft', () => {
	it('liest in einem gescannten Etikett ein ß hinter der Vorsilbe als Bindestrich', async () => {
		vi.mocked(apiFetch).mockResolvedValue(
			/** @type {any} */ ({
				ok: true,
				json: async () => ({ barcode: 'B-00123', exemplare: [], ereignisse: [] })
			})
		);
		const screen = render(ThekenAuskunft);

		const feld = screen.getByLabelText('Buch-Barcode');
		await fireEvent.input(feld, { target: { value: 'Bß00123' } });
		await fireEvent.submit(/** @type {HTMLFormElement} */ (feld.closest('form')));

		expect(vi.mocked(apiFetch)).toHaveBeenCalledWith('/api/audit/tresen-auskunft?barcode=B-00123');
	});
});
