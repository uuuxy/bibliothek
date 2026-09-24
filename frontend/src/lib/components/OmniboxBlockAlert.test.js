import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';

// „Einmalig ignorieren" und „Sperre aufheben" folgen dem Recht ihrer Route (OFFEN.md 3.3,
// 15.09.2026): Der Server lässt override_block nur mit edit_students wirken (api/action.go),
// und PATCH …/lock verlangt dasselbe Recht. Sichtbarkeit = hatRecht
// (frontend-hygiene-rechte.test.js), nicht Rolle.
//
// Seit dem 24.09.2026 entscheidet das Merkmal des Servers, welcher der beiden Knöpfe steht:
// Eine Sperre am Leser (X-Sperre: leser) lässt sich nur aufheben, ein Hinweis
// (X-Sperre: uebergehbar) nur einmalig übergehen.
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

// Nach jedem Fall die Zeitgeber der Theke stoppen. `scanfeldWiederScharfstellen` plant
// einen Fokussprung über 50 ms, der `document` anfasst — endet die Datei vorher, baut
// Vitest jsdom ab, und der Rückruf reisst den GANZEN Lauf rot („Unhandled Errors:
// document is not defined"), obwohl jeder Test grün ist. Genau so stand die CI am
// 16.09.2026. Belegt in stores/omniboxZeitgeber.test.js.
afterEach(() => omniboxStore.stoppeZeitgeber());

import { authStore } from '../stores/authStore.svelte.js';
import OmniboxBlockAlert from './OmniboxBlockAlert.svelte';

/** @param {'leser' | 'uebergehbar'} art */
function sperre(art) {
	omniboxStore.blockAlert = {
		message:
			art === 'leser'
				? 'die ausleihe ist gesperrt: Manuelle Sperre: Ausweis verloren'
				: 'die ausleihe ist gesperrt: 1 unbezahlte(r) Schadensfall/-fälle offen',
		query: 'B-1',
		art
	};
}

const mitRecht = () =>
	(authStore.currentUser = { rolle: 'helfer', permissions: ['edit_students'] });
const ohneRecht = () =>
	(authStore.currentUser = { rolle: 'helfer', permissions: ['view_students'] });

describe('OmniboxBlockAlert', () => {
	beforeEach(() => {
		vi.clearAllMocks();
		omniboxStore.activeStudent = { id: 's1', vorname: 'Anna', is_manually_blocked: true };
	});

	it('Sperre am Leser: nur Aufheben, kein Übergehen', () => {
		mitRecht();
		sperre('leser');
		const screen = render(OmniboxBlockAlert, { onReload: () => {} });
		expect(screen.getByText(/Ausleihe blockiert/)).toBeTruthy();
		expect(screen.getByRole('button', { name: 'Sperre aufheben' })).toBeTruthy();
		expect(screen.queryByText(/Einmalig ignorieren/)).toBeNull();
	});

	it('Hinweis: nur einmalig Übergehen, kein Aufheben', () => {
		mitRecht();
		sperre('uebergehbar');
		const screen = render(OmniboxBlockAlert, { onReload: () => {} });
		expect(screen.getByText(/Einmalig ignorieren/)).toBeTruthy();
		expect(screen.queryByText(/Sperre aufheben/)).toBeNull();
	});

	it('ohne edit_students: kein Knopf, der Satz nennt, was fehlt', () => {
		ohneRecht();
		sperre('leser');
		let screen = render(OmniboxBlockAlert, { onReload: () => {} });
		expect(screen.getByText(/Aufheben kann nur, wer Schülerdaten bearbeiten darf/)).toBeTruthy();
		expect(screen.queryByText(/Sperre aufheben/)).toBeNull();
		expect(screen.getByText(/Abbrechen/)).toBeTruthy();
		screen.unmount();

		sperre('uebergehbar');
		screen = render(OmniboxBlockAlert, { onReload: () => {} });
		expect(screen.getByText(/Übergehen kann nur, wer Schülerdaten bearbeiten darf/)).toBeTruthy();
		expect(screen.queryByText(/Einmalig ignorieren/)).toBeNull();
	});

	// Ein Handscanner schickt nach jedem Scan ein Enter. Der erste Fokus steht deshalb auf
	// „Abbrechen" — ein Scan bei offenem Dialog darf weder aufheben noch übergehen.
	for (const art of /** @type {const} */ (['leser', 'uebergehbar'])) {
		it(`der erste Fokus steht auf „Abbrechen" (${art})`, async () => {
			mitRecht();
			sperre(art);
			const screen = render(OmniboxBlockAlert, { onReload: () => {} });
			await vi.waitFor(() =>
				expect(document.activeElement).toBe(screen.getByRole('button', { name: 'Abbrechen' }))
			);
		});
	}

	it('Aufheben geht über die Sperr-Tür und wiederholt den Scan ohne Übergehen', async () => {
		mitRecht();
		sperre('leser');
		vi.mocked(apiClient.patch).mockResolvedValue(
			/** @type {any} */ ({
				ok: true,
				json: async () => ({ id: 's1', is_manually_blocked: false, ist_gesperrt: false })
			})
		);
		vi.mocked(apiClient.post).mockResolvedValue(
			/** @type {any} */ ({ ok: true, json: async () => ({ type: 'info' }) })
		);
		const screen = render(OmniboxBlockAlert, { onReload: () => {} });
		await fireEvent.click(screen.getByRole('button', { name: 'Sperre aufheben' }));

		await vi.waitFor(() => expect(apiClient.post).toHaveBeenCalled());
		expect(apiClient.patch).toHaveBeenCalledWith('/api/admin/students/s1/lock', {
			is_locked: false
		});
		expect(apiClient.post).toHaveBeenCalledWith(
			'/api/action',
			expect.objectContaining({ query: 'B-1', override_block: false })
		);
		expect(omniboxStore.activeStudent?.is_manually_blocked).toBe(false);
		expect(omniboxStore.blockAlert).toBeNull();
	});

	it('scheitert das Aufheben, steht der Grund im Dialog (M3: Fehler der Bestätigung im Dialog)', async () => {
		mitRecht();
		sperre('leser');
		vi.mocked(apiClient.patch).mockResolvedValue(
			/** @type {any} */ ({
				ok: false,
				json: async () => ({ error: 'Ein anonymisierter Datensatz bleibt gesperrt.' })
			})
		);
		const screen = render(OmniboxBlockAlert, { onReload: () => {} });
		await fireEvent.click(screen.getByRole('button', { name: 'Sperre aufheben' }));

		await vi.waitFor(() =>
			expect(screen.getByRole('alert').textContent).toContain(
				'Ein anonymisierter Datensatz bleibt gesperrt.'
			)
		);
		expect(apiClient.post).not.toHaveBeenCalled();
		expect(omniboxStore.blockAlert).not.toBeNull();
	});
});
