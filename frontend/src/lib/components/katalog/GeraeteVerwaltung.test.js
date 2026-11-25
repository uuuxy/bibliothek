import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, fireEvent, waitFor } from '@testing-library/svelte';
import GeraeteVerwaltung from './GeraeteVerwaltung.svelte';
import { apiFetch, apiClient } from '../../apiFetch.js';
import { authStore } from '../../stores/authStore.svelte.js';
import { snackbars } from '../../stores/snackbars.svelte.js';

vi.mock('../../apiFetch.js', async (importOriginal) => ({
	.../** @type {any} */ (await importOriginal()),
	apiFetch: vi.fn(),
	apiClient: { post: vi.fn(), put: vi.fn() },
	leseApiFehler: vi.fn(async (/** @type {any} */ res) => JSON.parse(await res.text()).error)
}));

vi.mock('../../stores/snackbars.svelte.js', () => ({ snackbars: { addToast: vi.fn() } }));

// „Noch keine Geräte erfasst" heißt: Der Schrank ist leer, leg eins an. Bis zum
// 12.09.2026 stand derselbe Satz da, wenn der Abruf gescheitert war — mit der Einladung,
// Laptops ein zweites Mal anzulegen, die längst im Bestand stehen (Register,
// Bestands-Durchgang 10.09.2026).
// Jeder Test beginnt mit edit_books, der Test ohne das Recht nimmt es sich selbst. Die CI mischt
// die Reihenfolge (--sequence.shuffle): Ein Recht aus dem Test davor gilt dort nicht.
beforeEach(() => {
	authStore.currentUser = { permissions: ['view_books', 'edit_books'] };
});

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
		expect(queryByText('Noch keine Geräte erfasst')).toBeNull();
	});

	it('leer bleibt leer', async () => {
		vi.mocked(apiFetch).mockResolvedValue(/** @type {any} */ ({ ok: true, json: async () => [] }));
		const { findByText } = render(GeraeteVerwaltung);
		expect(await findByText('Noch keine Geräte erfasst')).toBeTruthy();
	});
});

