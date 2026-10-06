import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, fireEvent, waitFor } from '@testing-library/svelte';

vi.mock('./apiFetch.js', async (original) => ({
	...(await original()),
	apiFetch: vi.fn(),
	apiClient: { put: vi.fn() }
}));

import { apiFetch, apiClient } from './apiFetch.js';
import { authStore } from './stores/authStore.svelte.js';
import { toastStore } from './stores/toastStore.svelte.js';
import PermissionManager from './PermissionManager.svelte';

// Die Rechte-Matrix zeigt nur, was der Server geliefert hat: Ohne geladene Matrix stünden
// alle Schalter auf „aus", als wären die Rechte entzogen, und ein Klick schriebe gegen einen
// Stand, den niemand kennt.

const MATRIX = [{ role: 'helfer', permission: 'view_books', allowed: true }];

/** @param {(url: string) => any} rechte Antwort auf den Abruf der Matrix */
function serverMit(rechte) {
	vi.mocked(apiFetch).mockImplementation(
		/** @type {any} */ (
			async (/** @type {string} */ url) =>
				String(url).startsWith('/api/admin/permissions')
					? rechte(url)
					: { ok: true, status: 200, json: async () => [], text: async () => '' }
		)
	);
}

const matrixOk = () => ({ ok: true, status: 200, json: async () => MATRIX });

async function oeffneRechte() {
	const screen = render(PermissionManager);
	await fireEvent.click(screen.getByRole('tab', { name: 'Rollen & Rechte' }));
	return screen;
}

describe('Rechte-Matrix', () => {
	beforeEach(() => {
		vi.clearAllMocks();
		toastStore.toasts = [];
		authStore.currentUser = { id: 1, rolle: 'admin', permissions: ['manage_users'] };
	});

	it('zeigt nach gescheitertem Abruf keine Schalter und lädt beim zweiten Versuch', async () => {
		let abrufe = 0;
		serverMit(() =>
			++abrufe === 1
				? { ok: false, status: 500, text: async () => 'Datenbank nicht da' }
				: matrixOk()
		);
		const screen = await oeffneRechte();

		await screen.findByText('Rechte nicht geladen');
		expect(screen.getByText('Datenbank nicht da')).toBeTruthy();
		expect(screen.queryAllByRole('switch')).toHaveLength(0);

		await fireEvent.click(screen.getByRole('button', { name: 'Erneut versuchen' }));

		await waitFor(() => expect(screen.queryAllByRole('switch').length).toBeGreaterThan(0));
		expect(screen.queryByText('Rechte nicht geladen')).toBeNull();
	});

	it('meldet den Grund des Servers, wenn ein Recht nicht gespeichert wird', async () => {
		serverMit(matrixOk);
		vi.mocked(apiClient.put).mockResolvedValue(
			/** @type {any} */ ({
				ok: false,
				json: async () => ({ error: 'Die Rechte-Matrix ändert nur ein Administrator.' })
			})
		);
		const screen = await oeffneRechte();
		const schalter = await screen.findAllByRole('switch', { name: 'HELFER Rechte umschalten' });

		await fireEvent.click(schalter[0]);

		await waitFor(() =>
			expect(toastStore.toasts.map((t) => `${t.type}: ${t.message}`)).toContain(
				'error: Die Rechte-Matrix ändert nur ein Administrator.'
			)
		);
	});

	// Die Gegenprobe: Ein gespeichertes Recht meldet sich als Erfolg, nicht als Fehler.
	it('meldet ein gespeichertes Recht als Erfolg', async () => {
		serverMit(matrixOk);
		vi.mocked(apiClient.put).mockResolvedValue(/** @type {any} */ ({ ok: true }));
		const screen = await oeffneRechte();
		const schalter = await screen.findAllByRole('switch', { name: 'HELFER Rechte umschalten' });

		await fireEvent.click(schalter[0]);

		await waitFor(() => expect(toastStore.toasts.map((t) => t.type)).toEqual(['success']));
	});
});
