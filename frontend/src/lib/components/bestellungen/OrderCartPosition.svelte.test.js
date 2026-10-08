import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';

vi.mock('../../apiFetch.js', async (original) => ({
	.../** @type {any} */ (await original()),
	apiGet: vi.fn(async () => []),
	apiPost: vi.fn(async () => ({})),
	apiPut: vi.fn(async () => ({})),
	apiDelete: vi.fn(async () => ({}))
}));
vi.mock('../../stores/toastStore.svelte.js', () => ({ toastStore: { addToast: vi.fn() } }));

import { orderStore } from '../../stores/orderStore.svelte.js';
import OrderCartPosition from './OrderCartPosition.svelte';

// Unter 1 geht die Menge einer Position nicht; entfernt wird sie über das Kreuz. Der Knopf
// „Menge verringern" zeigt das: Er ist bei 1 gesperrt, statt ohne Wirkung zu bleiben.

/** @param {number} menge */
function position(menge) {
	orderStore.cart = [
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
	const { getByRole } = render(OrderCartPosition, { item: orderStore.cart[0] });
	return {
		minus: /** @type {HTMLButtonElement} */ (getByRole('button', { name: 'Menge verringern' })),
		plus: /** @type {HTMLButtonElement} */ (getByRole('button', { name: 'Menge erhöhen' }))
	};
}

describe('OrderCartPosition: die Menge', () => {
	beforeEach(() => {
		orderStore.cart = [];
	});

	it('sperrt „Menge verringern" bei 1', () => {
		const { minus, plus } = position(1);
		expect(minus.disabled).toBe(true);
		expect(plus.disabled).toBe(false);
	});

	it('gibt den Knopf über 1 frei und sperrt ihn wieder, sobald die Menge bei 1 ankommt', async () => {
		const { minus, plus } = position(1);

		await fireEvent.click(plus);
		expect(orderStore.cart[0].menge).toBe(2);
		expect(minus.disabled).toBe(false);

		await fireEvent.click(minus);
		expect(orderStore.cart[0].menge).toBe(1);
		expect(minus.disabled).toBe(true);
	});

	it('eine Position mit größerer Menge lässt sich verringern', async () => {
		const { minus } = position(3);
		expect(minus.disabled).toBe(false);
		await fireEvent.click(minus);
		expect(orderStore.cart[0].menge).toBe(2);
	});
});
