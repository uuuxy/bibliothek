import { describe, it, expect, vi, afterEach } from 'vitest';
import { druckeAusweis } from './ausweisDruck.js';

const SEITENREGEL = '85.6mm 53.98mm';

/** Hängt die Seitenregel des Ausweisdrucks gerade im Dokument? */
function seitenregelDa() {
	return [...document.head.querySelectorAll('style')].some((s) =>
		s.textContent?.includes(SEITENREGEL)
	);
}

/**
 * Ersetzt window.print und hält fest, was der Browser im Augenblick des Druckens vorfindet:
 * Die Druck-CSS liest die beiden Attribute am body, die Seitenregel gilt nur solange.
 */
function beimDrucken() {
	/** @type {{ modus: string | null, seite: string | null, seitenregel: boolean }[]} */
	const gesehen = [];
	vi.stubGlobal('print', () => {
		gesehen.push({
			modus: document.body.getAttribute('data-print-mode'),
			seite: document.body.getAttribute('data-print-card-side'),
			seitenregel: seitenregelDa()
		});
	});
	return gesehen;
}

afterEach(() => vi.unstubAllGlobals());

describe('druckeAusweis', () => {
	it('druckt beide Seiten im Kartenformat, ohne eine Seite zu nennen', () => {
		const gesehen = beimDrucken();

		druckeAusweis();

		expect(gesehen).toEqual([{ modus: 'card-single', seite: null, seitenregel: true }]);
	});

	it.each(['front', 'back'])('nennt der Druck-CSS die Seite „%s“', (seite) => {
		const gesehen = beimDrucken();

		druckeAusweis(/** @type {'front' | 'back'} */ (seite));

		expect(gesehen).toEqual([{ modus: 'card-single', seite, seitenregel: true }]);
	});

	it('räumt nach dem Drucken Attribute und Seitenregel wieder ab', () => {
		beimDrucken();

		druckeAusweis('front');

		// Bliebe etwas stehen, druckte die nächste Seite dieses Tabs im Kartenformat.
		expect(document.body.hasAttribute('data-print-mode')).toBe(false);
		expect(document.body.hasAttribute('data-print-card-side')).toBe(false);
		expect(seitenregelDa()).toBe(false);
	});
});
