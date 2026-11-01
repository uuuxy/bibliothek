import { describe, it, expect, vi } from 'vitest';
import { render } from '@testing-library/svelte';
import OmniboxTreffer from './OmniboxTreffer.svelte';

// Der Server schickt je Gruppe höchstens zehn Zeilen. Nur der Zusatz „von N“ sagt, dass es mehr
// gibt und dass weiteres Tippen hilft.
const leser = (/** @type {number} */ n) =>
	Array.from({ length: n }, (_, i) => ({ id: `s${i}`, vorname: 'V', nachname: `N${i}` }));

const zeige = (/** @type {object} */ treffer) =>
	render(OmniboxTreffer, {
		suchTreffer: { students: [], books: [], ...treffer },
		markierterTreffer: -1,
		onSelect: vi.fn()
	});

describe('OmniboxTreffer: Überschrift der Gruppe', () => {
	it('nennt die Gesamtzahl, wenn die Liste gekappt ist', () => {
		const screen = zeige({ students: leser(2), studentsTotal: 37 });
		expect(screen.getByText('Leser (2 von 37)')).toBeTruthy();
	});

	it('nennt nur die Zahl, wenn alle Treffer dastehen', () => {
		const screen = zeige({ students: leser(2), studentsTotal: 2 });
		expect(screen.getByText('Leser (2)')).toBeTruthy();
	});
});
