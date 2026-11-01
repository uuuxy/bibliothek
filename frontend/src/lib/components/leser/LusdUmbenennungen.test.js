import { describe, it, expect } from 'vitest';
import { render } from '@testing-library/svelte';
import { SvelteSet } from 'svelte/reactivity';
import LusdUmbenennungen from './LusdUmbenennungen.svelte';

// Hat die LUSD ein Datum korrigiert, muss der Admin beide sehen, um das Paar zu beurteilen.
describe('LusdUmbenennungen', () => {
	it('nennt eine Korrektur des Geburtsdatums mit beiden Tagen', () => {
		const { container } = render(LusdUmbenennungen, {
			gewaehlt: new SvelteSet(),
			paare: /** @type {any} */ ([
				{
					zeile: 1,
					sicher: true,
					alt_vorname: 'Ada',
					alt_nachname: 'Alt',
					neu_vorname: 'Ada',
					neu_nachname: 'Neu',
					grund: 'Gleiche Anschrift',
					alt_geburtsdatum: '2011-03-04',
					neu_geburtsdatum: '2011-04-03'
				}
			])
		});
		expect(container.textContent).toContain('Geburtsdatum 04.03.2011 → 03.04.2011');
	});
});
