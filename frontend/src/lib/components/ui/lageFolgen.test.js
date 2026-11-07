import { describe, it, expect, vi, afterEach } from 'vitest';
import { folgeLage } from './lageFolgen.js';

// Rückt der Auslöser ohne Bildlauf, etwa weil ein zentrierter Dialog wächst, folgt die Fläche im
// nächsten Bild; steht alles still, wird nicht neu gemessen.
describe('folgeLage', () => {
	afterEach(() => {
		vi.useRealTimers();
	});

	it('misst sofort, dann nur, wenn der Auslöser rückt, und nach dem Aufräumen nicht mehr', () => {
		vi.useFakeTimers({
			toFake: ['requestAnimationFrame', 'cancelAnimationFrame', 'setTimeout', 'clearTimeout']
		});
		let oben = 100;
		const ausloeser = {
			getBoundingClientRect: () => ({ left: 10, top: oben, width: 200, height: 36 })
		};
		const messen = vi.fn();
		const aufraeumen = folgeLage(() => /** @type {any} */ (ausloeser), messen);
		expect(messen).toHaveBeenCalledTimes(1);

		vi.advanceTimersByTime(100);
		expect(messen).toHaveBeenCalledTimes(1);

		oben = 42;
		vi.advanceTimersByTime(20);
		expect(messen).toHaveBeenCalledTimes(2);

		aufraeumen();
		oben = 7;
		vi.advanceTimersByTime(100);
		expect(messen).toHaveBeenCalledTimes(2);
	});
});
