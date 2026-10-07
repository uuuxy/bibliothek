import { describe, it, expect } from 'vitest';
import { render } from '@testing-library/svelte';
import BorrowersListe from './BorrowersListe.svelte';

// Überfällig trägt Farbe, Zeichen und für Vorleseprogramme das Wort, wie in der Leserakte
// (UeberfaelligZeichen.svelte). Wer Farben schlecht unterscheidet, sieht an der Farbe allein
// keinen Unterschied.
describe('Ausleiher-Liste: überfällig', () => {
	const tag = 24 * 60 * 60 * 1000;
	/** @param {number} fristInTagen @param {boolean} [dauerleihe] */
	const zeile = (fristInTagen, dauerleihe = false) => ({
		schueler_name: 'Kim',
		schueler_nachname: 'Beispiel',
		klasse: '07a',
		schueler_barcode: 'A-1',
		exemplar_barcode: 'B-1',
		ausgeliehen_am: new Date(Date.now() - 30 * tag).toISOString(),
		rueckgabe_frist: new Date(Date.now() + fristInTagen * tag).toISOString(),
		ist_dauerleihe: dauerleihe
	});
	const fmtDate = (/** @type {string} */ d) => new Date(d).toLocaleDateString('de-DE');
	/** @param {any} z */
	const zeige = (z) => render(BorrowersListe, { zeilen: [z], onBack: () => {}, fmtDate });

	it('setzt an die überfällige Ausleihe ein Zeichen und das Wort', () => {
		const { container, getByText } = zeige(zeile(-3));
		expect(container.querySelector('svg[aria-hidden="true"]'), 'das Zeichen fehlt').not.toBeNull();
		expect(getByText('Überfällig').className).toContain('sr-only');
	});

	it('lässt die Ausleihe in der Frist und die Dauerleihe ohne Zeichen und Wort', () => {
		for (const z of [zeile(7), zeile(-3, true)]) {
			const { container, queryByText, unmount } = zeige(z);
			expect(container.querySelector('svg')).toBeNull();
			expect(queryByText('Überfällig')).toBeNull();
			unmount();
		}
	});
});
