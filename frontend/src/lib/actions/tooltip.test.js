import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { initTooltips } from './tooltip.js';

// Die Sprechblase hängt an document.body, nicht an ihrem Ziel. Verschwindet das Ziel unter
// dem ruhenden Zeiger — die Theke leert sich, der Sperrbildschirm kommt —, meldet der
// Browser kein Verlassen, und die Blase stünde ohne ihr Ziel da.
describe('Sprechblase: geht mit ihrem Ziel', () => {
	/** @type {() => void} */
	let abmelden;
	/** @type {ReturnType<typeof vi.fn>} */
	let zeigen;
	/** @type {ReturnType<typeof vi.fn>} */
	let verstecken;

	beforeEach(() => {
		vi.useFakeTimers();
		// jsdom kennt die Popover-API nicht; gezählt wird, was die Blase von ihr verlangt.
		zeigen = vi.fn();
		verstecken = vi.fn();
		Object.assign(HTMLElement.prototype, { showPopover: zeigen, hidePopover: verstecken });
		document.body.innerHTML =
			'<div id="rahmen"><table><tbody><tr id="zeile"><td><span id="titel" data-tip="Tintenherz · Funke · 5896">Tintenherz</span></td></tr></tbody></table></div>';
		abmelden = initTooltips();
	});
	afterEach(() => {
		abmelden();
		vi.useRealTimers();
		document.body.innerHTML = '';
	});

	/** Zeigt auf den Titel und wartet die Verzögerung ab. */
	function zeigeAufTitel() {
		document.getElementById('titel')?.dispatchEvent(new MouseEvent('mouseover', { bubbles: true }));
		vi.advanceTimersByTime(200);
		expect(zeigen).toHaveBeenCalledTimes(1);
		verstecken.mockClear();
	}

	it('die Zeile wird entfernt: die Blase schließt', async () => {
		zeigeAufTitel();
		document.getElementById('zeile')?.remove();
		await Promise.resolve();
		expect(verstecken).toHaveBeenCalled();
	});

	it('die Anwendung wird ausgeblendet (hidden, inert): die Blase schließt', async () => {
		zeigeAufTitel();
		document.getElementById('rahmen')?.setAttribute('hidden', '');
		await Promise.resolve();
		expect(verstecken).toHaveBeenCalled();
	});

	it('ändert sich nebenan etwas, bleibt sie stehen', async () => {
		zeigeAufTitel();
		document.body.appendChild(document.createElement('p'));
		await Promise.resolve();
		expect(verstecken).not.toHaveBeenCalled();
	});

	it('ein Ziel, das beim Ablauf der Verzögerung schon fort ist, bekommt keine Blase', () => {
		const titel = document.getElementById('titel');
		titel?.dispatchEvent(new MouseEvent('mouseover', { bubbles: true }));
		document.getElementById('zeile')?.remove();
		vi.advanceTimersByTime(200);
		expect(zeigen).not.toHaveBeenCalled();
	});
});
