import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, fireEvent, waitFor } from '@testing-library/svelte';

const leitung = vi.hoisted(() => ({
	/** @type {Map<string, Set<() => void>>} */
	hoerer: new Map(),
	/** @param {string} name */
	sende(name) {
		for (const holeNeu of this.hoerer.get(name) ?? []) holeNeu();
	}
}));
vi.mock('../../liveEvents.js', async (original) => ({
	.../** @type {any} */ (await original()),
	abonniereNeuladen: (/** @type {string} */ name, /** @type {() => void} */ holeNeu) => {
		if (!leitung.hoerer.has(name)) leitung.hoerer.set(name, new Set());
		leitung.hoerer.get(name)?.add(holeNeu);
		return () => leitung.hoerer.get(name)?.delete(holeNeu);
	}
}));
vi.mock('../../apiFetch.js', async (original) => ({
	.../** @type {any} */ (await original()),
	apiGet: vi.fn()
}));

import { apiGet } from '../../apiFetch.js';
import BestellHistorie from './BestellHistorie.svelte';

const holen = vi.mocked(apiGet);
const SUMMEN = {
	gesamt: 1,
	gesamtbetrag: 0,
	gesamt_exemplare: 2,
	offene_bestaetigungen: 0,
	nach_mittel: []
};
/** @param {string} lieferant @param {string} mittel */
const bestellung = (lieferant, mittel) => ({
	id: lieferant,
	bestelldatum: '2026-10-01T08:00:00Z',
	lieferant_name: lieferant,
	lieferant_email: 'handel@example.org',
	kundennummer: '',
	mittel,
	anzahl_exemplare: 2,
	gesamtbetrag: 0
});

describe('BestellHistorie', () => {
	beforeEach(() => vi.clearAllMocks());

	// Leer heißt leer, ein Ladefehler heißt Ladefehler: Vorher blieb „Lade Bestellhistorie…"
	// für immer stehen, und nach einem gescheiterten Nachladen stand die alte Liste unter dem
	// neuen Filter.
	it('zeigt nach einem gescheiterten Laden den Ladefehler mit dem Weg zurück', async () => {
		holen.mockRejectedValue(new Error('Server nicht erreichbar'));
		const screen = render(BestellHistorie);

		const meldung = await screen.findByRole('alert');
		expect(meldung.textContent).toContain('Bestellhistorie nicht geladen');
		expect(screen.queryByText(/Lade Bestellhistorie/)).toBeNull();
		expect(screen.queryByText(/Noch keine Bestellungen aufgegeben/)).toBeNull();

		holen.mockReset();
		holen.mockImplementation(async (url) =>
			String(url).includes('uebersicht') ? SUMMEN : [bestellung('Buchhandlung Nord', 'land')]
		);
		await fireEvent.click(screen.getByRole('button', { name: 'Erneut versuchen' }));
		expect(await screen.findByText('Buchhandlung Nord')).toBeTruthy();
	});

	// Zwei Filterwechsel kurz nacheinander: Es gilt die Antwort auf den jüngsten, auch wenn
	// die ältere später eintrifft.
	it('lässt eine späte Antwort auf den vorigen Filter die Liste nicht ersetzen', async () => {
		/** @type {(liste: any[]) => void} */
		let landFertig = () => {};
		holen.mockImplementation((url) => {
			const adresse = String(url);
			if (adresse.includes('uebersicht')) return Promise.resolve(SUMMEN);
			if (adresse.includes('mittel=land')) return new Promise((fertig) => (landFertig = fertig));
			if (adresse.includes('mittel=schultraeger'))
				return Promise.resolve([bestellung('Buchhandlung Süd', 'schultraeger')]);
			return Promise.resolve([bestellung('Buchhandlung Nord', 'land')]);
		});
		const screen = render(BestellHistorie);
		await screen.findByText('Buchhandlung Nord');

		const filter = screen.getByRole('combobox');
		await fireEvent.click(filter);
		await fireEvent.click(screen.getByRole('option', { name: 'Lernmittelfreiheit (Land)' }));
		await fireEvent.click(filter);
		await fireEvent.click(screen.getByRole('option', { name: 'Schülerbücherei (Schulträger)' }));
		await screen.findByText('Buchhandlung Süd');

		landFertig([bestellung('Späte Antwort', 'land')]);
		await waitFor(() => expect(holen.mock.calls.length).toBeGreaterThanOrEqual(6));
		await Promise.resolve();

		expect(screen.queryByText('Späte Antwort')).toBeNull();
		expect(screen.getByText('Buchhandlung Süd')).toBeTruthy();
	});
});

// Die Zeile unter der Liste sagt, ob sie alles zeigt. Beim gefilterten Blick zählt die Zahl des
// Topfes: Sonst stünde unter einer vollständigen Liste „Neueste 1 von 137“.
describe('BestellHistorie: die Liste zeigt nicht alles', () => {
	beforeEach(() => vi.clearAllMocks());

	it('zählt ohne Filter alle Bestellungen, mit Filter die des Topfes', async () => {
		holen.mockImplementation(async (url) =>
			String(url).includes('uebersicht')
				? { ...SUMMEN, gesamt: 137, nach_mittel: [{ mittel: 'land', gesamt: 1 }] }
				: [bestellung('Buchhandlung Nord', 'land')]
		);
		const screen = render(BestellHistorie);
		expect(await screen.findByText(/Neueste 1 von 137 Bestellungen/)).toBeTruthy();

		await fireEvent.click(screen.getByRole('combobox'));
		await fireEvent.click(screen.getByRole('option', { name: 'Lernmittelfreiheit (Land)' }));

		await waitFor(() => expect(screen.queryByText(/Neueste 1 von/)).toBeNull());
	});
});

