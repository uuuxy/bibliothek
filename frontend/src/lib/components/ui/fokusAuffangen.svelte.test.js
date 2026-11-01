import { describe, it, expect, afterEach } from 'vitest';
import { flushSync } from 'svelte';
import { fokusAuffangen } from './fokusAuffangen.svelte.js';

// Die Anwendung wird weiter bedient, während eine Liste nachlädt. Mit dem Element fiele der Fokus
// auf den Anfang der Seite, und die nächste Tab-Taste begönne vor der Navigation.
describe('fokusAuffangen', () => {
	/** @type {(() => void)[]} */
	let aufraeumen = [];
	afterEach(() => {
		for (const raeume of aufraeumen) raeume();
		aufraeumen = [];
	});

	/** @param {string} tag @param {string} [text] */
	function element(tag, text = '') {
		const el = document.createElement(tag);
		el.textContent = text;
		el.tabIndex = tag === 'button' ? 0 : -1;
		document.body.append(el);
		aufraeumen.push(() => el.remove());
		return el;
	}

	/** Knöpfe a und b in einer Wurzel; `zeige` entfernt die übrigen wie ein Zeichnen des Bauteils. */
	function bauteil({ mitZiel = true } = {}) {
		const wurzel = element('div');
		const knoepfe = ['a', 'b'].map((text) => wurzel.appendChild(element('button', text)));
		const ziel = element('p');
		let eintraege = $state(['a', 'b']);
		const beende = $effect.root(() => {
			fokusAuffangen(
				() => eintraege,
				() => wurzel,
				mitZiel ? () => ziel : undefined
			);
			$effect.pre(() => {
				for (const k of knoepfe) if (!eintraege.includes(k.textContent ?? '')) k.remove();
			});
		});
		flushSync();
		aufraeumen.push(beende);
		return {
			knoepfe,
			ziel,
			/** @param {string[]} neu */
			zeige(neu) {
				eintraege = neu;
				flushSync();
			}
		};
	}

	it('rückt auf das Ziel, wenn das Element mit dem Fokus verschwindet', () => {
		const { knoepfe, ziel, zeige } = bauteil();
		knoepfe[1].focus();

		zeige(['a']);

		expect(document.activeElement).toBe(ziel);
	});

	it('rückt ohne Ziel auf den Anfang des Inhalts', () => {
		const anfang = element('main');
		anfang.id = 'hauptinhalt';
		const { knoepfe, zeige } = bauteil({ mitZiel: false });
		knoepfe[1].focus();

		zeige(['a']);

		expect(document.activeElement).toBe(anfang);
	});

	it('lässt den Fokus, wenn sein Element bleibt', () => {
		const { knoepfe, zeige } = bauteil();
		knoepfe[0].focus();

		zeige(['a']);

		expect(document.activeElement).toBe(knoepfe[0]);
	});

	// Wer daneben arbeitet, etwa in einem Feld, behält den Fokus.
	it('lässt einen Fokus außerhalb der Wurzel in Ruhe', () => {
		const feld = element('input');
		const { zeige } = bauteil();
		feld.focus();

		zeige(['a']);

		expect(document.activeElement).toBe(feld);
	});

	it('steht der Fokus nirgends, bleibt es dabei', () => {
		const { zeige } = bauteil();
		/** @type {HTMLElement} */ (document.activeElement).blur();

		zeige(['a']);

		expect(document.activeElement).toBe(document.body);
	});
});
