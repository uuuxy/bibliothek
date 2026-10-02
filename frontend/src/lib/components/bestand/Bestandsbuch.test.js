import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';

vi.mock('../../apiFetch.js', () => ({ apiFetch: vi.fn() }));

import { apiFetch } from '../../apiFetch.js';
import Bestandsbuch from './Bestandsbuch.svelte';

const LAND = 'Lernmittelfreiheit (Land)';
const TRAEGER = 'Schülerbücherei (Schulträger)';
const OHNE = 'ohne Zuordnung';
const LEER = 'Kein Zugang in diesem Zeitraum.';

const spalten = [
	{ kopf: 'Zugang', feld: 'datum' },
	{ kopf: 'Nummer', feld: 'barcode' },
	{ kopf: 'Titel', feld: 'titel' }
];

/** @param {number} anzahl @param {string} kuerzel */
const zeilen = (anzahl, kuerzel) =>
	Array.from({ length: anzahl }, (_, i) => ({
		datum: '2026-09-20T00:00:00Z',
		barcode: `${kuerzel}-${i + 1}`,
		titel: `Titel ${kuerzel} ${i + 1}`
	}));

/** Die Antwort des Servers: Land und Schulträger stehen immer da, „ohne Zuordnung" nur mit Zeilen. */
const antwort = (/** @type {{ land?: number, traeger?: number, ohne?: number }} */ zahl) => {
	const abschnitte = [
		{ topf: 'land', titel: LAND, zeilen: zeilen(zahl.land ?? 0, 'L') },
		{ topf: 'schultraeger', titel: TRAEGER, zeilen: zeilen(zahl.traeger ?? 0, 'T') }
	];
	if (zahl.ohne) abschnitte.push({ topf: '', titel: OHNE, zeilen: zeilen(zahl.ohne, 'O') });
	return /** @type {any} */ ({
		ok: true,
		json: async () => ({ von: '2026-09-16', bis: '2027-03-15', abschnitte })
	});
};

/** @param {{ land?: number, traeger?: number, ohne?: number }} zahl */
async function zugangsbuch(zahl) {
	vi.mocked(apiFetch).mockResolvedValue(antwort(zahl));
	const screen = render(Bestandsbuch, {
		pfad: 'zugangsbuch',
		buchname: 'Zugangsbuch',
		wortSingular: 'Zugang',
		spalten
	});
	await screen.findByRole('group', { name: 'Nach Mittelherkunft filtern' });
	return screen;
}

/**
 * Die Namen der Tabellen, die zu sehen sind.
 * @param {{ queryAllByRole: (rolle: string) => HTMLElement[] }} screen
 */
const tabellen = (screen) =>
	screen.queryAllByRole('table').map((t) => t.querySelector('caption')?.textContent);

beforeEach(() => vi.clearAllMocks());

