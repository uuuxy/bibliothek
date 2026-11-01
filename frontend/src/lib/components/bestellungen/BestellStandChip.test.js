import { describe, it, expect } from 'vitest';
import { render } from '@testing-library/svelte';
import BestellStandChip from './BestellStandChip.svelte';

// Beide Wege der Bestätigung sagen im Tooltip, wer sie gegeben hat. Der gescheiterte Versand
// schlägt alles andere: Ohne Mail hat der Händler nichts, das er bestätigen könnte.
const chip = (/** @type {object} */ bestellung) => {
	const { container } = render(BestellStandChip, { bestellung });
	return /** @type {HTMLElement | null} */ (container.querySelector('[data-chip]'));
};

describe('BestellStandChip', () => {
	it('nennt die Bestätigung über den Link mit Tag', () => {
		const c = chip({
			mit_bestaetigung: true,
			bestaetigt_am: '2026-10-12T10:00:00Z',
			bestaetigt_durch: 'lieferant'
		});
		expect(c?.textContent?.trim()).toBe('Bestätigt · 12.10.');
		expect(c?.dataset.tip).toBe('Der Lieferant hat die Bestellung über den Link bestätigt');
	});

	it('nennt den Nachtrag in der Bibliothek als solchen', () => {
		const c = chip({
			mit_bestaetigung: true,
			bestaetigt_am: '2026-10-12T10:00:00Z',
			bestaetigt_durch: 'bibliothek'
		});
		expect(c?.dataset.tip).toBe('Bestätigung wurde in der Bibliothek von Hand nachgetragen');
	});

	it('wartet nur bei einer Bestellung mit Link', () => {
		expect(chip({ mit_bestaetigung: true })?.textContent?.trim()).toBe('Wartet auf Händler');
		expect(chip({ mit_bestaetigung: false })).toBeNull();
	});

	it('der gescheiterte Versand steht vor dem Warten', () => {
		const c = chip({ mit_bestaetigung: true, mail_gescheitert_am: '2026-10-13T10:00:00Z' });
		expect(c?.textContent?.trim()).toBe('Mail nicht versendet');
	});
});
