import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import { tick } from 'svelte';
import BedarfZeile from './BedarfZeile.svelte';
import Bestellbedarf from './Bestellbedarf.svelte';
import { bestellStore } from '../../stores/bestellStore.svelte.js';

// Der Bestellbedarf zählt am Buch: Gehört ein Titel zu einem
// Buch mit mehreren Auflagen, ist die Zeile das Buch — Titel und ISBN der neuesten Auflage,
// die Summe als Zahl, darunter die Auflagen einzeln (GET /api/bestellungen,
// internal/repository/bestellbedarf.go).

const buch = {
	id: 'neu',
	titel: 'Lambacher Schweizer 7',
	isbn: '9783120000026',
	verlag: 'Klett',
	gesamt_bestand: 3,
	bedarf: 5,
	fehlen: 2,
	auflagen: [
		{
			id: 'neu',
			auflage: '4. Aufl.',
			isbn: '9783120000026',
			erscheinungsjahr: 2023,
			gesamt_bestand: 1
		},
		{
			id: 'alt',
			auflage: '3. Aufl.',
			isbn: '9783120000019',
			erscheinungsjahr: 2019,
			gesamt_bestand: 2
		}
	]
};
const einzeln = {
	id: 'p8',
	titel: 'Physik 8',
	isbn: '9783120000033',
	gesamt_bestand: 1,
	bedarf: 2,
	fehlen: 1
};

describe('BedarfZeile', () => {
	it('schlüsselt ein Buch mit mehreren Auflagen auf', () => {
		const screen = render(BedarfZeile, { r: buch, onAddToCart: vi.fn() });
		expect(
			screen.getByText(
				'Bestand aus 2 Auflagen: 1 Exemplar der 4. Aufl. · 2023, 2 Exemplare der 3. Aufl. · 2019'
			)
		).toBeTruthy();
	});

	it('bleibt bei einem Titel ohne andere Auflage, wie sie war', () => {
		const screen = render(BedarfZeile, { r: einzeln, onAddToCart: vi.fn() });
		expect(screen.queryByText(/Bestand aus/)).toBeNull();
	});

	it('legt die Zeile des Buchs mit der fehlenden Menge in den Warenkorb — die neueste Auflage', async () => {
		const onAddToCart = vi.fn();
		const screen = render(BedarfZeile, { r: buch, onAddToCart });
		expect(
			screen.container
				.querySelector('[data-tip="Bestand, gemessen am Bedarf im nächsten Schuljahr"]')
				?.textContent?.replace(/\s+/g, ' ')
				.trim()
		).toBe('Bestand, gemessen am Bedarf im nächsten Schuljahr: 3 von 5');
		await fireEvent.click(
			screen.getByRole('button', {
				name: '2 hinzufügen: Lambacher Schweizer 7'
			})
		);
		expect(onAddToCart).toHaveBeenCalledWith(buch, 2);
	});
});

/** Eine Position im Warenkorb, wie bestellStore.addToCart sie anlegt. @param {string} id @param {string} isbn */
const position = (id, isbn = '') => ({
	id,
	titel: id,
	autor: '',
	isbn,
	verlag: '',
	cover_url: '',
	menge: 1,
	preis: 0,
	preis_vorschlag: 0,
	generate_barcodes: false,
	ist_lernmittel: true,
	mittel: /** @type {const} */ ('land')
});

// Bestellbedarf und Bestellung stehen in einer Liste: Was in der Bestellung steht, fällt aus dem
// Bestellbedarf heraus, auch in einer anderen Auflage desselben Buchs.
describe('Bestellbedarf: was schon in der Bestellung steht', () => {
	beforeEach(() => {
		bestellStore.nichtGeladen = { bedarf: '' };
		bestellStore.bedarfGrundlage = { kinder: 600, kinder_mit_schulbuch: 540, ausgeschaltet: false };
	});
	afterEach(() => {
		bestellStore.cart = [];
	});

	it('lässt das Buch weg, wenn die alte Auflage in der Bestellung steht, und zählt nur den Rest', () => {
		bestellStore.cart = [position('alt')];
		const screen = render(Bestellbedarf, { vorschlaege: [buch, einzeln], onAddToCart: vi.fn() });
		expect(screen.queryByText('Lambacher Schweizer 7')).toBeNull();
		expect(screen.getByText('Physik 8')).toBeTruthy();
		expect(
			screen.getByText('· 1 weiterer Titel · 1 Exemplar fehlt noch für das nächste Schuljahr')
		).toBeTruthy();
	});

	it('sagt, wenn der ganze Bestellbedarf in der Bestellung steht', () => {
		bestellStore.cart = [position('neu'), position('anders', '9783120000033')];
		const screen = render(Bestellbedarf, { vorschlaege: [buch, einzeln], onAddToCart: vi.fn() });
		expect(screen.getByText('Der ganze Bestellbedarf steht in der Bestellung')).toBeTruthy();
		expect(screen.queryByRole('listitem')).toBeNull();
	});

	// Der Knopf verschwindet mit seiner Zeile; der Fokus fiele sonst auf den Anfang der Seite.
	it('rückt den Fokus nach „hinzufügen" auf den Knopf der nächsten Zeile', async () => {
		const screen = render(Bestellbedarf, {
			vorschlaege: [buch, einzeln],
			onAddToCart: (/** @type {any} */ r) => bestellStore.cart.push(position(r.id, r.isbn))
		});
		const erster = screen.getByRole('button', { name: '2 hinzufügen: Lambacher Schweizer 7' });
		erster.focus();
		await fireEvent.click(erster);
		await tick();
		expect(document.activeElement).toBe(
			screen.getByRole('button', { name: '1 hinzufügen: Physik 8' })
		);
	});
});

