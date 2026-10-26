import { describe, it, expect, vi } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import InventurLaeufe from './InventurLaeufe.svelte';

/** Die Fassade, soweit die Abschnitte sie lesen. @param {Record<string, any>} [anders] */
const ablauf = (anders = {}) => ({
	offeneLaeufe: [],
	abgeschlosseneInventuren: [],
	nichtGeladen: { laeufe: '', frueher: '' },
	ladeFruehereLaeuft: false,
	ladeOffeneLaeufe: vi.fn(),
	ladeFruehereInventuren: vi.fn(),
	setzeLaufFort: vi.fn(),
	verwerfeLauf: vi.fn(),
	zeigeFrueherenFehlbestand: vi.fn(),
	...anders
});

// Fehlt die Liste der laufenden Inventuren, weil sie nicht kam, lädt die Seite zu einem neuen
// Lauf ein, während einer läuft; der Server weist ihn ab und verweist „unten" auf die Liste.
describe('InventurLaeufe', () => {
	it('nennt den Grund, wenn die laufenden Inventuren nicht kamen, und lädt neu', async () => {
		const a = ablauf({ nichtGeladen: { laeufe: 'Die Datenbank antwortet nicht', frueher: '' } });
		const screen = render(InventurLaeufe, { ablauf: a });

		expect(screen.getByText('Laufende Inventuren nicht geladen')).toBeTruthy();
		expect(screen.getByText('Die Datenbank antwortet nicht')).toBeTruthy();
		await fireEvent.click(screen.getByRole('button', { name: 'Erneut versuchen' }));
		expect(a.ladeOffeneLaeufe).toHaveBeenCalled();
	});

	it('nennt den Grund, wenn die früheren Inventuren nicht kamen, und lädt neu', async () => {
		const a = ablauf({ nichtGeladen: { laeufe: '', frueher: 'Zeitüberschreitung' } });
		const screen = render(InventurLaeufe, { ablauf: a });

		expect(screen.getByText('Frühere Inventuren nicht geladen')).toBeTruthy();
		await fireEvent.click(screen.getByRole('button', { name: 'Erneut versuchen' }));
		expect(a.ladeFruehereInventuren).toHaveBeenCalled();
	});

	it('zeigt die Listen mit ihren Knöpfen, wenn sie kamen', async () => {
		const lauf = { session_id: 's1', label: 'Gesamtbestand', erfasst: 3, erwartet: 9 };
		const frueher = {
			session_id: 's0',
			label: 'LMF Deu 7',
			abgeschlossen_am: '2026-09-01T10:00:00Z',
			erfasst: 40,
			verluste: 2,
			verworfen: false
		};
		const a = ablauf({ offeneLaeufe: [lauf], abgeschlosseneInventuren: [frueher] });
		const screen = render(InventurLaeufe, { ablauf: a });

		expect(screen.getByText('Laufende Inventuren')).toBeTruthy();
		expect(screen.getByText('2 fehlend')).toBeTruthy();
		await fireEvent.click(screen.getByRole('button', { name: 'Fortsetzen' }));
		expect(a.setzeLaufFort).toHaveBeenCalledWith(lauf);
		await fireEvent.click(screen.getByRole('button', { name: 'Fehlbestand' }));
		expect(a.zeigeFrueherenFehlbestand).toHaveBeenCalledWith(frueher);
	});
});
