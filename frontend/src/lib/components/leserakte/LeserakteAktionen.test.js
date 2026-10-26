import { describe, it, expect, vi, afterEach } from 'vitest';
import { render } from '@testing-library/svelte';
import LeserakteAktionen from './LeserakteAktionen.svelte';
import { idStore } from '../../designer/idDesignerStore.svelte.js';

// Ohne Ausweisnummer ist „Ausweis drucken" gesperrt, und die Blase sagt, wo die Nummer
// eingetragen wird: in jeder Akte unter „Stammdaten & Adresse" (LeserakteFelder.svelte). Ein
// Kollege ohne Konto steht in „Benutzer & Rechte" nicht. Das gilt auch mit gestalteter
// Rückseite, also dem geteilten Knopf mit Seitenwahl; die Vorgabe des Ausweises hat eine.
describe('Ausweis drucken in der Leserakte', () => {
	const rueckseite = idStore.back.elements;
	afterEach(() => {
		idStore.back.elements = rueckseite;
	});

	/** @param {string | null} barcode */
	const zeige = (barcode) =>
		render(LeserakteAktionen, {
			leser: {
				id: 'l1',
				vorname: 'Katrin',
				nachname: 'Wendlandt',
				art: 'lehrkraft',
				barcode_id: barcode
			},
			kontoauszugLaedt: false,
			rechnungLaedt: false,
			zeigeKontoauszug: vi.fn(),
			zeigeRechnung: vi.fn(),
			onPrint: vi.fn(),
			gueltigBis: null,
			onGueltigBis: vi.fn()
		});

	/** @param {import('@testing-library/svelte').RenderResult<any>} screen */
	const druckknopf = (screen) => screen.getByRole('button', { name: /^Ausweis drucken$/ });

	for (const [fall, mitRueckseite] of [
		['mit gestalteter Rückseite', true],
		['ohne Rückseite', false]
	]) {
		it(`nennt ohne Nummer den Ort, an dem sie eingetragen wird (${fall})`, () => {
			if (!mitRueckseite) {
				idStore.back.elements = rueckseite.map((e) => ({ ...e, show: false }));
			}
			const screen = zeige(null);
			const knopf = druckknopf(screen);
			expect(knopf.hasAttribute('disabled')).toBe(true);
			const blase = knopf.closest('[data-tip]')?.getAttribute('data-tip') ?? '';
			expect(blase).toContain('Stammdaten & Adresse');
			expect(blase).not.toContain('Benutzer & Rechte');
			expect(screen.queryByRole('button', { name: 'Ausweisseiten wählen' })).toBeNull();
		});
	}

	it('bietet mit Nummer und Rückseite die Seitenwahl an', () => {
		const screen = zeige('A-00042');
		expect(druckknopf(screen).hasAttribute('disabled')).toBe(false);
		expect(druckknopf(screen).closest('[data-tip]')).toBeNull();
		expect(screen.getByRole('button', { name: 'Ausweisseiten wählen' })).toBeTruthy();
	});
});
