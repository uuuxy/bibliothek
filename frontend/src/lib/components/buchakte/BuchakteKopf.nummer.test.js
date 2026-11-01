import { describe, it, expect } from 'vitest';
import { render } from '@testing-library/svelte';
import BuchakteKopf from './BuchakteKopf.svelte';

// Ein Buch trägt eine ISBN, eine CD oder DVD eine EAN.
const kopf = (/** @type {string} */ medientyp) =>
	render(BuchakteKopf, {
		titel: { id: 't-1', title: 'Natura', author: 'Beyer', isbn: '9780000000002', medientyp },
		ausleiher: [],
		exemplare: []
	}).container.textContent ?? '';

describe('Buchakte, Kopf: die Nummer', () => {
	it.each([
		['Buch', 'ISBN: 9780000000002'],
		['CD', 'EAN: 9780000000002'],
		['DVD', 'EAN: 9780000000002']
	])('%s: %s', (medientyp, erwartet) => {
		expect(kopf(medientyp)).toContain(erwartet);
	});
});
