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

	// Der Stand wechselt ohne Zutun, wenn der Händler bestätigt. Vorgelesen wird der ganze Kopf,
	// sonst hörte man „Bestätigt", ohne zu wissen, welche Bestellung gemeint ist.
	it('der Kopf liest den Wechsel des Stands mit dem Namen des Händlers vor', async () => {
		const wartet = gruppe('Naacher', { mit_bestaetigung: true });
		const screen = render(WareneingangTabelle, { incomingShipments: [wartet], totalItems: 1 });
		const region = screen.getByRole('status');
		expect(region.textContent).toContain('Wartet auf Händler');

		await screen.rerender({
			incomingShipments: [
				gruppe('Naacher', { mit_bestaetigung: true, bestaetigt_am: '2026-10-12T10:00:00Z' })
			]
		});

		expect(screen.getByRole('status')).toBe(region);
		expect(region.textContent).toContain('Naacher');
		expect(region.textContent).toContain('Bestätigt · 12.10.');
	});

	// Die Liste lädt neu, während jemand sie liest. An der Stelle in der Liste hinge der Kopf an
	// einer anderen Bestellung, und der Bildschirmleser läse jeden Kopf vor, der nur weitergerückt ist.
	it('ein Kopf bleibt bei seiner Bestellung, wenn sich die Reihenfolge ändert', async () => {
		const screen = render(WareneingangTabelle, {
			incomingShipments: [gruppe('A', {}), gruppe('B', {})],
			totalItems: 2
		});
		const kopfVonA = document.querySelector('[data-gruppe="A"]');

		await screen.rerender({ incomingShipments: [gruppe('B', {}), gruppe('A', {})] });

		expect(document.querySelector('[data-gruppe="A"]')).toBe(kopfVonA);
	});
});
