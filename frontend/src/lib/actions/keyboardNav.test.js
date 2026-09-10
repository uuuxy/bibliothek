import { describe, it, expect, vi } from 'vitest';
import { keyboardNav } from './keyboardNav.js';

/** @param {HTMLElement} node @param {string} key */
function taste(node, key) {
	node.dispatchEvent(new KeyboardEvent('keydown', { key, bubbles: true, cancelable: true }));
}

// Bestands-Durchgang 10.09.2026: Die Aktion führte einen EIGENEN Index und setzte ihn nur
// zurück, wenn das Dropdown schloss. Kamen beim Weitertippen neue Treffer, setzte der
// Store seinen Index auf -1 (nichts markiert), das Dropdown blieb offen — die Aktion stand
// weiter auf dem alten Index. Enter öffnete dann den zweiten Schüler der NEUEN Liste,
// obwohl nichts markiert war, und die folgenden Buchscans liefen auf dessen Konto.
// Der markierte Index hat seitdem EINE Quelle: den Store.
describe('keyboardNav', () => {
	it('Enter nach einer neuen Trefferliste wählt nichts, was nicht markiert ist', () => {
		const node = document.createElement('input');
		const onSelect = vi.fn();
		const onIndexChange = vi.fn();
		const p = {
			totalItems: 5,
			isOpen: true,
			selectedIndex: -1,
			onSelect,
			onIndexChange,
			onEscape: () => {}
		};
		const aktion = keyboardNav(node, p);

		taste(node, 'ArrowDown');
		aktion.update({ ...p, selectedIndex: 0 });
		taste(node, 'ArrowDown');
		expect(onIndexChange).toHaveBeenLastCalledWith(1);
		aktion.update({ ...p, selectedIndex: 1 });

		// Neue Treffer: Der Store setzt auf -1, das Dropdown bleibt offen.
		aktion.update({ ...p, totalItems: 4, selectedIndex: -1 });
		taste(node, 'Enter');
		expect(onSelect).not.toHaveBeenCalled();

		// Der nächste Pfeil beginnt oben, nicht beim alten Index.
		taste(node, 'ArrowDown');
		expect(onIndexChange).toHaveBeenLastCalledWith(0);
		aktion.destroy?.();
	});
});
