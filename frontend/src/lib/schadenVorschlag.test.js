import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, waitFor } from '@testing-library/svelte';
import DamageReportModal from './DamageReportModal.svelte';
import { apiFetch } from './apiFetch.js';

vi.mock('./apiFetch.js', () => ({ apiFetch: vi.fn() }));

// Gate gegen die Rückkehr der festen 15,00 € (OFFEN.md 9.3 a).
//
// Bis zum 17.09.2026 stand im Feld „Ersatzbetrag" ein `$state(15.0)` — eine Zahl ohne
// jeden Bezug zum Buch. Das Protokoll des Medienzentrums vom 16.09.2026 nennt genau das:
// „weder Einkaufs- bzw. Listenpreise noch Beschädigungsgrade hinterlegt, fehlende
// Restwertberechnung z.Zt. Handeingabe."
//
// Geprüft wird beides: dass der Vorschlag ANKOMMT (Betrag im Feld) und dass seine
// HERLEITUNG dabeisteht. Nur der Betrag wäre die halbe Wahrheit — eine Zahl, die aus dem
// Nichts kommt, ist für die Bibliothekskraft dasselbe wie die alte 15. Die Herleitung ist
// der Unterschied zwischen „das System sagt 24,90" und „3. Verleihjahr, 60 % von 41,50 €".
//
// Ebenfalls geprüft: ein gescheiterter Abruf sagt es. Sonst stünde eine 0,00 € im Feld,
// die wie ein berechneter Vorschlag aussieht — und eine Forderung über 0 € ist keine.
describe('Betragsvorschlag im Melde-Dialog', () => {
	const buch = { id: 'e1', ausleihe_id: 'a1', titel: 'Momo', barcode_id: 'B-1' };

	/** @param {any} antwort */
	function serverLiefert(antwort) {
		vi.mocked(apiFetch).mockResolvedValue(
			/** @type {any} */ ({ ok: true, json: async () => antwort })
		);
	}

	// clearAllMocks, NICHT mockReset: `mockReset` nimmt dem Mock seine Implementation, und
	// in dieser Kombination meldet Vitest 5 die Ablehnung des Fehlerfalls als
	// unbehandelt — der Test wäre rot, obwohl der Dialog den Fehler sauber fängt
	// (am 17.09.2026 eingegrenzt: dieselbe Probe ohne mockReset ist grün). Gebraucht wird
	// hier ohnehin nur eine leere Aufrufhistorie; die Implementation setzt jeder Test selbst.
	beforeEach(() => vi.clearAllMocks());

	it('trägt den Vorschlag des Servers ein, nicht die alte feste 15', async () => {
		serverLiefert({
			betrag: 24.9,
			herleitung: '3. Verleihjahr → 60 % von 41,50 € (Kaufpreis (kein Neupreis hinterlegt))',
			ist_lernmittel: true
		});

		const screen = render(DamageReportModal, {
			book: buch,
			onCancel: vi.fn(),
			onSubmit: vi.fn()
		});

		const feld = /** @type {HTMLInputElement} */ (screen.getByLabelText(/Ersatzbetrag/));
		await waitFor(() => expect(feld.value).toBe('24.9'));
		expect(feld.value, 'die feste 15 darf nicht zurückkommen').not.toBe('15');
	});

	it('fragt den Server nach genau diesem Exemplar', async () => {
		serverLiefert({ betrag: 12, herleitung: 'egal', ist_lernmittel: false });

		render(DamageReportModal, { book: buch, onCancel: vi.fn(), onSubmit: vi.fn() });

		await waitFor(() =>
			expect(apiFetch).toHaveBeenCalledWith('/api/buecher/exemplare/e1/ersatzwert-vorschlag')
		);
	});

	it('zeigt die Herleitung unter dem Feld', async () => {
		serverLiefert({
			betrag: 12,
			herleitung: 'Bücherei-Bestand: Neuwert ohne Abschlag (12,00 €, kein Neupreis hinterlegt)',
			ist_lernmittel: false
		});

		const screen = render(DamageReportModal, {
			book: buch,
			onCancel: vi.fn(),
			onSubmit: vi.fn()
		});

		await waitFor(() =>
			expect(screen.container.textContent ?? '').toContain('Neuwert ohne Abschlag')
		);
	});

	it('sagt es, wenn der Vorschlag nicht berechnet werden konnte', async () => {
		// mockImplementation statt mockRejectedValue: Letzteres erzeugt das abgelehnte
		// Promise schon beim Aufsetzen des Mocks — also einen Tick, bevor der Dialog seine
		// Kette daranhängt. Vitest meldet das als unbehandelte Ablehnung und färbt den Test
		// rot, obwohl der Dialog den Fehler sauber fängt.
		vi.mocked(apiFetch).mockImplementation(() => Promise.reject(new Error('Netz weg')));

		const screen = render(DamageReportModal, {
			book: buch,
			onCancel: vi.fn(),
			onSubmit: vi.fn()
		});

		await waitFor(() =>
			expect(screen.container.textContent ?? '').toContain('bitte Betrag selbst eintragen')
		);
	});

	it('fragt bei einem Kollegen gar nicht erst — dort entsteht keine Forderung', async () => {
		serverLiefert({ betrag: 99, herleitung: 'darf nie erscheinen', ist_lernmittel: true });

		render(DamageReportModal, {
			book: { ...buch, ohneForderung: true },
			onCancel: vi.fn(),
			onSubmit: vi.fn()
		});

		await new Promise((r) => setTimeout(r, 0));
		expect(apiFetch).not.toHaveBeenCalled();
	});
});
