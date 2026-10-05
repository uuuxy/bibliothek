import { describe, it, expect, vi, beforeEach } from 'vitest';
import { useFehlbestand } from './useFehlbestand.svelte.js';
import { meldeVerlustGefunden } from './inventurApi.js';
import { toastStore } from './stores/toastStore.svelte.js';

vi.mock('./inventurApi.js', () => ({
	ladeAbgeschlosseneInventuren: vi.fn(),
	ladeFehlbestand: vi.fn(),
	meldeVerlustGefunden: vi.fn(),
	loescheVerlustEndgueltig: vi.fn()
}));
vi.mock('./stores/toastStore.svelte.js', () => ({ toastStore: { addToast: vi.fn() } }));

// Ein als Verlust gebuchtes Buch taucht beim Absuchen der Regale wieder auf. Der Fund beendet
// auch die Forderung, die das Buch abgerechnet hatte; die Meldung nennt den Betrag.
describe('Fehlbestand: als gefunden verbuchen', () => {
	beforeEach(() => vi.clearAllMocks());

	it('nennt den stornierten Betrag mit Komma und zwei Nachkommastellen', async () => {
		vi.mocked(meldeVerlustGefunden).mockResolvedValue(
			/** @type {any} */ ({
				ok: true,
				data: { stornierte_forderungen: 1, stornierter_betrag: 1234.5 }
			})
		);

		await useFehlbestand().fehlbestandGefunden('ex-1');

		expect(toastStore.addToast).toHaveBeenCalledWith(
			'Als gefunden verbucht — Exemplar ist wieder verfügbar. Die Forderung über 1.234,50 € wurde storniert.',
			'success'
		);
	});

	it('nennt ohne stornierte Forderung keinen Betrag', async () => {
		vi.mocked(meldeVerlustGefunden).mockResolvedValue(
			/** @type {any} */ ({ ok: true, data: { stornierte_forderungen: 0 } })
		);

		await useFehlbestand().fehlbestandGefunden('ex-1');

		expect(toastStore.addToast).toHaveBeenCalledWith(
			'Als gefunden verbucht — Exemplar ist wieder verfügbar.',
			'success'
		);
	});
});
