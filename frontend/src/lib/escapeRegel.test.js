import { describe, it, expect } from 'vitest';
import { escapeGehoertJemandAnderem } from './escapeRegel.js';
import { escapeSchliesst } from './components/ui/escapeSchliesst.js';

// Die Regel des Routers: Escape bringt an die Theke — außer die Taste gehört schon
// jemandem. Der dritte Fall (offenes Overlay, Fokus auf einem Knopf) fehlte bis
// 06.09.2026: Der Router hört als Erster auf window, das Overlay schließt erst danach
// — und die Seite sprang mit demselben Tastendruck zur Theke.
describe('escapeGehoertJemandAnderem', () => {
	it('gibt die Taste frei, wenn nichts offen ist und der Fokus auf einem Knopf steht', () => {
		const knopf = document.createElement('button');
		document.body.appendChild(knopf);
		const e = new KeyboardEvent('keydown', { key: 'Escape' });
		Object.defineProperty(e, 'target', { value: knopf });
		expect(escapeGehoertJemandAnderem(e)).toBe(false);
		knopf.remove();
	});

	it('überlässt die Taste einem offenen Overlay — auch bei Fokus auf einem Knopf', () => {
		const knopf = document.createElement('button');
		const overlay = document.createElement('div');
		document.body.append(knopf, overlay);
		const aktion = escapeSchliesst(overlay, () => {});
		const e = new KeyboardEvent('keydown', { key: 'Escape' });
		Object.defineProperty(e, 'target', { value: knopf });
		expect(escapeGehoertJemandAnderem(e)).toBe(true);
		aktion.destroy();
		expect(escapeGehoertJemandAnderem(e)).toBe(false);
		knopf.remove();
		overlay.remove();
	});
});
