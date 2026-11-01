import { describe, it, expect, vi, afterEach } from 'vitest';
import { fireEvent, render } from '@testing-library/svelte';
import AbgangsjahrFeld from './AbgangsjahrFeld.svelte';

// „Neu berechnen" rechnet das Abgangsjahr aus der Klasse. Am 1. August beginnt das neue Schuljahr,
// dann rückt der Abschluss einer Klasse ein Jahr weiter.
describe('AbgangsjahrFeld: Neu berechnen', () => {
	afterEach(() => vi.useRealTimers());

	it.each([
		['am letzten Tag des Schuljahrs', new Date(2026, 6, 31), '2027'],
		['am ersten Tag des neuen Schuljahrs', new Date(2026, 7, 1), '2028']
	])('%s', async (_wann, heute, soll) => {
		vi.useFakeTimers({ now: heute, toFake: ['Date'] });
		const feld = render(AbgangsjahrFeld, {
			leser: { id: 's1', klasse: '9a', abgaenger_jahr: 2000 },
			darfBearbeiten: true
		});

		await fireEvent.click(feld.getByTitle('Abgangsjahr bearbeiten'));
		await fireEvent.click(feld.getByTitle('Automatisch aus Klasse berechnen'));

		expect(/** @type {HTMLInputElement} */ (feld.getByLabelText('Abgangsjahr')).value).toBe(soll);
	});
});
