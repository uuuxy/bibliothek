import { it, expect, vi } from 'vitest';
import { abonniere } from '../../liveEvents.js';
import { ladeAkteBeiBuchung } from './leserakteLive.js';

vi.mock('../../liveEvents.js', () => ({ abonniere: vi.fn(() => () => {}) }));

// Theke und Leserdatei laden die offene Akte neu, wenn ein anderer Platz für denselben Leser
// bucht, und nur dann.
it('lädt die offene Akte nur bei einer Buchung für denselben Leser neu', () => {
	const ladeNeu = vi.fn();
	let offen = /** @type {string | undefined} */ ('leser-1');
	ladeAkteBeiBuchung(() => offen, ladeNeu);
	const [name, handler] = /** @type {any} */ (abonniere).mock.calls[0];
	expect(name).toBe('action');
	/** Eine Live-Meldung, wie sie die Leitung zustellt. @param {object} inhalt */
	const melde = (inhalt) => {
		const data = JSON.stringify(inhalt);
		handler({ data });
	};

	melde({ event: 'rueckgabe', student_id: 'leser-2' });
	expect(ladeNeu).not.toHaveBeenCalled();
	melde({ event: 'rueckgabe', student_id: 'leser-1' });
	expect(ladeNeu).toHaveBeenCalledTimes(1);

	// Ohne offene Akte und ohne Kennung im Ereignis (Platz ohne Vertrauen) geschieht nichts.
	melde({ event: 'rueckgabe' });
	offen = undefined;
	melde({ event: 'rueckgabe', student_id: 'leser-1' });
	expect(ladeNeu).toHaveBeenCalledTimes(1);
});