describe('BedarfZeile: Neue Auflage bestellen', () => {
	it('bietet die Aktion im Menü der Zeile an und reicht die Zeile weiter', async () => {
		const onNeueAuflage = vi.fn();
		const screen = render(BedarfZeile, { r: einzeln, onAddToCart: vi.fn(), onNeueAuflage });
		await fireEvent.click(screen.getByRole('button', { name: 'Weitere Aktionen zu Physik 8' }));
		await fireEvent.click(await screen.findByRole('menuitem', { name: /Neue Auflage bestellen/ }));
		expect(onNeueAuflage).toHaveBeenCalledWith(einzeln);
	});

	it('zeigt kein Menü, wenn niemand die Aktion anbietet', () => {
		const screen = render(BedarfZeile, { r: einzeln, onAddToCart: vi.fn() });
		expect(screen.queryByRole('button', { name: /Weitere Aktionen/ })).toBeNull();
	});
});

// Im Kopf steht, wie viele Exemplare fehlen und worauf die Rechnung fußt. Ist die Liste leer,
// sagt die Seite, warum: ausgeschaltet, noch kein Schulbuch ausgegeben, oder es fehlt nichts.
describe('Bestellbedarf: Kopf und Leerzustand', () => {
	beforeEach(() => {
		bestellStore.nichtGeladen = { bedarf: '' };
		bestellStore.bedarfGrundlage = { kinder: 600, kinder_mit_schulbuch: 540, ausgeschaltet: false };
	});

	it('zeigt vor der ersten Antwort den Ladekreis statt eines Leerzustands', () => {
		bestellStore.nichtGeladen = {};
		const screen = render(Bestellbedarf, { vorschlaege: [], onAddToCart: vi.fn() });
		expect(screen.getByRole('progressbar', { name: 'Bestellbedarf lädt' })).toBeTruthy();
		expect(screen.queryByText('Noch keine Grundlage')).toBeNull();
	});

	it('nennt die fehlenden Exemplare und die Grundlage', () => {
		const screen = render(Bestellbedarf, { vorschlaege: [buch, einzeln], onAddToCart: vi.fn() });
		expect(
			screen.getByText('· 2 Titel · 3 Exemplare fehlen für das nächste Schuljahr')
		).toBeTruthy();
		expect(screen.getByText('Gerechnet mit den Schulbüchern von 540 der 600 Kinder')).toBeTruthy();
	});

	it('sagt „1 Exemplar fehlt" in der Einzahl', () => {
		const screen = render(Bestellbedarf, { vorschlaege: [einzeln], onAddToCart: vi.fn() });
		expect(screen.getByText('· 1 Titel · 1 Exemplar fehlt für das nächste Schuljahr')).toBeTruthy();
	});

	it('sagt bei leerer Liste, dass nichts fehlt', () => {
		const screen = render(Bestellbedarf, { vorschlaege: [], onAddToCart: vi.fn() });
		expect(screen.getByText('Kein Schulbuch fehlt')).toBeTruthy();
	});

	it('behauptet ohne ausgegebene Schulbücher nicht, es fehle nichts', () => {
		bestellStore.bedarfGrundlage = { kinder: 600, kinder_mit_schulbuch: 0, ausgeschaltet: false };
		const screen = render(Bestellbedarf, { vorschlaege: [], onAddToCart: vi.fn() });
		expect(screen.getByText('Noch keine Grundlage')).toBeTruthy();
		expect(screen.queryByText('Kein Schulbuch fehlt')).toBeNull();
	});

	it('sagt, wenn der Bestellbedarf ausgeschaltet ist', () => {
		bestellStore.bedarfGrundlage = { kinder: 0, kinder_mit_schulbuch: 0, ausgeschaltet: true };
		const screen = render(Bestellbedarf, { vorschlaege: [], onAddToCart: vi.fn() });
		expect(screen.getByText('Bestellbedarf ausgeschaltet')).toBeTruthy();
		expect(screen.queryByText('Kein Schulbuch fehlt')).toBeNull();
	});
});
