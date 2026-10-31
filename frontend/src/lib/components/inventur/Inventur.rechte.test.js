import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, cleanup, fireEvent, waitFor } from '@testing-library/svelte';
import Inventur from './Inventur.svelte';
import { apiFetch } from '../../apiFetch.js';
import { authStore } from '../../stores/authStore.svelte.js';

vi.mock('../../apiFetch.js', async (importOriginal) => ({
	.../** @type {any} */ (await importOriginal()),
	apiFetch: vi.fn()
}));

const LAUF = { session_id: 's1', label: 'Alles', erwartet: 3, fehlend: 1, gestartet_am: '' };
const abgerufen = () => vi.mocked(apiFetch).mock.calls.map(([pfad]) => String(pfad));

/** @param {string[]} rechte */
function melde(rechte) {
	authStore.currentUser = /** @type {any} */ ({ rolle: 'mitarbeiter', permissions: rechte });
}

beforeEach(() => {
	vi.mocked(apiFetch).mockReset();
	vi.mocked(apiFetch).mockImplementation(async (pfad) => {
		const liste = String(pfad).startsWith('/api/inventur/sessions') ? [LAUF] : [];
		return /** @type {any} */ ({ ok: true, status: 200, json: async () => liste });
	});
});

afterEach(cleanup);

// Der Menüpunkt hängt an inventory_scan; Starten, Abschließen, Verwerfen und die früheren Läufe
// verlangt der Server mit manage_inventory. Wer nur scannen darf, bekäme bei jedem Öffnen und
// jedem dieser Knöpfe eine Ablehnung.
describe('Inventur nach Rechten', () => {
	it('fragt mit inventory_scan allein nur die laufenden Inventuren ab', async () => {
		melde(['inventory_scan']);
		const screen = render(Inventur);

		await screen.findByRole('button', { name: 'Fortsetzen' });
		expect(screen.queryByRole('button', { name: 'Neue Bestandsprüfung starten' })).toBeNull();
		expect(screen.queryByRole('button', { name: 'Verwerfen' })).toBeNull();
		expect(screen.getByText(/Recht „Bestand verwalten“/)).toBeTruthy();
		expect(abgerufen()).toEqual(['/api/inventur/sessions']);

		await fireEvent.click(screen.getByRole('button', { name: 'Fortsetzen' }));
		expect(screen.getByLabelText('Barcode scannen')).toBeTruthy();
		expect(screen.queryByRole('button', { name: 'Inventur abschließen' })).toBeNull();
	});

	it('bietet mit manage_inventory alles an und lädt die Listen für den Start', async () => {
		melde(['inventory_scan', 'manage_inventory']);
		const screen = render(Inventur);

		await screen.findByRole('button', { name: 'Verwerfen' });
		expect(screen.getByRole('button', { name: 'Neue Bestandsprüfung starten' })).toBeTruthy();
		await waitFor(() =>
			expect(abgerufen()).toEqual([
				'/api/inventur/sessions',
				'/api/signaturen',
				'/api/faecher',
				'/api/inventur/abgeschlossen'
			])
		);

		await fireEvent.click(screen.getByRole('button', { name: 'Fortsetzen' }));
		expect(screen.getByRole('button', { name: 'Inventur abschließen' })).toBeTruthy();
	});
});