// Nach dem Speichern geht die Maske zu. Bliebe sie stehen, sähe es aus, als sei nichts geschehen.
describe('GeraeteVerwaltung: Speichern', () => {
	it('schließt die Maske nach dem Anlegen', async () => {
		vi.mocked(apiFetch).mockReset();
		vi.mocked(apiFetch).mockResolvedValue(/** @type {any} */ ({ ok: true, json: async () => [] }));
		vi.mocked(apiClient.post).mockResolvedValue(/** @type {any} */ ({ ok: true }));
		const screen = render(GeraeteVerwaltung);
		await screen.findByText('Noch keine Geräte erfasst');

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

// Anlegen, Ändern und „Defekt melden" verlangt der Server mit edit_books, die Liste nur mit
// view_books. Ein Konto ohne edit_books sah die Knöpfe und bekam erst nach dem Ausfüllen ein 403.
describe('GeraeteVerwaltung ohne edit_books', () => {
	it('zeigt die Liste ohne Anlegen, Bearbeiten und Defekt melden', async () => {
		authStore.currentUser = { permissions: ['view_books'] };
		vi.mocked(apiFetch).mockResolvedValue(
			/** @type {any} */ ({
				ok: true,
				json: async () => [
					{ id: 'g1', modellname: 'Tablet 9', barcode_id: 'G-1', ist_ausleihbar: true }
				]
			})
		);
		const { findByText, queryByRole } = render(GeraeteVerwaltung);

		expect(await findByText('Tablet 9')).toBeTruthy();
		expect(queryByRole('button', { name: /Gerät anlegen/ })).toBeNull();
		expect(queryByRole('button', { name: 'Tablet 9 bearbeiten' })).toBeNull();
		expect(queryByRole('button', { name: /Defekt melden/ })).toBeNull();
	});
});

// Ohne view_students verbirgt der Server den Namen und schickt ausgeliehen_an als leeren Text. Das
// Gerät ist trotzdem verliehen; „im Schrank" wäre an der Theke die falsche Auskunft.
describe('GeraeteVerwaltung ohne view_students', () => {
	it('zeigt ein verliehenes Gerät als verliehen, ohne Namen', async () => {
		authStore.currentUser = { permissions: ['view_books'] };
		vi.mocked(apiFetch).mockResolvedValue(
			/** @type {any} */ ({
				ok: true,
				json: async () => [
					{
						id: 'g1',
						modellname: 'Tablet 1',
						barcode_id: 'G-1',
						ist_ausleihbar: true,
						ausgeliehen_an: ''
					},
					{ id: 'g2', modellname: 'Tablet 2', barcode_id: 'G-2', ist_ausleihbar: true }
				]
			})
		);
		const { findByText, getAllByText, getByText } = render(GeraeteVerwaltung);

		await findByText('Tablet 1');
		expect(getByText('verliehen')).toBeTruthy();
		expect(getAllByText('im Schrank')).toHaveLength(1);
	});
});

// Ein verlorenes Gerät verschwindet nicht: Es steht als „verloren“ da, ohne Bearbeiten und ohne
// „Defekt melden“; taucht es auf, holt „Wieder freigeben“ es zurück wie ein repariertes.
describe('GeraeteVerwaltung: verloren und defekt', () => {
	it('zeigt das verlorene und das defekte Gerät mit „Wieder freigeben“, nur das defekte mit Bearbeiten', async () => {
		vi.mocked(apiFetch).mockReset();
		const geraet = (id, modell, ausleihbar, ausgesondert) => ({
			id,
			modellname: modell,
			barcode_id: `G-${id}`,
			ist_ausleihbar: ausleihbar,
			ist_ausgesondert: ausgesondert,
			ausgeliehen_an: null
		});
		vi.mocked(apiFetch).mockResolvedValue(
			/** @type {any} */ ({
				ok: true,
				json: async () => [
					geraet('1', 'iPad defekt', false, false),
					geraet('2', 'iPad weg', false, true)
				]
			})
		);
		const screen = render(GeraeteVerwaltung);

		expect(await screen.findByText('verloren')).toBeTruthy();
		expect(screen.getByText('defekt oder gesperrt')).toBeTruthy();
		expect(screen.queryByRole('button', { name: 'iPad weg bearbeiten' })).toBeNull();
		expect(screen.getByRole('button', { name: 'iPad defekt bearbeiten' })).toBeTruthy();
		expect(screen.getAllByRole('button', { name: /Wieder freigeben/ })).toHaveLength(2);
	});

	it('holt ein verlorenes Gerät zurück und nennt die stornierte Forderung', async () => {
		vi.mocked(apiFetch).mockReset();
		vi.mocked(apiFetch).mockResolvedValue(
			/** @type {any} */ ({
				ok: true,
				json: async () => [
					{
						id: '2',
						modellname: 'iPad weg',
						barcode_id: 'G-2',
						ist_ausleihbar: false,
						ist_ausgesondert: true
					}
				]
			})
		);
		vi.mocked(apiClient.put).mockResolvedValue(
			/** @type {any} */ ({
				ok: true,
				json: async () => ({
					status: 'success',
					zurueckgeholt: true,
					stornierte_forderungen: 1,
					stornierter_betrag: 15
				})
			})
		);
		const screen = render(GeraeteVerwaltung);
		await fireEvent.click(await screen.findByRole('button', { name: /Wieder freigeben/ }));

		expect(apiClient.put).toHaveBeenCalledWith('/api/geraete/2', { ist_ausleihbar: true });
		await waitFor(() =>
			expect(snackbars.addToast).toHaveBeenCalledWith(
				'Das Gerät ist wieder im Schrank, die Forderung über 15,00\u00a0€ ist storniert',
				'success'
			)
		);
	});
});
