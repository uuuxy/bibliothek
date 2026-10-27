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

const vorschauen = () =>
	apiFetchMock.mock.calls.filter(([, o]) => JSON.parse(String(o?.body)).dry_run === true);
const vorschau = () => serverAntwort(200, { ...zahlen, dry_run: true });
const keineVerbindung = () => {
	throw new Error('Keine Verbindung zum Server.');
};
const lief =
	'Der Schuljahreswechsel lief am 27.10.2026 um 22:15 Uhr. Ein zweiter Lauf würde alle Schüler ' +
	'ein weiteres Mal versetzen; der nächste geht ab dem 27.04.2027 um 22:15 Uhr.';

// Nach einer verlorenen Antwort stand die Vorschau mit „es wurde noch nichts geändert" da, obwohl
// der Wechsel gelaufen war, in Chromium, Firefox und WebKit (OFFEN.md 5.106). Ob er lief, weiß
// nur der Server; die Ansicht fragt ihn mit einer neuen Vorschau.
describe('Schuljahreswechsel: nach einem Fehler beim Ausführen', () => {
	/**
	 * @param {() => any} ausfuehrung Antwort auf den Wechsel
	 * @param {() => any} nachfrage Antwort auf die Vorschau danach
	 */
	function antworte(ausfuehrung, nachfrage) {
		apiFetchMock.mockReset();
		let rechnungen = 0;
		apiFetchMock.mockImplementation(async (_url, o) => {
			if (JSON.parse(String(o?.body)).confirm === true) return ausfuehrung();
			rechnungen += 1;
			return rechnungen === 1 ? vorschau() : nachfrage();
		});
	}

	it('nennt nach einer verlorenen Antwort den Lauf, den der Server kennt', async () => {
		antworte(keineVerbindung, () => serverAntwort(409, { error: lief }));
		const { k, ja } = await bisZurBestaetigung();
		await fireEvent.click(ja, { detail: 1 });
		await k.findByText(lief);
		expect(k.queryByText(/es wurde noch nichts geändert/)).toBeNull();
		expect(k.queryByRole('button', { name: 'Schuljahr wechseln' })).toBeNull();
	});

	it('zeigt nach einer Ablehnung den Grund und die neu gerechnete Vorschau', async () => {
		antworte(() => serverAntwort(500, { error: 'Die Datenbank antwortet nicht' }), vorschau);
		const { k, ja } = await bisZurBestaetigung();
		await fireEvent.click(ja, { detail: 1 });
		await k.findByText('Die Datenbank antwortet nicht');
		expect(vorschauen()).toHaveLength(2);
		expect(k.getByText(/es wurde noch nichts geändert/)).toBeTruthy();
	});

	it('sagt nicht, es sei nichts geändert, wenn auch die Nachfrage keine Antwort bekommt', async () => {
		antworte(keineVerbindung, keineVerbindung);
		const { k, ja } = await bisZurBestaetigung();
		await fireEvent.click(ja, { detail: 1 });
		await k.findByText('Keine Verbindung zum Server.');
		expect(vorschauen()).toHaveLength(2);
		expect(k.queryByText(/es wurde noch nichts geändert/)).toBeNull();
		expect(k.getByRole('button', { name: 'Vorschau berechnen' })).toBeTruthy();
	});
});
