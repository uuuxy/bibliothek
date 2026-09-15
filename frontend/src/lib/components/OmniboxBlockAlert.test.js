import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render } from '@testing-library/svelte';

// „Einmalig ignorieren" folgt dem Recht seiner Route (OFFEN.md 3.3, 15.09.2026): Der Server
// lässt override_block nur mit edit_students wirken (api/action.go) und verwirft es sonst —
// der Dialog erschien dann erneut. Bis dahin sah jede Rolle den Knopf, auch die Helferin, die
// damit nichts bewirken kann. Dasselbe Recht schützt PATCH …/lock („Sperre dauerhaft
// aufheben"). Sichtbarkeit = hatRecht (frontend-hygiene-rechte.test.js), nicht Rolle.
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
import { authStore } from '../stores/authStore.svelte.js';
import OmniboxBlockAlert from './OmniboxBlockAlert.svelte';

describe('OmniboxBlockAlert', () => {
	beforeEach(() => {
		omniboxStore.blockAlert = { message: 'Ausleihe gesperrt: Manuelle Sperre', query: 'B-1' };
		omniboxStore.activeStudent = { id: 's1', vorname: 'Anna', is_manually_blocked: true };
	});

	it('zeigt der Helferin ohne edit_students keinen Übergehen-Knopf', () => {
		authStore.currentUser = { rolle: 'helfer', permissions: ['view_students'] };
		const screen = render(OmniboxBlockAlert, { onReload: () => {} });
		expect(screen.getByText(/Ausleihe blockiert/)).toBeTruthy();
		expect(screen.queryByText(/Einmalig ignorieren/)).toBeNull();
		expect(screen.queryByText(/Sperre dauerhaft aufheben/)).toBeNull();
		expect(screen.getByText(/Abbrechen/)).toBeTruthy();
	});

	it('zeigt beide Knöpfe, wenn das Recht erteilt ist — unabhängig von der Rolle', () => {
		authStore.currentUser = { rolle: 'helfer', permissions: ['edit_students'] };
		const screen = render(OmniboxBlockAlert, { onReload: () => {} });
		expect(screen.getByText(/Einmalig ignorieren/)).toBeTruthy();
		expect(screen.getByText(/Sperre dauerhaft aufheben/)).toBeTruthy();
	});
});
