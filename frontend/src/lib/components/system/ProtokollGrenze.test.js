import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render } from '@testing-library/svelte';

vi.mock('../../apiFetch.js', async (importOriginal) => ({
	.../** @type {any} */ (await importOriginal()),
	apiFetch: vi.fn()
}));

import { apiFetch } from '../../apiFetch.js';
import VerwaltungsProtokoll from './VerwaltungsProtokoll.svelte';
import Logbuch from './Logbuch.svelte';

// Beide Protokolle liefert der Server nur bis zu den jüngsten Einträgen; die Grenze nennt er im
// Kopf X-Grenze. Das Logbuch sagte es, das Verwaltungsprotokoll nicht: Wer dort einen älteren
// Eintrag suchte, fand ihn nicht und bekam keinen Hinweis, dass es darunter weitergeht.
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

/**
 * @param {number} n
 * @param {string} grenze was der Server im Kopf X-Grenze nennt
 */
function antworte(n, grenze = '1000') {
	vi.mocked(apiFetch).mockResolvedValue(
		/** @type {any} */ ({
			ok: true,
			status: 200,
			headers: new Headers({ 'X-Grenze': grenze }),
			json: async () => zeilen(n)
		})
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

	// Die Zahl steht nur im Server. Führte die Oberfläche eine eigene, behauptete sie nach einer
	// geänderten Grenze eine falsche Zahl oder schwiege über eine gekappte Liste.
	it('nimmt die Grenze aus der Antwort des Servers', async () => {
		antworte(500, '500');
		const screen = render(/** @type {any} */ (Seite));
		const satz = await screen.findByText(SATZ);
		expect(satz.querySelector('strong')?.textContent).toBe('500');
	});
});
