import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';

import { toastStore } from './toastStore.svelte.js';

describe('toastStore', () => {
	beforeEach(() => {
		vi.useFakeTimers();
		toastStore.toasts.forEach((t) => toastStore.removeToast(t.id));
		toastStore.toasts = [];
	});

	afterEach(() => {
		vi.useRealTimers();
	});

	it('stapelt mehrere schnelle Meldungen statt sie zu überschreiben', () => {
		toastStore.addToast('Fehler 1', 'error');
		toastStore.addToast('Fehler 2', 'error');
		toastStore.addToast('Fehler 3', 'error');

		expect(toastStore.toasts).toHaveLength(3);
		expect(toastStore.toasts.map((t) => t.message)).toEqual(['Fehler 1', 'Fehler 2', 'Fehler 3']);
		expect(new Set(toastStore.toasts.map((t) => t.id)).size).toBe(3);
	});

	it('entfernt jede Meldung nach 5 Sekunden von selbst', () => {
		toastStore.addToast('vergänglich');
		vi.advanceTimersByTime(4999);
		expect(toastStore.toasts).toHaveLength(1);

		vi.advanceTimersByTime(1);
		expect(toastStore.toasts).toHaveLength(0);
	});

	it('lässt jüngere Meldungen stehen, wenn eine ältere abläuft', () => {
		toastStore.addToast('alt', 'error');
		vi.advanceTimersByTime(3000);
		toastStore.addToast('neu', 'error');

		vi.advanceTimersByTime(2000); // 'alt' läuft ab, 'neu' ist erst 2s alt
		expect(toastStore.toasts.map((t) => t.message)).toEqual(['neu']);
	});

	it('removeToast schließt manuell und der abgelaufene Timer räumt keine fremde Meldung ab', () => {
		toastStore.addToast('sofort weg', 'error');
		const id = toastStore.toasts[0].id;
		toastStore.removeToast(id);
		expect(toastStore.toasts).toHaveLength(0);

		toastStore.addToast('bleibt');
		vi.advanceTimersByTime(4999); // Timer des geschlossenen Toasts hätte hier gefeuert
		expect(toastStore.toasts).toHaveLength(1);
	});
});
