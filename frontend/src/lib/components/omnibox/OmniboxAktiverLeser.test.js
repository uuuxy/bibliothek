import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, cleanup, fireEvent, waitFor } from '@testing-library/svelte';

vi.mock('../../apiFetch.js', async (importOriginal) => ({
	.../** @type {any} */ (await importOriginal()),
	apiFetch: vi.fn(),
	apiClient: { post: vi.fn(), patch: vi.fn() },
	registriereSitzungAbgelaufenHandler: vi.fn()
}));
vi.mock('../../utils/audio.js', () => ({
	spieleErfolgston: vi.fn(),
	spieleFehlerton: vi.fn()
}));
vi.mock('../../../inventur/lib/store.svelte.js', () => ({ melde: vi.fn(), appState: {} }));

import OmniboxAktiverLeser from './OmniboxAktiverLeser.svelte';
import { omniboxStore } from '../../stores/omnibox.svelte.js';
import { authStore } from '../../stores/authStore.svelte.js';
import { apiFetch } from '../../apiFetch.js';

// So kommt der Leser aus der Antwort auf den Scan an (api/theke/theken_leser.go): ohne
// Abgangsjahr, Sperrgrund, Foto und Forderungen.
const AUS_DEM_SCAN = {
	id: 'l-1',
	art: 'schueler',
	barcode_id: 'S-1',
	vorname: 'Hanna',
	nachname: 'Beispiel',
	klasse: '07A',
	ist_gesperrt: false,
	is_manually_blocked: false
};

/** @param {string[]} rechte */
function melde(rechte) {
	authStore.currentUser = /** @type {any} */ ({ rolle: 'helfer', permissions: rechte });
}

beforeEach(() => {
	vi.mocked(apiFetch).mockReset();
	vi.mocked(apiFetch).mockResolvedValue(
		/** @type {any} */ ({ ok: true, status: 200, json: async () => [] })
	);
	omniboxStore.aktiverLeser = AUS_DEM_SCAN;
});

afterEach(() => {
	omniboxStore.aktiverLeser = null;
	omniboxStore.stoppeZeitgeber();
	cleanup();
});

// Die Akte steht hinter view_students, die Theke hängt an perform_actions. Der Helfer bekam bei
// jedem Scan „Akte nicht geladen" mit einem Knopf „Erneut versuchen", der nie etwas ändern konnte.
describe('Der geladene Leser an der Theke', () => {
	it('zeigt ohne view_students, was der Scan trägt, und fragt den Server nichts', () => {
		melde(['perform_actions', 'view_books']);
		const screen = render(OmniboxAktiverLeser);

		expect(screen.getByRole('heading', { name: 'Hanna Beispiel' })).toBeTruthy();
		expect(screen.getByText('Klasse 07A')).toBeTruthy();
		expect(screen.getByText('S-1')).toBeTruthy();
		expect(screen.getByText('Aktiv')).toBeTruthy();
		expect(screen.getByText(/Recht „Leserdatei anzeigen“/)).toBeTruthy();
		expect(screen.queryByText('Akte nicht geladen')).toBeNull();
		expect(screen.queryByText('Offene Forderungen')).toBeNull();
		expect(screen.queryByText(/^Abgang/)).toBeNull();
		expect(apiFetch).not.toHaveBeenCalled();
	});

	it('nennt eine Sperre auch ohne view_students', () => {
		melde(['perform_actions']);
		omniboxStore.aktiverLeser = { ...AUS_DEM_SCAN, is_manually_blocked: true };
		const screen = render(OmniboxAktiverLeser);

		expect(screen.getByText('Ausleihe gesperrt')).toBeTruthy();
		expect(screen.getByText('Gesperrt')).toBeTruthy();
	});

	it('schließt ohne view_students wie die Akte', async () => {
		melde(['perform_actions']);
		const screen = render(OmniboxAktiverLeser);

		await fireEvent.click(screen.getByRole('button', { name: 'Schließen' }));
		expect(omniboxStore.aktiverLeser).toBeNull();
	});

	it('öffnet mit view_students die Akte', async () => {
		melde(['perform_actions', 'view_students']);
		render(OmniboxAktiverLeser);

		await waitFor(() => expect(apiFetch).toHaveBeenCalledWith('/api/schueler/l-1'));
	});
});
