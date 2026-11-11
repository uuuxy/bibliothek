import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, fireEvent, waitFor } from '@testing-library/svelte';
import GeraeteVerwaltung from './GeraeteVerwaltung.svelte';
import { apiFetch, apiClient } from '../../apiFetch.js';

vi.mock('../../apiFetch.js', () => ({
	apiFetch: vi.fn(),
	apiClient: { post: vi.fn(), put: vi.fn() },
	leseApiFehler: vi.fn(async (/** @type {any} */ res) => JSON.parse(await res.text()).error)
}));

vi.mock('../../stores/snackbars.svelte.js', () => ({ snackbars: { addToast: vi.fn() } }));

// „Noch keine Geräte erfasst" heißt: Der Schrank ist leer, leg eins an. Bis zum
// 12.09.2026 stand derselbe Satz da, wenn der Abruf gescheitert war — mit der Einladung,
// Laptops ein zweites Mal anzulegen, die längst im Bestand stehen (Register,
// Bestands-Durchgang 10.09.2026).
describe('GeraeteVerwaltung', () => {
	beforeEach(() => {
		vi.mocked(apiFetch).mockReset();
	});

	it('unterscheidet „Schrank leer" von „nicht geladen"', async () => {
		vi.mocked(apiFetch).mockResolvedValue(
			/** @type {any} */ ({
				ok: false,
				status: 500,
				text: async () => JSON.stringify({ error: 'Datenbank nicht erreichbar' })
			})
		);
		const { findByText, queryByText } = render(GeraeteVerwaltung);

		expect(await findByText('Datenbank nicht erreichbar')).toBeTruthy();
		expect(queryByText('Noch keine Geräte erfasst.')).toBeNull();
	});

	it('leer bleibt leer', async () => {
		vi.mocked(apiFetch).mockResolvedValue(/** @type {any} */ ({ ok: true, json: async () => [] }));
		const { findByText } = render(GeraeteVerwaltung);
		expect(await findByText('Noch keine Geräte erfasst.')).toBeTruthy();
	});
});

// Nach dem Speichern geht die Maske zu. Bliebe sie stehen, sähe es aus, als sei nichts geschehen.
describe('GeraeteVerwaltung: Speichern', () => {
	it('schließt die Maske nach dem Anlegen', async () => {
		vi.mocked(apiFetch).mockReset();
		vi.mocked(apiFetch).mockResolvedValue(/** @type {any} */ ({ ok: true, json: async () => [] }));
		vi.mocked(apiClient.post).mockResolvedValue(/** @type {any} */ ({ ok: true }));
		const screen = render(GeraeteVerwaltung);
		await screen.findByText('Noch keine Geräte erfasst.');

		await fireEvent.click(screen.getByRole('button', { name: /anlegen|erfassen|Neues Gerät/i }));
		const modell = screen.getByLabelText(/Modell/);
		await fireEvent.input(modell, { target: { value: 'iPad 9' } });
		await fireEvent.click(screen.getByRole('button', { name: /^(Speichern|Anlegen)/ }));

		await waitFor(() => expect(apiClient.post).toHaveBeenCalledTimes(1));
		await waitFor(() => expect(screen.queryByLabelText(/Modell/)).toBeNull());
	});
});

// Beim Bearbeiten bleibt nur der Barcode gesperrt, er klebt am Gerät. Die Seriennummer lässt sich
// nachtragen: Ein Gerät, das ohne sie angelegt wurde, bekäme sie sonst nie.
describe('GeraeteVerwaltung: Bearbeiten', () => {
	it('nimmt eine Seriennummer an und schickt sie', async () => {
		vi.mocked(apiFetch).mockReset();
		vi.mocked(apiFetch).mockResolvedValue(
			/** @type {any} */ ({
				ok: true,
				json: async () => [
					{ id: 'g1', modellname: 'iPad 9', barcode_id: 'G-0001', seriennummer: '' }
				]
			})
		);
		vi.mocked(apiClient.put).mockResolvedValue(/** @type {any} */ ({ ok: true }));
		const screen = render(GeraeteVerwaltung);

		await fireEvent.click(await screen.findByLabelText('iPad 9 bearbeiten'));
		const barcode = /** @type {HTMLInputElement} */ (screen.getByLabelText(/Barcode/));
		const serie = /** @type {HTMLInputElement} */ (screen.getByLabelText('Seriennummer'));
		expect(barcode.disabled).toBe(true);
		expect(serie.disabled).toBe(false);

		await fireEvent.input(serie, { target: { value: 'DMPXK12345' } });
		await fireEvent.click(screen.getByRole('button', { name: 'Speichern' }));

		await waitFor(() =>
			expect(apiClient.put).toHaveBeenCalledWith('/api/geraete/g1', { seriennummer: 'DMPXK12345' })
		);
	});
});
