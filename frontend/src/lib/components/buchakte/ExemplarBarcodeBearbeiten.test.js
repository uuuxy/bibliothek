import { describe, it, expect, vi } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';

vi.mock('../../apiFetch.js', () => ({
	apiFetch: vi.fn(),
	apiClient: { put: vi.fn() },
	leseApiFehler: vi.fn()
}));

import { apiClient } from '../../apiFetch.js';
import ExemplarBarcodeBearbeiten from './ExemplarBarcodeBearbeiten.svelte';

// Ein Handscanner mit amerikanischer Tastatur tippt an einem deutschen Rechner den Bindestrich
// als ß (FACHKONZEPT §1): Gespeichert wird, was die Theke beim Scan nachschlägt.
describe('Barcode eines Exemplars', () => {
	it('speichert eine gescannte Nummer so, wie die Theke sie liest', async () => {
		vi.mocked(apiClient.put).mockResolvedValue(/** @type {any} */ ({ ok: true }));
		const ex = { id: 'e1', barcode_id: '4711' };
		const onDone = vi.fn();
		const screen = render(ExemplarBarcodeBearbeiten, { ex, vorgabe: '', onDone });

		await fireEvent.input(screen.getByLabelText('Barcode des Exemplars'), {
			target: { value: 'Bß00123' }
		});
		await fireEvent.click(screen.getByText('Speichern'));

		expect(vi.mocked(apiClient.put)).toHaveBeenCalledWith('/api/buecher/exemplare/e1/barcode', {
			barcode: 'B-00123'
		});
		await vi.waitFor(() => expect(onDone).toHaveBeenCalled());
		expect(ex.barcode_id).toBe('B-00123');
	});
});
