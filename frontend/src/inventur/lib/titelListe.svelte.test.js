import { describe, it, expect, vi, beforeEach } from 'vitest';

vi.mock('./admin_api.js', () => ({ holeBuecherListe: vi.fn() }));

import { holeBuecherListe } from './admin_api.js';
import { erstelleTitelListe } from './titelListe.svelte.js';
import { keineAenderungen, merkeAenderungen, mitAenderungen } from './titelListeAenderungen.js';

/** Eine Antwort, die der Test selbst freigibt. */
function spaeteAntwort() {
	/** @type {(wert: any[]) => void} */
	let gib = () => {};
	const versprechen = new Promise((fertig) => (gib = fertig));
	return { versprechen, gib };
}

// Die Liste lädt beim Öffnen der Seite; bei 13.000 Titeln dauert das. Was die Maske in dieser
// Zeit ändert, kennt die Antwort nicht.
describe('Titelliste: Änderungen während des Ladens', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('ein Titel, der während des Ladens gespeichert wurde, steht nach der Antwort oben', async () => {
		const antwort = spaeteAntwort();
		vi.mocked(holeBuecherListe).mockReturnValueOnce(/** @type {any} */ (antwort.versprechen));
		const liste = erstelleTitelListe();
		const laden = liste.lade();

		const neu = { id: 'neu', title: 'Eben gespeichert' };
		liste.buecher = [neu, ...liste.buecher];
		antwort.gib([
			{ id: 'a', title: 'Alt A' },
			{ id: 'b', title: 'Alt B' }
		]);
		await laden;

		expect(liste.buecher.map((b) => b.id)).toEqual(['neu', 'a', 'b']);
		expect(liste.wirdGeladen).toBe(false);
	});

	it('ein geänderter Titel behält seinen neuen Stand, ein gelöschter bleibt gelöscht', async () => {
		const erste = [
			{ id: 'a', title: 'Alt A' },
			{ id: 'b', title: 'Alt B' }
		];
		vi.mocked(holeBuecherListe).mockResolvedValueOnce(erste);
		const liste = erstelleTitelListe();
		await liste.lade();

		const antwort = spaeteAntwort();
		vi.mocked(holeBuecherListe).mockReturnValueOnce(/** @type {any} */ (antwort.versprechen));
		const laden = liste.lade();
		const geaendert = { id: 'a', title: 'Neu A' };
		liste.buecher = liste.buecher.map((b) => (b.id === 'a' ? geaendert : b));
		liste.buecher = liste.buecher.filter((b) => b.id !== 'b');
		antwort.gib([
			{ id: 'a', title: 'Alt A' },
			{ id: 'b', title: 'Alt B' },
			{ id: 'c', title: 'Alt C' }
		]);
		await laden;

		expect(liste.buecher).toEqual([geaendert, { id: 'c', title: 'Alt C' }]);
	});

	it('ohne laufenden Abruf gilt die Liste, wie sie gesetzt wurde, und der nächste Abruf ersetzt sie', async () => {
		vi.mocked(holeBuecherListe).mockResolvedValueOnce([{ id: 'a' }]);
		const liste = erstelleTitelListe();
		await liste.lade();
		liste.buecher = [{ id: 'neu' }, ...liste.buecher];
		expect(liste.buecher.map((b) => b.id)).toEqual(['neu', 'a']);

		vi.mocked(holeBuecherListe).mockResolvedValueOnce([{ id: 'a' }, { id: 'neu' }]);
		await liste.lade();
		expect(liste.buecher.map((b) => b.id)).toEqual(['a', 'neu']);
	});

	it('nur die Antwort des jüngsten Abrufs gilt', async () => {
		const alte = spaeteAntwort();
		vi.mocked(holeBuecherListe).mockReturnValueOnce(/** @type {any} */ (alte.versprechen));
		vi.mocked(holeBuecherListe).mockResolvedValueOnce([{ id: 'treffer' }]);
		const liste = erstelleTitelListe();
		const erster = liste.lade();
		await liste.lade();
		alte.gib([{ id: 'a' }, { id: 'b' }]);
		await erster;

		expect(liste.buecher.map((b) => b.id)).toEqual(['treffer']);
	});
});

describe('Titelliste: die Rechnung', () => {
	it('legt Neues in der Reihenfolge des Speicherns oben an, das Jüngste zuerst', () => {
		const aenderungen = keineAenderungen();
		const eins = { id: '1' };
		const zwei = { id: '2' };
		merkeAenderungen([], [eins], aenderungen);
		merkeAenderungen([eins], [zwei, eins], aenderungen);
		expect(mitAenderungen([{ id: 'a' }], aenderungen).map((b) => b.id)).toEqual(['2', '1', 'a']);
	});

	it('legt einen Titel, den die Antwort schon kennt, nicht doppelt an', () => {
		const aenderungen = keineAenderungen();
		const neu = { id: 'neu', title: 'von der Maske' };
		merkeAenderungen([], [neu], aenderungen);
		expect(mitAenderungen([{ id: 'a' }, { id: 'neu', title: 'vom Server' }], aenderungen)).toEqual([
			{ id: 'a' },
			neu
		]);
	});

	it('vergisst einen Titel, der während des Ladens angelegt und wieder gelöscht wurde', () => {
		const aenderungen = keineAenderungen();
		const neu = { id: 'neu' };
		merkeAenderungen([], [neu], aenderungen);
		merkeAenderungen([neu], [], aenderungen);
		expect(mitAenderungen([{ id: 'a' }], aenderungen)).toEqual([{ id: 'a' }]);
	});
});
