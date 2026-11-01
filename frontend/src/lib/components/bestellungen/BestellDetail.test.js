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
	apiGet: vi.fn(),
	apiPut: vi.fn()
}));

import { apiGet, apiPut } from '../../apiFetch.js';
import { authStore } from '../../stores/authStore.svelte.js';
import BestellDetail from './BestellDetail.svelte';

const holen = vi.mocked(apiGet);
const WARTET = {
	id: 'b1',
	lieferant_name: 'Naacher',
	lieferant_email: 'naacher@example.invalid',
	bestelldatum: '2026-10-01T08:00:00Z',
	mittel: 'land',
	anzahl_exemplare: 2,
	gesamtbetrag: 0,
	mit_bestaetigung: true,
	link_aktiv: true,
	positionen: [],
	exemplare: []
};
const BESTAETIGT = {
	...WARTET,
	bestaetigt_am: '2026-10-12T10:00:00Z',
	bestaetigt_durch: 'lieferant'
};
const QUITTUNG = 'Der Lieferant hat die Bestellung bestätigt';

async function geoeffnet() {
	const screen = render(BestellDetail, { bestellungId: 'b1', onBack: vi.fn() });
	await screen.findByText('Warten auf den Händler');
	return screen;
}

// Die geöffnete Bestellung holt ihren Stand neu, wenn ein Händler bestätigt. Dabei steht vielleicht
// gerade ein Link da, der nur einmal zu sehen ist, oder ein getippter Grund: Beides überlebt das.
describe('BestellDetail: der Stand kommt, während die Bestellung offen ist', () => {
	beforeEach(() => {
		vi.clearAllMocks();
		leitung.hoerer.clear();
		holen.mockResolvedValue(WARTET);
		authStore.currentUser = /** @type {any} */ ({ rolle: 'admin', permissions: ['*'] });
	});

	it('wechselt zur Quittung, ohne die Ansicht neu aufzubauen', async () => {
		const screen = await geoeffnet();

		holen.mockResolvedValue(BESTAETIGT);
		leitung.sende('bestellstand');

		expect(screen.queryByText('Lade Bestellung…')).toBeNull();
		expect(await screen.findByText(QUITTUNG)).toBeTruthy();
		expect(screen.queryByText('Warten auf den Händler')).toBeNull();
	});

	it('ein eben erzeugter Link bleibt stehen', async () => {
		vi.mocked(apiPut).mockResolvedValue({ link: 'https://schule.example/bestellung/neu' });
		const screen = await geoeffnet();
		await fireEvent.click(screen.getByRole('button', { name: 'Neuen Link erzeugen' }));
		const feld = /** @type {HTMLInputElement} */ (
			await screen.findByLabelText('Bestätigungs-Link')
		);
		const anfragen = holen.mock.calls.length;

		// Eine andere Bestellung ist bestätigt worden; diese hier wartet weiter.
		leitung.sende('bestellstand');

		await waitFor(() => expect(holen.mock.calls.length).toBe(anfragen + 1));
		await Promise.resolve();
		expect(/** @type {HTMLInputElement} */ (screen.getByLabelText('Bestätigungs-Link')).value).toBe(
			'https://schule.example/bestellung/neu'
		);
		expect(screen.getByLabelText('Bestätigungs-Link')).toBe(feld);
	});

	it('der getippte Grund im Dialog „Topf ändern" bleibt', async () => {
		const screen = await geoeffnet();
		await fireEvent.click(screen.getByRole('button', { name: 'Topf ändern' }));
		const grund = /** @type {HTMLInputElement} */ (screen.getByLabelText(/Grund der Korrektur/));
		await fireEvent.input(grund, { target: { value: 'falsch gekennzeichnet' } });
		const anfragen = holen.mock.calls.length;

		holen.mockResolvedValue(BESTAETIGT);
		leitung.sende('bestellstand');

		expect(await screen.findByText(QUITTUNG)).toBeTruthy();
		expect(holen.mock.calls.length).toBe(anfragen + 1);
		expect(grund.value).toBe('falsch gekennzeichnet');
	});

	// Kommt beim Nachladen ein anderer Topf, weil ein anderer Platz ihn geändert hat, liefe eine
	// Vorbelegung, die den Topf mitliest, noch einmal und nähme den getippten Grund mit.
	it('der getippte Grund bleibt auch, wenn das Nachladen einen anderen Topf bringt', async () => {
		const screen = await geoeffnet();
		await fireEvent.click(screen.getByRole('button', { name: 'Topf ändern' }));
		const grund = /** @type {HTMLInputElement} */ (screen.getByLabelText(/Grund der Korrektur/));
		await fireEvent.input(grund, { target: { value: 'falsch gekennzeichnet' } });

		holen.mockResolvedValue({ ...BESTAETIGT, mittel: 'schultraeger' });
		leitung.sende('bestellstand');

		expect(await screen.findByText(QUITTUNG)).toBeTruthy();
		expect(grund.value).toBe('falsch gekennzeichnet');
	});

	// Ein Signal und das Laden nach einer eigenen Aktion können sich überholen.
	it('eine ältere Antwort überschreibt die neuere nicht', async () => {
		const screen = await geoeffnet();

		/** @type {(wert: any) => void} */
		let aeltereAntwort = () => {};
		holen.mockReturnValueOnce(new Promise((fertig) => (aeltereAntwort = fertig)));
		leitung.sende('bestellstand');
		holen.mockResolvedValueOnce(BESTAETIGT);
		leitung.sende('bestellstand');
		await screen.findByText(QUITTUNG);

		aeltereAntwort(WARTET);
		await new Promise((fertig) => setTimeout(fertig));

		expect(screen.getByText(QUITTUNG)).toBeTruthy();
	});

	// Kommt das Signal, während die Bestellung noch zum ersten Mal lädt, zählt nur die jüngere Anfrage:
	// Die ältere darf weder den Ladezustand beenden noch mit ihrem Fehler den Inhalt verdecken.
	it('beim ersten Laden entscheidet die jüngere Anfrage', async () => {
		/** @type {(grund: Error) => void} */
		let aeltereScheitert = () => {};
		/** @type {(wert: any) => void} */
		let juengereAntwort = () => {};
		holen.mockReturnValueOnce(new Promise((_, nein) => (aeltereScheitert = nein)));
		holen.mockReturnValueOnce(new Promise((fertig) => (juengereAntwort = fertig)));
		const screen = render(BestellDetail, { bestellungId: 'b1', onBack: vi.fn() });
		leitung.sende('bestellstand');

		aeltereScheitert(new Error('Server nicht erreichbar'));
		await new Promise((fertig) => setTimeout(fertig));
		expect(screen.getByText('Lade Bestellung…')).toBeTruthy();
		expect(screen.queryByRole('alert')).toBeNull();

		juengereAntwort(WARTET);
		expect(await screen.findByText('Warten auf den Händler')).toBeTruthy();
		expect(screen.queryByRole('alert')).toBeNull();
	});

	it('scheitert das Auffrischen, bleibt die Ansicht stehen', async () => {
		const screen = await geoeffnet();

		holen.mockRejectedValue(new Error('Server nicht erreichbar'));
		leitung.sende('bestellstand');
		await new Promise((fertig) => setTimeout(fertig));

		expect(screen.getByText('Warten auf den Händler')).toBeTruthy();
		expect(screen.queryByRole('alert')).toBeNull();
	});

	// Der Kasten verschwindet samt dem Knopf, auf dem der Fokus steht. M3, Flow: „Focus is particularly
	// important when an element is activated by the user or the user changes context."
	it('bestätigt der Händler, rückt der Fokus aus dem Kasten auf die Quittung', async () => {
		const screen = await geoeffnet();
		screen.getByRole('button', { name: 'Kleine Etiketten' }).focus();

		holen.mockResolvedValue(BESTAETIGT);
		leitung.sende('bestellstand');

		const quittung = await screen.findByText(QUITTUNG);
		const region = quittung.closest('[role="status"]');
		await waitFor(() => expect(region?.contains(document.activeElement)).toBe(true));
	});

	it('steht der Fokus woanders, bleibt er dort', async () => {
		const screen = await geoeffnet();
		const zurueck = screen.getByRole('button', { name: 'Zurück zur Bestellhistorie' });
		zurueck.focus();

		holen.mockResolvedValue(BESTAETIGT);
		leitung.sende('bestellstand');
		await screen.findByText(QUITTUNG);

		expect(document.activeElement).toBe(zurueck);
	});

	// Der Block zur gescheiterten Mail verschwindet, wenn der Händler bestätigt, und ohne ihn hat der
	// Fokus keinen Nachfolger: Er rückt auf den Anfang des Inhalts statt vor die Navigation.
	it('verschwindet der Block zur gescheiterten Mail, rückt der Fokus auf den Inhalt', async () => {
		const anfang = document.createElement('main');
		anfang.id = 'hauptinhalt';
		anfang.tabIndex = -1;
		document.body.append(anfang);
		try {
			holen.mockResolvedValue({ ...WARTET, mail_gescheitert_am: '2026-10-12T10:00:00Z' });
			const screen = await geoeffnet();
			screen.getByRole('button', { name: 'Erneut senden' }).focus();

			holen.mockResolvedValue(WARTET);
			leitung.sende('bestellstand');

			await waitFor(() =>
				expect(screen.queryByRole('button', { name: 'Erneut senden' })).toBeNull()
			);
			await waitFor(() => expect(document.activeElement).toBe(anfang));
		} finally {
			anfang.remove();
		}
	});

	// Danach holt die Leitung nach (liveEvents.js); die Ansicht sagt ihr, ob die Antwort kam.
	it('meldet der Leitung, ob die Antwort kam', async () => {
		await geoeffnet();
		const holeNeu = () => [...(leitung.hoerer.get('bestellstand') ?? [])][0]();

		expect(await holeNeu()).toBe(true);

		holen.mockRejectedValue(new Error('Server nicht erreichbar'));
		expect(await holeNeu()).toBe(false);
	});

	it('hört nach dem Schließen nicht mehr zu', async () => {
		const screen = await geoeffnet();
		expect(leitung.hoerer.get('bestellstand')?.size).toBe(1);

		screen.unmount();

		expect(leitung.hoerer.get('bestellstand')?.size).toBe(0);
	});
});
