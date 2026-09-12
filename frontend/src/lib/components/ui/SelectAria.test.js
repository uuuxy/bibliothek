import { describe, it, expect } from 'vitest';
import { render } from '@testing-library/svelte';
import Select from './Select.svelte';

// Der Screenreader muss der Pfeiltasten-Markierung folgen (#593, aus Paket 1).
//
// Das Auswahlfeld ist ein nachgebautes Widget: Der Fokus bleibt auf dem Knopf
// (role="combobox"), die Markierung wandert in der Liste daneben. Ohne
// `aria-activedescendant` erfährt ein Screenreader davon nichts — er liest beim Öffnen
// den Knopf vor und danach schweigt er, während die Sehende die Auswahl wandern sieht.
// WAI-ARIA nennt genau diese Kombination (combobox + listbox + activedescendant).

const optionen = [
	{ value: 'a', label: 'Anna' },
	{ value: 'b', label: 'Bernd' },
	{ value: 'c', label: 'Cem' }
];

/** @param {HTMLElement} knopf @param {string} taste */
function tippe(knopf, taste) {
	knopf.dispatchEvent(new KeyboardEvent('keydown', { key: taste, bubbles: true }));
}

describe('Select: was der Screenreader mitbekommt', () => {
	it('nennt geschlossen keinen aktiven Eintrag', () => {
		const screen = render(Select, { options: optionen, value: 'a' });
		expect(screen.getByRole('combobox').getAttribute('aria-activedescendant')).toBeNull();
	});

	it('zeigt beim Öffnen auf den gewählten Eintrag', async () => {
		const screen = render(Select, { options: optionen, value: 'b' });
		const knopf = screen.getByRole('combobox');

		tippe(knopf, 'ArrowDown');
		await Promise.resolve();

		const aktiv = knopf.getAttribute('aria-activedescendant');
		expect(aktiv, 'ohne Verweis liest der Screenreader nur den Knopf').toBeTruthy();
		const zeile = document.getElementById(/** @type {string} */ (aktiv));
		expect(zeile?.textContent).toContain('Bernd');
		expect(zeile?.getAttribute('role')).toBe('option');
	});

	it('wandert mit der Markierung weiter', async () => {
		const screen = render(Select, { options: optionen, value: 'a' });
		const knopf = screen.getByRole('combobox');

		tippe(knopf, 'ArrowDown'); // öffnet, steht auf „Anna"
		await Promise.resolve();
		tippe(knopf, 'ArrowDown'); // eine Zeile weiter
		await Promise.resolve();

		const aktiv = knopf.getAttribute('aria-activedescendant');
		expect(document.getElementById(/** @type {string} */ (aktiv))?.textContent).toContain('Bernd');
	});

	// Zwei Felder auf einer Seite dürfen nicht auf dieselben Zeilen verweisen — sonst
	// liest der Screenreader die Zeile des anderen Feldes vor.
	it('vergibt je Feld eigene Kennungen, auch ohne id von außen', async () => {
		// Zwei Felder in EINEM Dokument — deshalb je Container gesucht, nicht global.
		const a = render(Select, { options: optionen, value: 'a' });
		const b = render(Select, { options: optionen, value: 'a' });
		const knopfIn = (/** @type {HTMLElement} */ behaelter) =>
			/** @type {HTMLElement} */ (behaelter.querySelector('[role="combobox"]'));

		tippe(knopfIn(a.container), 'ArrowDown');
		tippe(knopfIn(b.container), 'ArrowDown');
		await Promise.resolve();

		const kennungA = knopfIn(a.container).getAttribute('aria-activedescendant');
		const kennungB = knopfIn(b.container).getAttribute('aria-activedescendant');
		expect(kennungA).toBeTruthy();
		expect(kennungA, 'beide Felder verweisen auf dieselbe Zeile').not.toBe(kennungB);
	});
});
