import { describe, it, expect, vi, beforeEach } from 'vitest';

// Die Theke, wenn ein abgeschriebenes Buch zurückkommt (#597, Etappe 2).
//
// Der Server sagt zwei verschiedene Dinge: was er erledigt hat (Forderung storniert) und
// was ein Mensch noch tun muss (der Bescheid liegt bei der Schulaufsicht, sie ist
// unverzüglich zu informieren). Das Zweite in einer grünen Erfolgsmeldung mitzuliefern
// hieße, es zu verstecken — geprüft wird deshalb der KANAL, nicht nur der Text.

vi.mock('./toastStore.svelte.js', () => ({ toastStore: { addToast: vi.fn() } }));
vi.mock('../apiFetch.js', () => ({ apiFetch: vi.fn(), apiClient: { post: vi.fn() } }));
vi.mock('../audio.js', () => ({
	playSoundSuccess: vi.fn(),
	playSoundError: vi.fn(),
	playSuccessBeep: vi.fn(),
	playErrorBeep: vi.fn()
}));

import { apiClient } from '../apiFetch.js';
import { toastStore } from './toastStore.svelte.js';
import { createOmniboxStore } from './omnibox.svelte.js';

const STORNO =
	'Buch reaktiviert. Die Forderung über 12,00 € wurde storniert — das Buch ist zurück.';
const AUFSICHT =
	'Die Forderung steht auf Bescheid 4801-2026-1234-0001, der bereits an die Schulaufsicht übergeben wurde — sie ist unverzüglich zu informieren.';

/** @param {Record<string, unknown>} data @returns {any} */
function antwort(data) {
	return { ok: true, json: async () => data };
}

/** @param {any} data */
async function scanne(data) {
	vi.mocked(apiClient.post).mockResolvedValue(antwort(data));
	const store = createOmniboxStore();
	store.queryVal = 'B-ZURUECK-1';
	await store.submitAction(null, null);
	return store;
}

describe('Rückkehr eines abgerechneten Buches', () => {
	beforeEach(() => vi.clearAllMocks());

	it('meldet die Stornierung als Erfolg', async () => {
		await scanne({ type: 'info', message: STORNO });

		expect(toastStore.addToast).toHaveBeenCalledWith(
			expect.stringContaining('storniert'),
			'success'
		);
		expect(toastStore.addToast).toHaveBeenCalledTimes(1);
	});

	it('meldet die offene Aufgabe als Warnung, nicht als Erfolg', async () => {
		await scanne({ type: 'info', message: 'Buch reaktiviert', aufsicht_informieren: AUFSICHT });

		expect(toastStore.addToast).toHaveBeenCalledWith(
			expect.stringContaining('Schulaufsicht'),
			'warning'
		);
	});

	// Das reservierende Kind bekommt das Buch im selben Zug: Der Scan endet als Ausleihe,
	// und dieser Zweig kannte `message`/`aufsicht_informieren` nicht — der Hinweis fiel
	// genau dort weg, wo jemand vor der Theke steht.
	it('verliert die Hinweise nicht, wenn der Scan in eine Ausleihe läuft', async () => {
		await scanne({
			type: 'ausleihe',
			book: { titel: 'Mathematik 7' },
			message: STORNO,
			aufsicht_informieren: AUFSICHT
		});

		expect(toastStore.addToast).toHaveBeenCalledWith(
			expect.stringContaining('storniert'),
			'success'
		);
		expect(toastStore.addToast).toHaveBeenCalledWith(
			expect.stringContaining('Schulaufsicht'),
			'warning'
		);
	});

	// Gegenprobe: Ein gewöhnlicher Scan ohne Forderung bekommt keinen zweiten Toast.
	it('schweigt, wenn es nichts zu melden gibt', async () => {
		await scanne({ type: 'info', message: 'Buch reaktiviert' });

		expect(toastStore.addToast).toHaveBeenCalledTimes(1);
		expect(toastStore.addToast).toHaveBeenCalledWith('Buch reaktiviert', 'success');
	});
});