// Die Töpfe stehen als Felder mit ihrer Zahl unter dem Zeitraum. Ein leerer Topf belegt unten
// keinen Abschnitt; dass nichts zuging, sagt die Null im Feld.
describe('Bestandsbuch: die Töpfe als Felder mit Zahl', () => {
	it('nennt jeden Topf mit seiner Zahl und zeigt unten nur Töpfe mit Einträgen', async () => {
		const screen = await zugangsbuch({ ohne: 2 });

		for (const name of [`${LAND} · 0`, `${TRAEGER} · 0`, `${OHNE} · 2`])
			expect(screen.getByRole('button', { name }), name).toBeTruthy();
		expect(tabellen(screen)).toEqual([`Zugangsbuch — ${OHNE}`]);
		expect(screen.queryByText(LEER)).toBeNull();
		expect(screen.getByRole('heading', { name: `${OHNE} · 2 Exemplare` })).toBeTruthy();
	});

	it('zeigt nach einem Klick nur diesen Topf und nach dem zweiten wieder alle', async () => {
		const screen = await zugangsbuch({ land: 1, traeger: 2 });
		expect(tabellen(screen)).toEqual([`Zugangsbuch — ${LAND}`, `Zugangsbuch — ${TRAEGER}`]);

		const feld = screen.getByRole('button', { name: `${TRAEGER} · 2` });
		await fireEvent.click(feld);
		expect(tabellen(screen)).toEqual([`Zugangsbuch — ${TRAEGER}`]);
		expect(feld.getAttribute('aria-pressed')).toBe('true');

		await fireEvent.click(feld);
		expect(tabellen(screen)).toEqual([`Zugangsbuch — ${LAND}`, `Zugangsbuch — ${TRAEGER}`]);
	});

	it('sagt bei einem leeren Topf erst nach dem Klick darauf, dass nichts zuging', async () => {
		const screen = await zugangsbuch({ traeger: 1 });
		expect(screen.queryByText(LEER)).toBeNull();

		await fireEvent.click(screen.getByRole('button', { name: `${LAND} · 0` }));
		expect(tabellen(screen)).toEqual([]);
		expect(screen.getByRole('heading', { name: `${LAND} · 0 Exemplare` })).toBeTruthy();
		expect(screen.getByText(LEER)).toBeTruthy();
	});

	it('sagt es einmal, wenn in keinem Topf etwas steht', async () => {
		const screen = await zugangsbuch({});
		expect(tabellen(screen)).toEqual([]);
		expect(screen.getAllByText(LEER)).toHaveLength(1);
	});

	// Eine Liste mit Zehntausenden Zeilen neu aufzubauen dauert Sekunden. Sie bleibt deshalb im
	// Dokument und wird nur ausgeblendet.
	it('baut eine ausgeblendete Liste beim Zurückschalten nicht neu auf', async () => {
		const screen = await zugangsbuch({ land: 1, traeger: 2 });
		const vorher = screen.getByRole('table', { name: `Zugangsbuch — ${TRAEGER}` });

		const feld = screen.getByRole('button', { name: `${LAND} · 1` });
		await fireEvent.click(feld);
		expect(tabellen(screen)).toEqual([`Zugangsbuch — ${LAND}`]);
		await fireEvent.click(feld);

		expect(screen.getByRole('table', { name: `Zugangsbuch — ${TRAEGER}` })).toBe(vorher);
	});

	it('schreibt große Zahlen mit Tausenderpunkt, im Feld wie in der Überschrift', async () => {
		const screen = await zugangsbuch({ ohne: 1000 });
		expect(screen.getByRole('button', { name: `${OHNE} · 1.000` })).toBeTruthy();
		expect(screen.getByRole('heading', { name: `${OHNE} · 1.000 Exemplare` })).toBeTruthy();
	});

	// Der Satz erklärt die Liste „ohne Zuordnung“; ohne die Liste erklärt er nichts.
	it('erklärt „ohne Zuordnung“ nur, solange diese Liste zu sehen ist', async () => {
		const screen = await zugangsbuch({ land: 1, ohne: 1 });
		expect(screen.getByText(/keine Bestellung hinterlegt/)).toBeTruthy();

		await fireEvent.click(screen.getByRole('button', { name: `${LAND} · 1` }));
		expect(screen.queryByText(/keine Bestellung hinterlegt/)).toBeNull();
	});

	// Ein neuer Zeitraum kann die Liste „ohne Zuordnung“ entfallen lassen. Dann gilt die Wahl
	// nicht mehr, und die Seite zeigt wieder alle Töpfe mit Einträgen.
	it('zeigt wieder alle Töpfe, wenn der gewählte im neuen Zeitraum nicht vorkommt', async () => {
		const screen = await zugangsbuch({ land: 1, ohne: 1 });
		await fireEvent.click(screen.getByRole('button', { name: `${OHNE} · 1` }));
		expect(tabellen(screen)).toEqual([`Zugangsbuch — ${OHNE}`]);

		vi.mocked(apiFetch).mockResolvedValue(antwort({ land: 3 }));
		await fireEvent.click(screen.getByRole('button', { name: 'Anzeigen' }));
		await screen.findByRole('button', { name: `${LAND} · 3` });
		expect(tabellen(screen)).toEqual([`Zugangsbuch — ${LAND}`]);
	});
});
