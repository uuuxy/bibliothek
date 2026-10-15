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

	it('stapelt mehrere schnelle Meldungen statt sie zu überschreiben', () => {
		snackbars.addToast('Fehler 1', 'error');
		snackbars.addToast('Fehler 2', 'error');
		snackbars.addToast('Fehler 3', 'error');

		expect(snackbars.toasts).toHaveLength(3);
		expect(snackbars.toasts.map((t) => t.message)).toEqual(['Fehler 1', 'Fehler 2', 'Fehler 3']);
		expect(new Set(snackbars.toasts.map((t) => t.id)).size).toBe(3);
	});

	it('entfernt jede Meldung nach 5 Sekunden von selbst', () => {
		snackbars.addToast('vergänglich');
		vi.advanceTimersByTime(4999);
		expect(snackbars.toasts).toHaveLength(1);

		vi.advanceTimersByTime(1);
		expect(snackbars.toasts).toHaveLength(0);
	});

	it('lässt jüngere Meldungen stehen, wenn eine ältere abläuft', () => {
		snackbars.addToast('alt', 'error');
		vi.advanceTimersByTime(3000);
		snackbars.addToast('neu', 'error');

		vi.advanceTimersByTime(2000); // 'alt' läuft ab, 'neu' ist erst 2s alt
		expect(snackbars.toasts.map((t) => t.message)).toEqual(['neu']);
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
