import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render } from '@testing-library/svelte';

vi.mock('../../apiFetch.js', async (importOriginal) => ({
	.../** @type {any} */ (await importOriginal()),
	apiFetch: vi.fn()
}));

import { apiFetch } from '../../apiFetch.js';
import VerwaltungsProtokoll from './VerwaltungsProtokoll.svelte';
import Logbuch from './Logbuch.svelte';

// Beide Protokolle liefert der Server nur bis zu den jüngsten 1000 Einträgen. Das Logbuch sagte
// es, das Verwaltungsprotokoll nicht: Wer dort einen älteren Eintrag suchte, fand ihn nicht und
// bekam keinen Hinweis, dass es darunter weitergeht.
const SATZ = /jüngsten Einträge\. Ältere Vorgänge sind protokolliert/;

/** @param {number} n */
const zeilen = (n) =>
	Array.from({ length: n }, (_, i) => ({
		id: `z${i}`,
		admin_name: 'Admin',
		aktion: 'BENUTZER_GEAENDERT',
		details: {},
		ip_adresse: '',
		zeitstempel: '2026-11-01T10:00:00Z',
		action: 'checkout',
		created_at: '2026-11-01T10:00:00Z'
	}));

/** @param {number} n */
function antworte(n) {
	vi.mocked(apiFetch).mockResolvedValue(
		/** @type {any} */ ({ ok: true, status: 200, json: async () => zeilen(n) })
	);
}

describe.each([
	['Verwaltungsprotokoll', VerwaltungsProtokoll],
	['Logbuch', Logbuch]
])('%s: gekappte Liste', (_name, Seite) => {
	// Mit Klammern: Was beforeEach zurückgibt, ruft Vitest nach dem Test als Aufräumfunktion.
	beforeEach(() => {
		vi.mocked(apiFetch).mockReset();
	});

	it('nennt die Grenze bei 1000 geladenen Einträgen', async () => {
		antworte(1000);
		const screen = render(/** @type {any} */ (Seite));
		expect(await screen.findByText(SATZ)).toBeTruthy();
	});

	it('schweigt darunter (Gegenprobe)', async () => {
		antworte(999);
		const screen = render(/** @type {any} */ (Seite));
		await vi.waitFor(() => expect(apiFetch).toHaveBeenCalled());
		await new Promise((r) => setTimeout(r, 50));
		expect(screen.queryByText(SATZ)).toBeNull();
	});
});
