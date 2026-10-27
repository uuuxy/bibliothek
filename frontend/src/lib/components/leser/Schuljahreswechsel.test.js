import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';

// leseApiFehler bleibt echt (ARCHITEKTUR.md 8.14).
vi.mock('../../apiFetch.js', async (importOriginal) => ({
	.../** @type {any} */ (await importOriginal()),
	apiFetch: vi.fn()
}));

import { apiFetch } from '../../apiFetch.js';
import { serverAntwort } from '../../serverAntwort.js';
import Schuljahreswechsel from './Schuljahreswechsel.svelte';

const apiFetchMock = vi.mocked(apiFetch);
const zahlen = {
	promoted_count: 120,
	archived_count: 30,
	mapping_versetzt: 4,
	mapping_entfernt: 1
};

/** Die Aufrufe, die den Wechsel ausführen, nicht nur vorrechnen. */
const ausfuehrungen = () =>
	apiFetchMock.mock.calls.filter(([, o]) => JSON.parse(String(o?.body)).confirm === true);

/** Bis zur Bestätigung: Vorschau, dann „Schuljahr wechseln" mit einem ersten Klick. */
async function bisZurBestaetigung() {
	const k = render(Schuljahreswechsel);
	await fireEvent.click(k.getByRole('button', { name: 'Vorschau berechnen' }));
	await fireEvent.click(await k.findByRole('button', { name: 'Schuljahr wechseln' }), {
		detail: 1
	});
	return { k, ja: k.getByRole('button', { name: 'Ja, unwiderruflich ausführen' }) };
}

// Ein Doppelklick auf „Schuljahr wechseln" führte den Wechsel aus: Die Bestätigung erscheint an
// derselben Stelle, und der zweite Klick traf sie, in Chromium, Firefox und WebKit (OFFEN.md 5.106).
describe('Schuljahreswechsel: die Bestätigung', () => {
	beforeEach(() => {
		apiFetchMock.mockReset();
		apiFetchMock.mockImplementation(async (_url, o) =>
			serverAntwort(200, { ...zahlen, dry_run: JSON.parse(String(o?.body)).dry_run === true })
		);
	});

	it('nimmt den zweiten Klick eines Doppelklicks nicht als Bestätigung', async () => {
		const { k, ja } = await bisZurBestaetigung();
		await fireEvent.click(ja, { detail: 2 });
		expect(ausfuehrungen()).toHaveLength(0);
		expect(k.getByRole('button', { name: 'Ja, unwiderruflich ausführen' })).toBeTruthy();
	});

	it.each([
		['mit einem eigenen Klick', 1],
		['über die Tastatur', 0]
	])('führt %s aus (detail %i)', async (_weg, detail) => {
		const { k, ja } = await bisZurBestaetigung();
		await fireEvent.click(ja, { detail });
		await k.findByText('Schuljahreswechsel abgeschlossen.');
		expect(ausfuehrungen()).toHaveLength(1);
	});
});
