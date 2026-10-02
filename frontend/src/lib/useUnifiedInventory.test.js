import { describe, it, expect, vi, beforeEach } from 'vitest';

vi.mock('./stores/toastStore.svelte.js', () => ({
	toastStore: { addToast: vi.fn() }
}));
vi.mock('./stores/bestaetigung.svelte.js', () => ({ bestaetigen: vi.fn() }));
vi.mock('./inventurApi.js');
vi.mock('./apiFetch.js', () => ({
	apiFetch: vi.fn(async () => ({ ok: false, json: async () => ({}) }))
}));

import { bestaetigen } from './stores/bestaetigung.svelte.js';
import { schliesseAb, ladeOffeneSessions } from './inventurApi.js';
import { useUnifiedInventory, abschlussText } from './useUnifiedInventory.svelte.js';

const frage = vi.mocked(bestaetigen);
const abschluss = vi.mocked(schliesseAb);

// Der Abschluss bucht jedes nicht gescannte Exemplar als Verlust und sondert es aus. Davor
// steht die Rückfrage des Hauses; „nein" (auch Escape) lässt die Inventur weiterlaufen.
describe('Inventur abschließen', () => {
	beforeEach(() => {
		vi.clearAllMocks();
		vi.mocked(ladeOffeneSessions).mockResolvedValue([]);
	});

	it('nennt in der Rückfrage, wie viele Bücher als verloren gebucht werden', () => {
		expect(abschlussText(0)).toContain('keines als verloren');
		expect(abschlussText(1)).toMatch(/^1 Buch .* ist nicht gescannt\. Es wird unwiderruflich/);
		expect(abschlussText(47)).toMatch(
			/^47 Bücher .* sind nicht gescannt\. Sie werden unwiderruflich/
		);
	});

	it('schließt bei „nein" nicht ab und gibt den Fokus zurück ins Scanfeld', async () => {
		frage.mockResolvedValue(false);
		const fokus = vi.fn();
		await useUnifiedInventory().abschliessenNachRueckfrage(fokus);

		expect(frage).toHaveBeenCalledWith(
			expect.objectContaining({ titel: 'Inventur abschließen?', gefaehrlich: true })
		);
		expect(abschluss).not.toHaveBeenCalled();
		expect(fokus).toHaveBeenCalledTimes(1);
	});

	it('schließt bei „ja" ab', async () => {
		frage.mockResolvedValue(true);
		abschluss.mockResolvedValue(
			/** @type {any} */ ({ ok: true, data: { verloren_gemeldet: 0, fehlbestand: [] } })
		);
		await useUnifiedInventory().abschliessenNachRueckfrage();

		expect(abschluss).toHaveBeenCalledTimes(1);
	});
});
