import { describe, it, expect, vi, afterEach } from 'vitest';
import { fireEvent, render } from '@testing-library/svelte';
import AbgangsjahrFeld from './AbgangsjahrFeld.svelte';

// „Neu berechnen" rechnet das Abgangsjahr aus der Klasse (abgangsjahr.js, geprüft in
// abgangsjahr.test.js). Am 1. August beginnt das neue Schuljahr, dann rückt der Abschluss einer
// Klasse ein Jahr weiter. „9a" hat keinen Zweigbuchstaben und endet wie im Server mit 13.
describe('AbgangsjahrFeld: Neu berechnen', () => {
	afterEach(() => vi.useRealTimers());

	it.each([
		['am letzten Tag des Schuljahrs', new Date(2026, 6, 31), '2030'],
		['am ersten Tag des neuen Schuljahrs', new Date(2026, 7, 1), '2031']
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
