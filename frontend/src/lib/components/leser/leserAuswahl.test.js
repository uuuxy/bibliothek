import { describe, it, expect } from 'vitest';
import { erzeugeLeserAuswahl } from './leserAuswahl.svelte.js';

// Der Haken an der Zeile der Leserdatei legt fest, wessen Ausweis gedruckt wird.
describe('Auswahl der Leserdatei', () => {
	it('schaltet eine Zeile an und wieder aus', () => {
		const auswahl = erzeugeLeserAuswahl(() => [{ id: 'l1' }, { id: 'l2' }]);

		auswahl.umschalten('l1');
		expect(auswahl.markierte.map((/** @type {any} */ l) => l.id)).toEqual(['l1']);

		auswahl.umschalten('l1');
		expect(auswahl.markierte).toEqual([]);
	});
});