// Bestätigt ein Händler, während die Liste offen ist, wechselt der Chip ohne Zutun.
describe('BestellHistorie: der Stand kommt, während die Liste offen ist', () => {
	const wartet = { ...bestellung('Buchhandlung Nord', 'land'), mit_bestaetigung: true };
	const bestaetigt = { ...wartet, bestaetigt_am: '2026-10-12T10:00:00Z' };
	/** @param {any[]} liste @param {object} [summen] */
	const antworte = (liste, summen = SUMMEN) =>
		holen.mockImplementation(async (url) => (String(url).includes('uebersicht') ? summen : liste));
	const listenAbrufe = () =>
		holen.mock.calls.filter(
			([url]) => String(url).startsWith('/api/bestellhistorie?') || url === '/api/bestellhistorie'
		).length;

	beforeEach(() => {
		vi.clearAllMocks();
		leitung.hoerer.clear();
	});

	it('wechselt den Chip und die Zahl im Kopf, ohne die Liste auszutauschen', async () => {
		antworte([wartet], { ...SUMMEN, offene_bestaetigungen: 1 });
		const screen = render(BestellHistorie);
		await screen.findByText('Wartet auf Händler');
		expect(
			screen.getByText('1 Bestellung wartet noch auf die Bestätigung des Händlers')
		).toBeTruthy();

		antworte([bestaetigt]);
		leitung.sende('bestellstand');

		expect(screen.queryByText(/Lade Bestellhistorie/)).toBeNull();
		expect(await screen.findByText('Bestätigt · 12.10.')).toBeTruthy();
		expect(screen.queryByText('Wartet auf Händler')).toBeNull();
		expect(screen.queryByText(/wartet noch/)).toBeNull();
	});

	// Die Liste zeigt weiter, was sie weiß, und apiGet nennt den Fehler. Ein Kasten „nicht geladen"
	// anstelle der Liste, nur weil das Auffrischen im Hintergrund scheiterte, nähme die Arbeit weg.
	it('scheitert das Auffrischen, bleibt die Liste stehen', async () => {
		antworte([wartet]);
		const screen = render(BestellHistorie);
		await screen.findByText('Buchhandlung Nord');

		holen.mockRejectedValue(new Error('Server nicht erreichbar'));
		leitung.sende('bestellstand');
		await new Promise((fertig) => setTimeout(fertig));

		expect(screen.getByText('Buchhandlung Nord')).toBeTruthy();
		expect(screen.queryByRole('alert')).toBeNull();
	});

	// Unter dem neuen Filter stünde sonst die Liste des vorigen Topfes.
	it('scheitert der Wechsel des Filters, steht die alte Liste nicht unter dem neuen', async () => {
		antworte([wartet]);
		const screen = render(BestellHistorie);
		await screen.findByText('Buchhandlung Nord');

		holen.mockRejectedValue(new Error('Server nicht erreichbar'));
		await fireEvent.click(screen.getByRole('combobox'));
		await fireEvent.click(screen.getByRole('option', { name: 'Lernmittelfreiheit (Land)' }));

		expect(await screen.findByRole('alert')).toBeTruthy();
		expect(screen.queryByText('Buchhandlung Nord')).toBeNull();
	});

	it('hinter einer geöffneten Bestellung lädt nur sie neu, der Rückweg dann die Liste', async () => {
		antworte([wartet]);
		const screen = render(BestellHistorie);
		await screen.findByText('Buchhandlung Nord');
		const detail = { ...wartet, positionen: [], exemplare: [], link_aktiv: true };
		holen.mockImplementation(async (url) =>
			String(url) === `/api/bestellhistorie/${wartet.id}` ? detail : [wartet]
		);
		await fireEvent.click(
			screen.getByRole('button', { name: /Buchhandlung Nord.*Bestellung öffnen/ })
		);
		await screen.findByText('Warten auf den Händler');
		const liste = listenAbrufe();
		const alle = holen.mock.calls.length;

		leitung.sende('bestellstand');

		await waitFor(() => expect(holen.mock.calls.length).toBe(alle + 1));
		expect(holen.mock.calls.at(-1)?.[0]).toBe(`/api/bestellhistorie/${wartet.id}`);
		expect(listenAbrufe()).toBe(liste);
	});

	it('hört nach dem Verlassen nicht mehr zu', async () => {
		antworte([wartet]);
		const screen = render(BestellHistorie);
		await screen.findByText('Buchhandlung Nord');
		expect(leitung.hoerer.get('bestellstand')?.size).toBe(1);

		screen.unmount();

		expect(leitung.hoerer.get('bestellstand')?.size).toBe(0);
	});
});
