import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import BedarfZeile from './BedarfZeile.svelte';
import Bestellbedarf from './Bestellbedarf.svelte';
import { bestellStore } from '../../stores/bestellStore.svelte.js';

// Der Bestellbedarf zählt am Buch (docs/OFFEN.md 4.18, Stufe 3): Gehört ein Titel zu einem
// Buch mit mehreren Auflagen, ist die Zeile das Buch — Titel und ISBN der neuesten Auflage,
// die Summe als Zahl, darunter die Auflagen einzeln (GET /api/bestellungen,
// repository/bestellbedarf.go).

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
			screen.getByText('Bestand aus 2 Auflagen: 4. Aufl. · 2023 (1), 3. Aufl. · 2019 (2)')
		).toBeTruthy();
	});

	it('bleibt bei einem Titel ohne andere Auflage, wie sie war', () => {
		const screen = render(BedarfZeile, { r: einzeln, onAddToCart: vi.fn() });
		expect(screen.queryByText(/Bestand aus/)).toBeNull();
	});

	it('legt die Zeile des Buchs mit der fehlenden Menge in den Warenkorb — die neueste Auflage', async () => {
		const onAddToCart = vi.fn();
		const screen = render(BedarfZeile, { r: buch, onAddToCart });
		expect(screen.getByTitle('Bestand / Bedarf im nächsten Schuljahr').textContent?.trim()).toBe(
			'3/5'
		);
		await fireEvent.click(
			screen.getByRole('button', {
				name: '2 Exemplare von Lambacher Schweizer 7 zur Bestellung hinzufügen'
			})
		);
		expect(onAddToCart).toHaveBeenCalledWith(buch, 2);
	});
});

describe('Bestellbedarf: Filter', () => {
	it('findet das Buch auch über die ISBN der alten Auflage', async () => {
		const screen = render(Bestellbedarf, {
			vorschlaege: [buch, einzeln],
			onAddToCart: vi.fn()
		});
		await fireEvent.input(screen.getByLabelText('Bestellvorschläge filtern'), {
			target: { value: '9783120000019' }
		});
		expect(screen.getByText('Lambacher Schweizer 7')).toBeTruthy();
		expect(screen.queryByText('Physik 8')).toBeNull();
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
			screen.getByText(
				'3 Exemplare fehlen für das nächste Schuljahr, gerechnet mit den Schulbüchern von 540 der 600 Kinder.'
			)
		).toBeTruthy();
	});

	it('sagt „1 Exemplar fehlt" in der Einzahl', () => {
		const screen = render(Bestellbedarf, { vorschlaege: [einzeln], onAddToCart: vi.fn() });
		expect(
			screen.getByText(
				'1 Exemplar fehlt für das nächste Schuljahr, gerechnet mit den Schulbüchern von 540 der 600 Kinder.'
			)
		).toBeTruthy();
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
