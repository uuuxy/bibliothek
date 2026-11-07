import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';

const leitung = vi.hoisted(() => ({
	/** @type {Map<string, Set<() => void>>} */
	hoerer: new Map(),
	/** @param {string} name */
	sende(name) {
		for (const holeNeu of this.hoerer.get(name) ?? []) holeNeu();
	}
}));
vi.mock('../../liveEvents.js', async (importOriginal) => ({
	.../** @type {any} */ (await importOriginal()),
	abonniereNeuladen: (/** @type {string} */ name, /** @type {() => void} */ holeNeu) => {
		if (!leitung.hoerer.has(name)) leitung.hoerer.set(name, new Set());
		leitung.hoerer.get(name)?.add(holeNeu);
		return () => leitung.hoerer.get(name)?.delete(holeNeu);
	}
}));
vi.mock('../../apiFetch.js', async (importOriginal) => ({
	.../** @type {any} */ (await importOriginal()),
	apiGet: vi.fn(async () => [])
}));

import { apiGet } from '../../apiFetch.js';
import { bestellStore } from '../../stores/bestellStore.svelte.js';
import Wareneingang from './Wareneingang.svelte';

const holen = vi.mocked(apiGet);
const offen = { incomingShipments: [], onReceived: vi.fn() };
/** @param {string[]} exemplare @param {object} [stand] */
const gruppe = (exemplare, stand = {}) => ({
	id: 'b1',
	supplierName: 'Händler',
	date: '30.09.2026',
	mit_bestaetigung: true,
	items: [
		{
			titel_id: 't1',
			titel: 'Titel A',
			isbn: '',
			cover_url: '',
			menge: exemplare.length,
			exemplar_ids: exemplare
		}
	],
	...stand
});

describe('Wareneingang: der Zulauf bleibt aktuell', () => {
	beforeEach(() => {
		holen.mockReset();
		holen.mockResolvedValue([]);
		leitung.hoerer.clear();
		bestellStore.incomingShipments = [];
	});

	// Der Stand im Kopf einer Bestellung ändert sich außerhalb dieses Reiters. Das Laden des
	// Bestellwesens gilt 60 Sekunden als frisch; ohne eigenes Laden stünde hier noch „Wartet auf
	// Händler", wenn die Historie die Zusage eben nachgetragen hat.
	it('holt den Zulauf beim Öffnen frisch', async () => {
		const liste = [gruppe([])];
		holen.mockResolvedValue(liste);

		render(Wareneingang, offen);

		expect(holen).toHaveBeenCalledWith('/api/bestellungen/zulauf');
		await vi.waitFor(() => expect(bestellStore.incomingShipments).toEqual(liste));
	});

	// Bestätigt der Händler, während der Reiter offen ist, wechselt der Chip. Die Markierung gehört
	// der laufenden Arbeit und überlebt das Nachladen.
	it('holt den Zulauf neu, wenn bestätigt wird, und die Markierung bleibt', async () => {
		holen.mockResolvedValue([gruppe(['e1', 'e2'])]);
		const screen = render(Wareneingang, { ...offen, incomingShipments: [gruppe(['e1', 'e2'])] });
		await fireEvent.click(screen.getByRole('checkbox', { name: 'Titel A auswählen' }));
		expect(screen.getByText('2 Exemplare markiert')).toBeTruthy();
		const anfragen = holen.mock.calls.length;

		const bestaetigt = [gruppe(['e1', 'e2'], { bestaetigt_am: '2026-10-12T10:00:00Z' })];
		holen.mockResolvedValue(bestaetigt);
		leitung.sende('bestellstand');
		await vi.waitFor(() => expect(bestellStore.incomingShipments).toEqual(bestaetigt));
		await screen.rerender({ incomingShipments: bestellStore.incomingShipments });

		expect(holen.mock.calls).toHaveLength(anfragen + 1);
		expect(screen.getByText('Bestätigt · 12.10.')).toBeTruthy();
		expect(screen.getByText('2 Exemplare markiert')).toBeTruthy();
		expect(screen.getByRole('checkbox', { name: 'Titel A auswählen' })).toHaveProperty(
			'checked',
			true
		);
	});

	// Hat ein anderer Arbeitsplatz eingebucht, fällt das Exemplar beim Nachladen aus der Liste. Die
	// Leiste nennte sonst eine Zahl, die man nirgends sieht, und „Alle auswählen" zählte falsch.
	it('was aus der Liste fällt, bleibt nicht markiert', async () => {
		const screen = render(Wareneingang, { ...offen, incomingShipments: [gruppe(['e1', 'e2'])] });
		await fireEvent.click(screen.getByRole('checkbox', { name: 'Titel A auswählen' }));

		await screen.rerender({ incomingShipments: [gruppe(['e1'])] });
		expect(screen.getByText('1 Exemplar markiert')).toBeTruthy();

		await screen.rerender({ incomingShipments: [] });
		expect(screen.queryByText(/markiert/)).toBeNull();
	});

	// Bucht ein anderer Platz die Zeile ein, auf der der Fokus steht, fällt er auf den Anfang des Inhalts
	// und nicht vor die Navigation. Auf die Zeile darunter darf er nicht: Die Leertaste markierte sie.
	it('fällt die Zeile mit dem Fokus aus der Liste, rückt er auf den Anfang des Inhalts', async () => {
		const anfang = document.createElement('main');
		anfang.id = 'hauptinhalt';
		anfang.tabIndex = -1;
		document.body.append(anfang);
		try {
			const screen = render(Wareneingang, { ...offen, incomingShipments: [gruppe(['e1', 'e2'])] });
			screen.getByRole('checkbox', { name: 'Titel A auswählen' }).focus();

			await screen.rerender({ incomingShipments: [] });

			expect(document.activeElement).toBe(anfang);
		} finally {
			anfang.remove();
		}
	});

	// Danach holt die Leitung nach (liveEvents.js); die Ansicht sagt ihr, ob die Antwort kam.
	it('meldet der Leitung, ob die Antwort kam', async () => {
		render(Wareneingang, offen);
		const holeNeu = () => [...(leitung.hoerer.get('bestellstand') ?? [])][0]();

		expect(await holeNeu()).toBe(true);

		holen.mockRejectedValue(new Error('Server nicht erreichbar'));
		expect(await holeNeu()).toBe(false);
	});

	it('hört nach dem Schließen des Reiters nicht mehr zu', () => {
		const screen = render(Wareneingang, offen);
		expect(leitung.hoerer.get('bestellstand')?.size).toBe(1);

		screen.unmount();

		expect(leitung.hoerer.get('bestellstand')?.size).toBe(0);
	});
});
