import { describe, it, expect, vi, afterEach, beforeEach } from 'vitest';
import { druckeAusweis } from './ausweisDruck.js';
import { designFuerDruck } from './designer/ausweisDesignLaden.js';

vi.mock('./designer/ausweisDesignLaden.js', () => ({ designFuerDruck: vi.fn() }));

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

beforeEach(() => {
	vi.mocked(designFuerDruck).mockResolvedValue(true);
});
afterEach(() => vi.unstubAllGlobals());

describe('druckeAusweis', () => {
	it('druckt beide Seiten im Kartenformat, ohne eine Seite zu nennen', async () => {
		const gesehen = beimDrucken();

		await druckeAusweis();

		expect(gesehen).toEqual([{ modus: 'card-single', seite: null, seitenregel: true }]);
	});

	it.each(['front', 'back'])('nennt der Druck-CSS die Seite „%s“', async (seite) => {
		const gesehen = beimDrucken();

		await druckeAusweis(/** @type {'front' | 'back'} */ (seite));

		expect(gesehen).toEqual([{ modus: 'card-single', seite, seitenregel: true }]);
	});

	// Ohne das gespeicherte Design käme die Karte mit den Standardwerten aus dem Drucker.
	it('druckt nicht, solange das Ausweis-Design nicht geladen ist', async () => {
		vi.mocked(designFuerDruck).mockResolvedValue(false);
		const gesehen = beimDrucken();

		await druckeAusweis();

		expect(gesehen).toEqual([]);
		expect(document.body.hasAttribute('data-print-mode')).toBe(false);
		expect(seitenregelDa()).toBe(false);
	});

	it('räumt nach dem Drucken Attribute und Seitenregel wieder ab', async () => {
		beimDrucken();

		await druckeAusweis('front');

		// Bliebe etwas stehen, druckte die nächste Seite dieses Tabs im Kartenformat.
		expect(document.body.hasAttribute('data-print-mode')).toBe(false);
		expect(document.body.hasAttribute('data-print-card-side')).toBe(false);
		expect(seitenregelDa()).toBe(false);
	});
});
