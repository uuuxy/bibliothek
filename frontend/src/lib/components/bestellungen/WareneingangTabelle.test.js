import { describe, it, expect } from 'vitest';
import { render, within } from '@testing-library/svelte';
import WareneingangTabelle from './WareneingangTabelle.svelte';

// Der Kopf jeder Bestellung nennt ihren Stand, damit die Bestätigung des Händlers auch hier zu
// sehen ist. Der Altbestand hat keine Bestellung und damit keinen Stand.
const gruppe = (/** @type {string} */ name, /** @type {object} */ stand) => ({
	id: name,
	supplierName: name,
	date: '30.09.2026',
	items: [{ titel: `Titel ${name}`, isbn: '', cover_url: '', menge: 1, exemplar_ids: [name] }],
	mit_bestaetigung: false,
	...stand
});

function kopf(/** @type {string} */ name) {
	const zeile = document.querySelector(`[data-gruppe="${name}"]`);
	expect(zeile).toBeTruthy();
	return within(/** @type {HTMLElement} */ (zeile));
}

describe('WareneingangTabelle: Stand im Kopf der Bestellung', () => {
	it('nennt je Bestellung ihren eigenen Stand', () => {
		render(WareneingangTabelle, {
			incomingShipments: [
				gruppe('Bestaetigt', {
					mit_bestaetigung: true,
					bestaetigt_am: '2026-10-12T10:00:00Z',
					bestaetigt_durch: 'lieferant'
				}),
				gruppe('Wartet', { mit_bestaetigung: true }),
				gruppe('Mail', { mit_bestaetigung: true, mail_gescheitert_am: '2026-10-13T10:00:00Z' }),
				gruppe('Altbestand', {})
			],
			totalItems: 4
		});

		expect(kopf('Bestaetigt').getByText('Bestätigt · 12.10.')).toBeTruthy();
		expect(kopf('Wartet').getByText('Wartet auf Händler')).toBeTruthy();
		expect(kopf('Mail').getByText('Mail nicht versendet')).toBeTruthy();
		expect(kopf('Mail').queryByText('Wartet auf Händler')).toBeNull();
		expect(document.querySelector('[data-gruppe="Altbestand"] [data-chip]')).toBeNull();
	});
});
