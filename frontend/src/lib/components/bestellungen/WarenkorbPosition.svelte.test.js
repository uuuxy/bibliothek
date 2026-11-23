import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';

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

// Eine Position lässt sich in jeden anderen Topf schieben: ein falsch gekennzeichnetes Schulbuch
// in die Schülerbücherei, eine Spende in die Bestellung ohne Topf. Jeder Knopf nennt sein Ziel.
describe('WarenkorbPosition: der Topf', () => {
	beforeEach(() => {
		bestellStore.cart = [];
	});

	it('bietet beide anderen Töpfe an und schiebt die Position in die Spende', async () => {
		position(1);
		// position() hat die Zeile schon gezeichnet; gezählt wird an ihr.
		const ziele = [...document.querySelectorAll('button[aria-label*="in die Bestellung"]')].map(
			(k) => k.getAttribute('aria-label')
		);
		expect(ziele).toEqual([
			'Mathematik 7 in die Bestellung „Schülerbücherei“ verschieben',
			'Mathematik 7 in die Bestellung „Spende“ verschieben'
		]);

		await fireEvent.click(
			/** @type {HTMLButtonElement} */ (
				document.querySelector('button[aria-label$="„Spende“ verschieben"]')
			)
		);
		expect(bestellStore.cart[0].mittel).toBe('spende');
	});
});
