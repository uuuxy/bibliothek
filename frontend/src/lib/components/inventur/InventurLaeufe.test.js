import { describe, it, expect, vi } from 'vitest';
import { render, fireEvent, cleanup } from '@testing-library/svelte';

vi.mock('../../stores/bestaetigung.svelte.js', () => ({
	bestaetigen: vi.fn(),
	loeschenBestaetigen: vi.fn()
}));

import { bestaetigen, loeschenBestaetigen } from '../../stores/bestaetigung.svelte.js';
import InventurLaeufe from './InventurLaeufe.svelte';

/** Die Fassade, soweit die Abschnitte sie lesen. @param {Record<string, any>} [anders] */
const ablauf = (anders = {}) => ({
	offeneLaeufe: [],
	frueher: frueheres(),
	nichtGeladen: { laeufe: '', frueher: '' },
	ladeFruehereLaeuft: false,
	ladeOffeneLaeufe: vi.fn(),
	setzeLaufFort: vi.fn(),
	verwerfeLauf: vi.fn(),
	zeigeFrueherenFehlbestand: vi.fn(),
	...anders
});

/** Die Liste der früheren Inventuren. @param {Record<string, any>} [anders] */
const frueheres = (anders = {}) => ({
	laeufe: [],
	weitere: 0,
	laedtMehr: false,
	lade: vi.fn(),
	ladeMehr: vi.fn(),
	loesche: vi.fn(),
	...anders
});

const fruehereInventur = {
	session_id: 's0',
	label: 'LMF Deu 7',
	abgeschlossen_am: '2026-09-01T10:00:00Z',
	erfasst: 40,
	verluste: 2,
	verworfen: false
};

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
		expect(a.frueher.lade).toHaveBeenCalled();
	});

	it('zeigt die Listen mit ihren Knöpfen, wenn sie kamen', async () => {
		const lauf = { session_id: 's1', label: 'Gesamtbestand', erwartet: 9, fehlend: 6 };
		const frueher = fruehereInventur;
		const a = ablauf({ offeneLaeufe: [lauf], frueher: frueheres({ laeufe: [frueher] }) });
		const screen = render(InventurLaeufe, { ablauf: a });

		expect(screen.getByText('Laufende Inventuren')).toBeTruthy();
		// Gefunden sind die erwarteten Bücher, die nicht mehr fehlen, wie der Abschluss sie zählt.
		expect(screen.getByText(/^3 \/ 9 erfasst/)).toBeTruthy();
		expect(screen.getByText('2 fehlend')).toBeTruthy();
		await fireEvent.click(screen.getByRole('button', { name: 'Fortsetzen' }));
		expect(a.setzeLaufFort).toHaveBeenCalledWith(lauf);
		await fireEvent.click(screen.getByRole('button', { name: 'Fehlbestand' }));
		expect(a.zeigeFrueherenFehlbestand).toHaveBeenCalledWith(frueher);
	});

	// „Verwerfen" steht neben „Fortsetzen" und lässt sich nicht zurücknehmen: Erst die Rückfrage,
	// und „Abbrechen" lässt die Inventur laufen.
	it('verwirft eine laufende Inventur erst nach der Rückfrage', async () => {
		const lauf = { session_id: 's1', label: 'Gesamtbestand', erwartet: 9, fehlend: 6 };
		const a = ablauf({ offeneLaeufe: [lauf] });
		const screen = render(InventurLaeufe, { ablauf: a });
		const verwerfen = screen.getByRole('button', { name: 'Verwerfen' });

		vi.mocked(bestaetigen).mockResolvedValueOnce(false);
		await fireEvent.click(verwerfen);
		await new Promise((r) => setTimeout(r));
		expect(bestaetigen).toHaveBeenCalledWith(
			expect.objectContaining({ titel: 'Inventur „Gesamtbestand“ verwerfen?', gefaehrlich: true })
		);
		expect(a.verwerfeLauf).not.toHaveBeenCalled();

		vi.mocked(bestaetigen).mockResolvedValueOnce(true);
		await fireEvent.click(verwerfen);
		await vi.waitFor(() => expect(a.verwerfeLauf).toHaveBeenCalledWith(lauf));
	});

	// Ältere Inventuren stehen nicht still außerhalb der Liste: „Mehr laden" nennt ihre Zahl und
	// holt sie; ohne weitere steht kein Knopf da.
	it('holt mit „Mehr laden" die älteren Inventuren und nennt ihre Zahl', async () => {
		const a = ablauf({ frueher: frueheres({ laeufe: [fruehereInventur], weitere: 12 }) });
		const screen = render(InventurLaeufe, { ablauf: a });

		await fireEvent.click(screen.getByRole('button', { name: 'Mehr laden (12 weitere)' }));
		expect(a.frueher.ladeMehr).toHaveBeenCalled();

		cleanup();
		const ohne = render(InventurLaeufe, {
			ablauf: ablauf({ frueher: frueheres({ laeufe: [fruehereInventur] }) })
		});
		expect(ohne.queryByRole('button', { name: /^Mehr laden/ })).toBeNull();
	});

	// Löschen steht im Menü der Zeile und fragt vorher; die Rückfrage nennt, was mitgeht und was
	// am Buch bleibt.
	it('löscht eine frühere Inventur erst nach der Rückfrage', async () => {
		const a = ablauf({ frueher: frueheres({ laeufe: [fruehereInventur] }) });
		const screen = render(InventurLaeufe, { ablauf: a });
		const loeschen = async () => {
			await fireEvent.click(screen.getByRole('button', { name: 'Aktionen für „LMF Deu 7“' }));
			await fireEvent.click(await screen.findByRole('menuitem', { name: 'Löschen' }));
		};

		vi.mocked(loeschenBestaetigen).mockResolvedValueOnce(false);
		await loeschen();
		await vi.waitFor(() =>
			expect(loeschenBestaetigen).toHaveBeenCalledWith(
				'Inventur „LMF Deu 7“ löschen?',
				'Ihr Fehlbestandsbericht wird mit gelöscht. Die 2 fehlenden Bücher bleiben als verloren gebucht; ein Buch, das sich findet, bucht ein Scan an der Theke zurück. Das lässt sich nicht rückgängig machen.'
			)
		);
		expect(a.frueher.loesche).not.toHaveBeenCalled();

		vi.mocked(loeschenBestaetigen).mockResolvedValueOnce(true);
		await loeschen();
		await vi.waitFor(() => expect(a.frueher.loesche).toHaveBeenCalledWith(fruehereInventur));
	});
});
