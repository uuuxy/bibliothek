import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { render } from '@testing-library/svelte';
import { flushSync } from 'svelte';
import SnackbarAnzeige from './SnackbarAnzeige.svelte';
import { snackbars } from '../../stores/snackbars.svelte.js';
import { meldungenWeichenAus } from '../../actions/meldungenWeichenAus.js';

// Die Verdrahtung: Wer sich über use:meldungenWeichenAus einträgt, verschiebt die Meldungen;
// wer geht, gibt den Platz frei. Die Rechnung prüft meldungsPlatz.test.js, den Bildschirm
// e2e/meldungen-unten.spec.js. jsdom misst nicht: Die Elemente bekommen feste Kästen.

/** @param {{ top: number, bottom: number, right: number }} k */
function element(k) {
	const el = document.createElement('div');
	const kasten = { ...k, left: 0, x: 0, y: k.top, width: k.right, height: k.bottom - k.top };
	el.getBoundingClientRect = () => /** @type {DOMRect} */ ({ ...kasten, toJSON: () => kasten });
	document.body.append(el);
	return el;
}

/** @type {typeof globalThis.ResizeObserver | undefined} */
let vorher;

beforeEach(() => {
	vorher = globalThis.ResizeObserver;
	globalThis.ResizeObserver = /** @type {any} */ (
		class {
			observe() {}
			unobserve() {}
			disconnect() {}
		}
	);
	snackbars.toasts.forEach((t) => snackbars.removeToast(t.id));
});

afterEach(() => {
	snackbars.toasts.forEach((t) => snackbars.removeToast(t.id));
	globalThis.ResizeObserver = /** @type {any} */ (vorher);
});

describe('SnackbarAnzeige: Platz', () => {
	it('steht unten am Rand, rechts der Navigation, und steigt über eine Leiste, solange sie da ist', () => {
		const h = window.innerHeight;
		const navigation = meldungenWeichenAus(element({ top: 0, bottom: h, right: 256 }), 'links');
		const k = render(SnackbarAnzeige);
		const flaeche = /** @type {HTMLElement} */ (k.container.firstElementChild);

		snackbars.addToast('Gespeichert.', 'success');
		flushSync();
		expect(flaeche.style.left).toBe('256px');
		expect(flaeche.style.bottom).toBe('24px');

		// Eine Leiste 16 px über dem Fensterrand, 64 px hoch. Sie reicht in den Rand von 24 px,
		// denn jsdom misst die Meldungen mit 0 px Höhe.
		const leiste = meldungenWeichenAus(element({ top: h - 80, bottom: h - 16, right: 900 }));
		flushSync();
		expect(flaeche.style.bottom).toBe(`${16 + 64 + 16}px`);

		leiste.destroy();
		flushSync();
		expect(flaeche.style.bottom).toBe('24px');
		navigation.destroy();
	});
});

// Googles Snackbar: eine Fläche für jede Meldung (inverse-surface, Erhebung level3), kein
// Zeichen vor dem Text (M3: „Avoid adding icons to snackbars"), die Folgehandlung als Textknopf
// in inverse-primary, Schließen in der Farbe des Texts.
describe('SnackbarAnzeige: Form nach M3', () => {
	it('zeigt Fehler, Erfolg und Hinweis auf derselben Fläche ohne Zeichen', () => {
		const { container } = render(SnackbarAnzeige);
		for (const typ of /** @type {const} */ (['error', 'success', 'warning', 'info'])) {
			snackbars.addToast(`Meldung ${typ}`, typ);
			flushSync();
			const meldung = /** @type {HTMLElement} */ (
				container.querySelector('[role=alert], [role=status]')
			);
			expect(meldung.className).toContain('bg-inverse-surface');
			expect(meldung.className).toContain('text-inverse-on-surface');
			expect(meldung.className).toContain('shadow-2xl');
			expect(meldung.className).not.toMatch(/\bbg-(error|success|warning)\b/);
			expect(meldung.firstElementChild?.textContent?.trim()).toBe(`Meldung ${typ}`);
		}
	});

	it('trägt die Folgehandlung als Textknopf in inverse-primary', () => {
		const { getByRole } = render(SnackbarAnzeige);
		snackbars.addToast('Gespeichert', 'success', { label: 'Rückgängig', onClick: () => {} });
		flushSync();
		const aktion = getByRole('button', { name: 'Rückgängig' });
		expect(aktion.className).toContain('text-inverse-primary');
		expect(aktion.className).not.toContain('underline');
		expect(getByRole('button', { name: 'Schließen' }).className).not.toContain('opacity');
	});
});
