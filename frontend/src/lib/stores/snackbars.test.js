import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';

import { snackbars } from './snackbars.svelte.js';

describe('snackbars', () => {
	beforeEach(() => {
		vi.useFakeTimers();
		snackbars.toasts.forEach((t) => snackbars.removeToast(t.id));
		snackbars.toasts = [];
	});

	afterEach(() => {
		vi.useRealTimers();
	});

	// Material 3, Snackbar: „Only one snackbar may be displayed at a time." und „a snackbar
	// with updated information can immediately replace an outdated snackbar."
	it('eine neue Meldung ersetzt die stehende, auch eine mit Folgehandlung', () => {
		snackbars.addToast('Eingebucht.', 'success', { label: 'Etiketten drucken', onClick: () => {} });
		snackbars.addToast('„Tintenherz" ausgeliehen.', 'success');
		snackbars.addToast('Das Buch ist vorgemerkt.', 'warning');

		expect(snackbars.toasts.map((t) => t.message)).toEqual(['Das Buch ist vorgemerkt.']);
	});

	it('entfernt jede Meldung nach 5 Sekunden von selbst', () => {
		snackbars.addToast('vergänglich');
		vi.advanceTimersByTime(4999);
		expect(snackbars.toasts).toHaveLength(1);

		vi.advanceTimersByTime(1);
		expect(snackbars.toasts).toHaveLength(0);
	});

	it('die neue steht ihre volle Zeit, der Timer der ersetzten nimmt sie nicht mit', () => {
		snackbars.addToast('alt', 'error');
		vi.advanceTimersByTime(3000);
		snackbars.addToast('neu', 'error');

		vi.advanceTimersByTime(2000); // hier liefe 'alt' ab
		expect(snackbars.toasts.map((t) => t.message)).toEqual(['neu']);
		vi.advanceTimersByTime(2999);
		expect(snackbars.toasts.map((t) => t.message)).toEqual(['neu']);
		vi.advanceTimersByTime(1);
		expect(snackbars.toasts).toHaveLength(0);
	});

	it('removeToast schließt manuell und der abgelaufene Timer räumt keine fremde Meldung ab', () => {
		snackbars.addToast('sofort weg', 'error');
		const id = snackbars.toasts[0].id;
		snackbars.removeToast(id);
		expect(snackbars.toasts).toHaveLength(0);

		snackbars.addToast('bleibt');
		vi.advanceTimersByTime(4999); // Timer des geschlossenen Toasts hätte hier gefeuert
		expect(snackbars.toasts).toHaveLength(1);
	});
});
