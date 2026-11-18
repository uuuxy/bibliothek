import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { richteWelleEin } from './welle.js';

// Die Welle nach Googles md-ripple: entsteht beim Druck in Knöpfen mit State-Layer, nicht in
// gesperrten, nicht in Elementen ohne Schicht; verschwindet nach dem Loslassen frühestens 225 ms
// nach dem Druck; ein Klick über die Tastatur bekommt sie aus der Mitte.

/** @param {Element} el @param {string} art @param {Record<string, unknown>} [werte] */
function zeiger(el, art, werte = {}) {
	const e = new Event(art, { bubbles: true, cancelable: true });
	for (const [k, v] of Object.entries({
		isPrimary: true,
		button: 0,
		pointerType: 'mouse',
		clientX: 30,
		clientY: 10,
		...werte
	}))
		Object.defineProperty(e, k, { value: v });
	el.dispatchEvent(e);
}

/** Ein Element mit Maßen; jsdom misst sonst 0 × 0. @param {string} html */
function knopf(html) {
	document.body.innerHTML = html;
	const el = /** @type {HTMLElement} */ (document.body.firstElementChild);
	el.getBoundingClientRect = () =>
		/** @type {DOMRect} */ ({ left: 0, top: 0, width: 120, height: 36 });
	return el;
}

describe('Welle beim Drücken (utils/welle.js)', () => {
	/** @type {() => void} */
	let abmelden;
	/** @type {{ onfinish: null | (() => void), currentTime: number }[]} */
	let animationen;
	beforeEach(() => {
		vi.useFakeTimers();
		animationen = [];
		// jsdom kennt die Web Animations API nicht; die Attrappe merkt sich jede Animation.
		HTMLElement.prototype.animate = /** @type {any} */ (
			function () {
				const a = { onfinish: null, currentTime: 0 };
				animationen.push(a);
				return a;
			}
		);
		abmelden = richteWelleEin(document);
	});
	afterEach(() => {
		abmelden();
		vi.useRealTimers();
		// @ts-expect-error  die Attrappe wieder entfernen
		delete HTMLElement.prototype.animate;
		document.body.innerHTML = '';
	});

	it('wächst beim Druck in einem Knopf und verschwindet nach dem Loslassen', () => {
		const el = knopf('<button>Speichern</button>');
		zeiger(el, 'pointerdown');
		expect(el.querySelectorAll('.m3-welle')).toHaveLength(1);
		expect(el.querySelector('.m3-welle')?.getAttribute('aria-hidden')).toBe('true');
		zeiger(el, 'pointerup');
		vi.advanceTimersByTime(224);
		expect(animationen.length, 'erst nach 225 ms ausblenden').toBe(2);
		vi.advanceTimersByTime(1);
		expect(animationen.length).toBe(3);
		animationen[2].onfinish?.();
		expect(el.querySelector('.m3-welle')).toBeNull();
	});

	it('bleibt aus bei gesperrten Knöpfen, ohne Schicht und mit der rechten Taste', () => {
		const gesperrt = knopf('<button disabled>Gesperrt</button>');
		zeiger(gesperrt, 'pointerdown');
		expect(gesperrt.querySelector('.m3-welle')).toBeNull();
		const ohne = knopf('<button class="no-state-layer">Schalter</button>');
		zeiger(ohne, 'pointerdown');
		expect(ohne.querySelector('.m3-welle')).toBeNull();
		const rechts = knopf('<button>Menü</button>');
		zeiger(rechts, 'pointerdown', { button: 2 });
		expect(rechts.querySelector('.m3-welle')).toBeNull();
		const text = knopf('<div>Text</div>');
		zeiger(text, 'pointerdown');
		expect(text.querySelector('.m3-welle')).toBeNull();
	});

	it('kommt auch in Elementen mit .m3-state und aus der Mitte bei einem Klick der Tastatur', () => {
		const zeile = knopf('<label class="m3-state">Zeile</label>');
		zeiger(zeile, 'pointerdown');
		expect(zeile.querySelector('.m3-welle')).not.toBeNull();
		zeiger(zeile, 'pointerup');

		const el = knopf('<button>Weiter</button>');
		el.dispatchEvent(new MouseEvent('click', { bubbles: true, detail: 0 }));
		expect(el.querySelector('.m3-welle')).not.toBeNull();
		el.dispatchEvent(new MouseEvent('click', { bubbles: true, detail: 1 }));
		expect(
			el.querySelectorAll('.m3-welle'),
			'ein Klick der Maus kam schon beim Druck'
		).toHaveLength(1);
	});

	it('wächst in der Schicht von Kästchen und Radio, nicht wenn das Feld gesperrt ist', () => {
		const schicht = knopf('<span data-welle><input type="checkbox" /></span>');
		zeiger(/** @type {Element} */ (schicht.querySelector('input')), 'pointerdown');
		expect(schicht.querySelector('.m3-welle')).not.toBeNull();
		zeiger(schicht, 'pointerup');
		const gesperrt = knopf('<span data-welle><input type="checkbox" disabled /></span>');
		zeiger(/** @type {Element} */ (gesperrt.querySelector('input')), 'pointerdown');
		expect(gesperrt.querySelector('.m3-welle')).toBeNull();
	});

	it('wartet beim Finger 150 ms und bricht ab, wenn gewischt wird', () => {
		const el = knopf('<button>Finger</button>');
		zeiger(el, 'pointerdown', { pointerType: 'touch' });
		expect(el.querySelector('.m3-welle')).toBeNull();
		zeiger(el, 'pointercancel', { pointerType: 'touch' });
		vi.advanceTimersByTime(200);
		expect(el.querySelector('.m3-welle'), 'Wischen löst keine Welle aus').toBeNull();

		zeiger(el, 'pointerdown', { pointerType: 'touch' });
		vi.advanceTimersByTime(150);
		expect(el.querySelector('.m3-welle')).not.toBeNull();
	});
});
