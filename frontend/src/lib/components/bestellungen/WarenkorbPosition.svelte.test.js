import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, fireEvent, screen } from '@testing-library/svelte';

vi.mock('../../apiFetch.js', async (original) => ({
	.../** @type {any} */ (await original()),
	apiGet: vi.fn(async () => []),
	apiPost: vi.fn(async () => ({})),
	apiPut: vi.fn(async () => ({})),
	apiDelete: vi.fn(async () => ({}))
}));
vi.mock('../../stores/snackbars.svelte.js', () => ({ snackbars: { addToast: vi.fn() } }));

import { bestellStore } from '../../stores/bestellStore.svelte.js';
import WarenkorbPosition from './WarenkorbPosition.svelte';

// Unter 1 geht die Menge einer Position nicht; entfernt wird sie über das Kreuz. Der Knopf
// „Menge verringern" zeigt das: Er ist bei 1 gesperrt, statt ohne Wirkung zu bleiben.

/** @param {number} menge */
function position(menge) {
	bestellStore.cart = [
		{
			id: 't-1',
			titel: 'Mathematik 7',
			autor: 'Autor',
			isbn: '9789991940014',
			verlag: '',
			cover_url: '',
			menge,
			preis: 0,
			preis_vorschlag: 0,
			generate_barcodes: false,
			ist_lernmittel: true,
			mittel: 'land'
		}
	];
	// Die Position aus dem Speicher, nicht das Objekt von oben: Nur sie ist reaktiv.
	const { getByRole } = render(WarenkorbPosition, { position: bestellStore.cart[0] });
	return {
		minus: /** @type {HTMLButtonElement} */ (getByRole('button', { name: 'Menge verringern' })),
		plus: /** @type {HTMLButtonElement} */ (getByRole('button', { name: 'Menge erhöhen' }))
	};
}

describe('WarenkorbPosition: die Menge', () => {
	beforeEach(() => {
		bestellStore.cart = [];
	});

	it('sperrt „Menge verringern" bei 1', () => {
		const { minus, plus } = position(1);
		expect(minus.disabled).toBe(true);
		expect(plus.disabled).toBe(false);
	});

	it('gibt den Knopf über 1 frei und sperrt ihn wieder, sobald die Menge bei 1 ankommt', async () => {
		const { minus, plus } = position(1);

		await fireEvent.click(plus);
		expect(bestellStore.cart[0].menge).toBe(2);
		expect(minus.disabled).toBe(false);

		await fireEvent.click(minus);
		expect(bestellStore.cart[0].menge).toBe(1);
		expect(minus.disabled).toBe(true);
	});

	it('eine Position mit größerer Menge lässt sich verringern', async () => {
		const { minus } = position(3);
		expect(minus.disabled).toBe(false);
		await fireEvent.click(minus);
		expect(bestellStore.cart[0].menge).toBe(2);
	});
});

// Das Feld „Topf" zeigt, in welchem Topf die Position liegt, und schiebt sie in einen anderen: ein
// falsch gekennzeichnetes Schulbuch in die Schülerbücherei, eine Spende in die Bestellung ohne Topf.
describe('WarenkorbPosition: der Topf', () => {
	beforeEach(() => {
		bestellStore.cart = [];
	});

	it('zeigt den Topf der Position und schiebt sie über das Feld in die Spende', async () => {
		position(1);
		const feld = screen.getByRole('combobox', { name: 'Topf von Mathematik 7' });
		expect(feld.textContent).toContain('Lernmittelfreiheit');

		await fireEvent.click(feld);
		const toepfe = screen.getAllByRole('option').map((o) => o.textContent ?? '');
		expect(toepfe).toHaveLength(3);
		['Lernmittelfreiheit', 'Schülerbücherei', 'Spende'].forEach((wort, i) =>
			expect(toepfe[i]).toContain(wort)
		);

		await fireEvent.click(screen.getByRole('option', { name: /^Spende/ }));
		expect(bestellStore.cart[0].mittel).toBe('spende');
	});
});

// Steht das Buch im Bestellbedarf, nennt die Position, wie viele fehlen, auch wenn eine ältere
// Auflage bestellt wird; ohne Zeile im Bestellbedarf steht keine Marke.
describe('WarenkorbPosition: die Marke „Bestellbedarf"', () => {
	beforeEach(() => {
		bestellStore.cart = [];
		bestellStore.recommendations = [];
	});

	it('nennt die fehlende Menge des Buchs, zu dem die Position gehört', () => {
		bestellStore.recommendations = [
			{ id: 't-neu', isbn: '9789991940021', fehlen: 18, auflagen: [{ id: 't-1', isbn: '' }] }
		];
		position(5);
		expect(screen.getByText('Bestellbedarf 18')).toBeTruthy();
	});

	it('zeigt keine Marke für einen Titel ohne Bestellbedarf', () => {
		bestellStore.recommendations = [{ id: 't-anders', isbn: '9789991940038', fehlen: 3 }];
		position(1);
		expect(screen.queryByText(/^Bestellbedarf/)).toBeNull();
	});
});
