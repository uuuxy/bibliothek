import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/svelte';
import MahnwesenReiter from './MahnwesenReiter.svelte';
import { apiFetch } from '../../apiFetch.js';
import { mahnwesenStore } from '../../stores/mahnwesen.svelte.js';
import { berechneMahnstufe } from './mahnungen.js';

vi.mock('../../apiFetch.js', async (importOriginal) => ({
	...(await importOriginal()),
	apiFetch: vi.fn()
}));

// Die Regel bleibt die echte; der zweite Fall verschiebt die Grenze, wie es eine Entscheidung
// der Schule täte. Liste und Reiter lesen dasselbe Modul.
vi.mock('./mahnungen.js', async (importOriginal) => {
	const echt = /** @type {typeof import('./mahnungen.js')} */ (await importOriginal());
	return { ...echt, berechneMahnstufe: vi.fn(echt.berechneMahnstufe) };
});
const echt = /** @type {typeof import('./mahnungen.js')} */ (
	await vi.importActual('./mahnungen.js')
);

/** Ein Kind mit einem Buch je Angabe in Tagen. @param {string} name @param {number[]} tage */
const kind = (name, ...tage) => ({
	schueler_id: name,
	name,
	medien: tage.map((t, i) => ({ ausleihe_id: `${name}${i}`, titel: 'Band', tage_ueberfaellig: t }))
});

// A ist seit heute fällig, B steht auf der Grenze, C einen Tag darüber; bei D und H zählt das
// längste Buch. I ist ehemalig und steht in der eigenen Gruppe am Ende.
const KLASSEN = [
	{
		klasse: '07a',
		schueler: [kind('A', 0), kind('B', 14), kind('C', 15), kind('D', 3, 20), kind('E', 5)]
	},
	{ klasse: '08b', schueler: [kind('F', 1), kind('G', 21), kind('H', 40, 2)] },
	{ klasse: 'Ehemalige', ehemalige: true, schueler: [kind('I', 7)] }
];

async function zeichne() {
	vi.mocked(apiFetch).mockResolvedValue(
		/** @type {any} */ ({ ok: true, json: async () => structuredClone({ klassen: KLASSEN }) })
	);
	mahnwesenStore.stufenFilter = 'Alle';
	await mahnwesenStore.lade();
	render(MahnwesenReiter);
}

/** Je Reiter die Zahl daran und die Kinder in der Liste nach dem Klick darauf. */
async function zahlUndListe() {
	/** @type {Record<string, { zahl: number, liste: number }>} */
	const ergebnis = {};
	for (const etikett of ['Alle', 'Akut fällig', 'Eskaliert']) {
		const reiter = screen.getByRole('tab', { name: new RegExp(`^${etikett}`) });
		const zahl = Number(reiter.textContent?.match(/\d+/)?.[0] ?? 0);
		await fireEvent.click(reiter);
		ergebnis[etikett] = { zahl, liste: mahnwesenStore.gefilterteSchueler.length };
	}
	return ergebnis;
}

describe('MahnwesenReiter: die Zahl am Reiter ist die Zahl der Kinder in der Liste', () => {
	beforeEach(() => {
		vi.mocked(berechneMahnstufe).mockImplementation(echt.berechneMahnstufe);
	});

	it('führt bis 14 Tage unter „Akut fällig“, darüber unter „Eskaliert“', async () => {
		await zeichne();
		expect(await zahlUndListe()).toEqual({
			Alle: { zahl: 9, liste: 9 },
			'Akut fällig': { zahl: 5, liste: 5 },
			Eskaliert: { zahl: 4, liste: 4 }
		});
	});

	// Die Grenze steht nur in berechneMahnstufe: Eine eigene Rechnung am Reiter bliebe bei 14.
	it('folgt einer anderen Grenze in berechneMahnstufe', async () => {
		vi.mocked(berechneMahnstufe).mockImplementation((t) => (t > 21 ? 'Mahnung' : '1. Erinnerung'));
		await zeichne();
		expect(await zahlUndListe()).toEqual({
			Alle: { zahl: 9, liste: 9 },
			'Akut fällig': { zahl: 8, liste: 8 },
			Eskaliert: { zahl: 1, liste: 1 }
		});
	});
});
