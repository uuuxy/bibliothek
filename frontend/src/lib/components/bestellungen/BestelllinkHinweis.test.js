import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/svelte';

vi.mock('../../apiFetch.js', async (importOriginal) => ({
	.../** @type {any} */ (await importOriginal()),
	apiGet: vi.fn()
}));

import { apiGet } from '../../apiFetch.js';
import { bestellStore } from '../../stores/bestellStore.svelte.js';
import BestelllinkHinweis from './BestelllinkHinweis.svelte';

const HINWEIS = 'Bestell-Einstellungen nicht geladen';

// Bei jedem ersten Öffnen des Bestellwesens stand „Bestell-Einstellungen nicht geladen", bis die
// Konfiguration kam, in Chromium, Firefox und WebKit (OFFEN.md 5.106).
describe('BestelllinkHinweis', () => {
	it('schweigt, solange die Konfiguration aussteht, und meldet erst ihr Ausbleiben', async () => {
		/** @type {(e: Error) => void} */
		let scheitere = () => {};
		vi.mocked(apiGet).mockImplementation(() => new Promise((_, nein) => (scheitere = nein)));
		render(BestelllinkHinweis);
		const laden = bestellStore.loadKonfiguration();

		expect(screen.queryByText(HINWEIS), 'während die Antwort aussteht').toBeNull();

		scheitere(new Error('Zu viele Anfragen'));
		expect(await laden).toBe(false);
		expect(await screen.findByText(HINWEIS)).toBeTruthy();

		vi.mocked(apiGet).mockResolvedValue({ bestelllink_ohne_adresse: false });
		expect(await bestellStore.loadKonfiguration()).toBe(true);
		await vi.waitFor(() => expect(screen.queryByText(HINWEIS)).toBeNull());
	});
});
