import { describe, it, expect, vi } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import LmfPlanVorrat from './LmfPlanVorrat.svelte';

// „Noch nicht im Plan" (06.09.2026): Offen steht nur, was ohne Regel fehlt — die neue
// Klasse nach dem LUSD-Import. Was Regel oder gespeicherter Plan bewusst auslassen (die
// Oberstufe), liegt eingeklappt hinter „11 Klassen bleiben draußen". Sonst böte die Seite
// jedes Jahr elf Chips an, die niemand will, und die eine echte Lücke ginge unter.
const KEIN_MARKER = { ohneSchueler: () => false };

describe('LmfPlanVorrat', () => {
	it('zeigt Klassen ohne Regel offen und die bewusst ausgelassenen eingeklappt', async () => {
		const hinein = vi.fn();
		const r = render(LmfPlanVorrat, {
			klassen: ['06F4', '12T1', '12T2', '13T1'],
			draussen: (k) => /^1[23]/.test(k),
			marker: KEIN_MARKER,
			onhinein: hinein
		});
		expect(r.getByRole('button', { name: '06F4 einplanen' })).toBeTruthy();
		expect(r.queryByRole('button', { name: '12T1 einplanen' })).toBeNull();

		const knopf = r.getByRole('button', { name: '3 Klassen bleiben draußen' });
		expect(knopf.getAttribute('aria-expanded')).toBe('false');
		await fireEvent.click(knopf);
		await fireEvent.click(r.getByRole('button', { name: '12T1 einplanen' }));
		expect(hinein).toHaveBeenCalledWith('12T1');
	});

	// Der Knopf nennt den Bereich, den er aufklappt, über dessen Kennung. Beide stehen an zwei
	// Stellen der Datei; der Test hält sie als Paar, damit keine der beiden allein umbenannt wird.
	it('nennt am Knopf die Kennung des Bereichs, den er aufklappt', async () => {
		const r = render(LmfPlanVorrat, {
			klassen: ['12T1'],
			draussen: () => true,
			marker: KEIN_MARKER,
			onhinein: vi.fn()
		});
		const knopf = r.getByRole('button', { name: 'Eine Klasse bleibt draußen' });
		await fireEvent.click(knopf);
		expect(knopf.getAttribute('aria-expanded')).toBe('true');
		const bereich = r.getByTestId('lmf-vorrat-draussen');
		expect(bereich.id).not.toBe('');
		expect(knopf.getAttribute('aria-controls')).toBe(bereich.id);
	});

	it('nimmt eine getippte Klasse über das Dialogfenster auf — Enter bestätigt', async () => {
		const hinein = vi.fn();
		const r = render(LmfPlanVorrat, {
			klassen: [],
			draussen: () => false,
			marker: KEIN_MARKER,
			onhinein: hinein
		});
		// Ohne fehlende Klasse: kein „Noch nicht im Plan", nur der Eintragen-Chip.
		expect(r.queryByText('Noch nicht im Plan:')).toBeNull();
		await fireEvent.click(r.getByRole('button', { name: 'Andere Klasse eintragen' }));
		const feld = r.getByLabelText('Klasse');
		await fireEvent.input(feld, { target: { value: ' 07G1 ' } });
		await fireEvent.submit(/** @type {HTMLElement} */ (feld.closest('form')));
		expect(hinein).toHaveBeenCalledWith('07G1');
		expect(r.queryByRole('dialog')).toBeNull();
	});
});
