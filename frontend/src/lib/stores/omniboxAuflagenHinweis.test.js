import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render } from '@testing-library/svelte';

// Gemischte Auflagen an der Theke (docs/OFFEN.md 4.18, Stufe 5): Die Antwort einer Ausleihe
// trägt `auflagen_hinweis`, wenn Kinder derselben Klasse eine andere Auflage haben. Der Store
// hält ihn bis zum nächsten Scan, die Theke zeigt ihn als Zeile über dem Konto und blitzt wie
// bei der Fremdrückgabe. Gebucht ist die Ausleihe trotzdem.

vi.mock('./toastStore.svelte.js', () => ({ toastStore: { addToast: vi.fn() } }));
vi.mock('../apiFetch.js', () => ({ apiFetch: vi.fn(), apiClient: { post: vi.fn() } }));
vi.mock('../audio.js', () => ({
	playSoundSuccess: vi.fn(),
	playSoundError: vi.fn(),
	playSuccessBeep: vi.fn(),
	playErrorBeep: vi.fn()
}));

import { apiClient } from '../apiFetch.js';
import { createOmniboxStore } from './omnibox.svelte.js';
import OmniboxThekeHinweise from '../components/OmniboxThekeHinweise.svelte';
import { omniboxStore } from './omnibox.svelte.js';

const HINWEIS = {
	klasse: '07B',
	auflage: '4. Aufl.',
	erscheinungsjahr: 2023,
	andere: [{ auflage: '3. Aufl.', erscheinungsjahr: 2019, kinder: 12 }]
};

/** @type {ReturnType<typeof createOmniboxStore>[]} */
const stores = [];
afterEach(() => {
	while (stores.length) stores.pop()?.stoppeZeitgeber();
});

/** @param {ReturnType<typeof createOmniboxStore>} store @param {any} data */
async function scanne(store, data) {
	vi.mocked(apiClient.post).mockResolvedValue(
		/** @type {any} */ ({ ok: true, json: async () => data })
	);
	store.queryVal = 'B-518-1';
	await store.submitAction(null, null);
}

function neuerStore() {
	const store = createOmniboxStore();
	stores.push(store);
	return store;
}

describe('Theke: Hinweis bei gemischten Auflagen', () => {
	beforeEach(() => vi.clearAllMocks());

	it('hält den Hinweis der Ausleihe und blitzt als Warnung', async () => {
		const store = neuerStore();
		await scanne(store, {
			type: 'ausleihe',
			book: { titel: 'Mathe 7' },
			auflagen_hinweis: HINWEIS
		});
		expect(store.lastAuflagenHinweis).toEqual(HINWEIS);
		expect(store.screenFlash).toBe('warning');
	});

	it('bleibt bei einer gewöhnlichen Ausleihe leer und blitzt grün', async () => {
		const store = neuerStore();
		await scanne(store, { type: 'ausleihe', book: { titel: 'Mathe 7' } });
		expect(store.lastAuflagenHinweis).toBeNull();
		expect(store.screenFlash).toBe('success');
	});

	// Der nächste Scan ist eine Rückgabe, keine Ausleihe: Eine Ausleihe setzt den Hinweis ohnehin
	// neu, geprüft wäre dann nur ihr eigener Zweig — nicht das Zurücksetzen am Anfang jedes Scans.
	it('verschwindet mit dem nächsten Scan, auch wenn der keine Ausleihe ist', async () => {
		const store = neuerStore();
		await scanne(store, {
			type: 'ausleihe',
			book: { titel: 'Mathe 7' },
			auflagen_hinweis: HINWEIS
		});
		await scanne(store, { type: 'rueckgabe', book: { titel: 'Deutsch 7' } });
		expect(store.lastAuflagenHinweis).toBeNull();
	});

	it('steht als Zeile über dem Konto', async () => {
		omniboxStore.activeStudent = { vorname: 'Ida', art: 'schueler' };
		await scanne(omniboxStore, {
			type: 'ausleihe',
			book: { titel: 'Mathe 7' },
			auflagen_hinweis: HINWEIS
		});
		const screen = render(OmniboxThekeHinweise);
		expect(screen.getByRole('status').textContent).toContain(
			'Andere Auflage in der 07B: 12 Kinder haben 3. Aufl. · 2019 — dieses Exemplar ist 4. Aufl. · 2023.'
		);
		omniboxStore.stoppeZeitgeber();
	});
});
