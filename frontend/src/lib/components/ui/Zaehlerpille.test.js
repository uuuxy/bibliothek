import { describe, it, expect } from 'vitest';
import { render } from '@testing-library/svelte';
import Zaehlerpille from './Zaehlerpille.svelte';

/**
 * Hintergrund (04.09.2026): Das Zähler-Badge gab es in ZWEI Fassungen. Die Seitenleiste
 * kappte die Zahl bei 999, die Reiterleiste nicht — im Druck-Center stand deshalb links
 * „999+" und daneben „30674". Aus der 16 px hohen Pille wurde ein 45 px langer Strich mit
 * 11-px-Ziffern, und die Frage dazu war „das entspricht sicherlich nicht Material 3?".
 *
 * Sie hatte recht: M3 kappt das Badge bei drei Zeichen. Diese Tests halten die Kappung
 * fest, weil sie an der echten Datenmenge hängt — lokal stehen 740 Etiketten offen, und
 * mit einer dreistelligen Zahl hätte jedes Gate am Bildschirm sie durchgewunken.
 */
describe('Zaehlerpille', () => {
	it('kappt bei 999+, damit aus der Pille kein Strich wird', () => {
		const { getByText } = render(Zaehlerpille, { anzahl: 30674 });
		expect(getByText('999+')).toBeTruthy();
	});

	// Nicht im aria-label: An einem <span> ohne Rolle ist es nicht erlaubt (axe
	// „aria-prohibited-attr"), und Bildschirmleser lesen dort den sichtbaren Text „999+".
	it('nennt die volle Zahl trotzdem — als Text für den Screenreader, die Kurzform ist ausgeblendet', () => {
		const { container, getByText } = render(Zaehlerpille, {
			anzahl: 30674,
			beschreibung: 'Etiketten offen'
		});
		expect(getByText('30674 Etiketten offen').className).toContain('sr-only');
		expect(getByText('999+').getAttribute('aria-hidden')).toBe('true');
		expect(container.querySelector('[aria-label]')).toBeNull();
	});

	it('zeigt Zahlen bis 999 unverändert', () => {
		const { getByText } = render(Zaehlerpille, { anzahl: 999 });
		expect(getByText('999')).toBeTruthy();
	});

	it('zeigt bei 0 gar nichts — M3 kennt kein leeres Badge', () => {
		const { container } = render(Zaehlerpille, { anzahl: 0 });
		expect(container.querySelector('span')).toBeNull();
	});

	it('trägt die Error-Rolle, die M3 für Badges vorsieht', () => {
		const { container } = render(Zaehlerpille, { anzahl: 5 });
		const klassen = (container.querySelector('span')?.getAttribute('class') || '').split(/\s+/);
		expect(klassen).toContain('bg-error');
		expect(klassen).toContain('text-on-error');
		// 16 dp Höhe, label-small: die Maße des M3 „large badge".
		expect(klassen).toContain('h-4');
		expect(klassen).toContain('text-label-small');
	});
});
