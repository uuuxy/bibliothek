import { describe, it, expect, vi, beforeEach } from 'vitest';
import { erzeugeAusweisdruck } from './ausweisdruck.svelte.js';
import { idStore, vergissDesignLadezustand } from '../../designer/idDesignerStore.svelte.js';
import { apiFetch } from '../../apiFetch.js';
import { toastStore } from '../../stores/toastStore.svelte.js';

vi.mock('../../apiFetch.js', () => ({ apiFetch: vi.fn() }));
vi.mock('../../stores/toastStore.svelte.js', () => ({ toastStore: { addToast: vi.fn() } }));

// Raster-Fund 24.08.2026 („wer lädt den geteilten Zustand?"): Dieses Modul entscheidet
// am zentral gespeicherten printMode, ob Karten oder Etiketten aus dem Drucker kommen —
// geladen hat den Wert auf dem Schülerdatei-Pfad aber nur StudentBatchPrint, also genau
// die Komponente, die das Ergebnis des eigenen Ladens wieder abräumt. Das ging nur
// zufällig gut. Hier steht deshalb fest: Wer den Zustand liest, lädt ihn auch selbst.

/** Lässt die anstehenden Microtasks (fetch → json → applyDesign) durchlaufen. */
const stillhalten = () => new Promise((fertig) => setTimeout(fertig, 0));

beforeEach(() => {
	vi.clearAllMocks();
	// Sitzungs-Merker zurück: Jeder Fall hier prüft das VERHALTEN BEIM ERSTEN LADEN.
	vergissDesignLadezustand();
	idStore.printMode = 'card';
	idStore.etikettFormat = 'zweckform_l4760';
});

describe('erzeugeAusweisdruck: lädt das zentrale Design selbst', () => {
	it('übernimmt einen zentral gespeicherten Etikettenmodus ohne fremde Hilfe', async () => {
		vi.mocked(apiFetch).mockResolvedValue(
			/** @type {any} */ ({
				ok: true,
				json: async () => ({ printMode: 'etikett', etikettFormat: 'avery_3475' })
			})
		);

		const druck = erzeugeAusweisdruck();
		await stillhalten();

		expect(apiFetch).toHaveBeenCalledWith('/api/ausweis-layout');
		expect(druck.etikettModus).toBe(true);
		expect(druck.maxPosition).toBe(24); // avery_3475: 3×8
	});

	it('sagt es laut, wenn die Druckeinstellung nicht zu laden ist', async () => {
		// Der stille Rückfall auf 'card' wäre genau der falsche Ausdruck: Karten auf
		// Kartenrohlinge, obwohl die Schule Etiketten eingestellt hat.
		vi.mocked(apiFetch).mockRejectedValue(new Error('Netz weg'));

		const druck = erzeugeAusweisdruck();
		await stillhalten();

		expect(druck.etikettModus).toBe(false);
		expect(toastStore.addToast).toHaveBeenCalledWith(
			expect.stringContaining('Druckeinstellung'),
			'error'
		);
	});

	it('lädt nicht erneut, wenn das Design in dieser Sitzung schon geladen wurde', async () => {
		// Der zweite GET war nicht nur überflüssig — er überschrieb frische, noch nicht
		// fertig gespeicherte Designer-Änderungen mit dem alten Serverstand.
		vi.mocked(apiFetch).mockResolvedValue(
			/** @type {any} */ ({ ok: true, json: async () => ({ printMode: 'etikett' }) })
		);

		erzeugeAusweisdruck();
		await stillhalten();
		expect(apiFetch).toHaveBeenCalledTimes(1);

		// Zweiter Bildschirm derselben Sitzung — z. B. Schülerdatei erneut geöffnet,
		// nachdem der Designer inzwischen auf 'card' zurückgestellt hat (nur im Store).
		idStore.printMode = 'card';
		const druck = erzeugeAusweisdruck();
		await stillhalten();

		expect(apiFetch).toHaveBeenCalledTimes(1);
		expect(druck.etikettModus).toBe(false); // der Store gilt, nicht der alte Serverstand
	});
});
