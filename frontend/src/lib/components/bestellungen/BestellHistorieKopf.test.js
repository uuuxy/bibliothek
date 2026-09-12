import { describe, it, expect, vi } from 'vitest';
import { render } from '@testing-library/svelte';

vi.mock('../../stores/orderStore.svelte.js', () => ({
	orderStore: { preiseErfassen: true }
}));

import BestellHistorieKopf from './BestellHistorieKopf.svelte';

// Der Kopf der Bestellhistorie zeigt, woraus die Gesamtzahl besteht (#596): Für das
// Schulamt zählt der Landes-Anteil, für den Schulträger seiner. Und er trägt den Filter,
// mit dem sich die Liste auf einen Topf einengen lässt — auch auf die Alt-Bestellungen
// ohne Zuordnung, die einzige Stelle, an der man sie zum Nachtragen findet.

const euro = (/** @type {number} */ n) => `${n.toFixed(2).replace('.', ',')} €`;
const topfLabel = (/** @type {string} */ w) =>
	({ land: 'Lernmittelfreiheit', schultraeger: 'Schülerbücherei' })[w] ?? 'ohne Zuordnung';

const basis = {
	offeneBestaetigungen: 0,
	gesamtsumme: 255.5,
	gesamtExemplare: 57,
	zeigeKennzahlen: true,
	mittel: '',
	mittelFilter: [
		{ value: '', label: 'Alle Mittel' },
		{ value: 'land', label: 'Lernmittelfreiheit (Land)' },
		{ value: 'schultraeger', label: 'Schülerbücherei (Schulträger)' },
		{ value: 'ohne', label: 'ohne Zuordnung' }
	],
	euro,
	topfLabel,
	onFilterWechsel: () => {}
};

describe('BestellHistorieKopf', () => {
	it('zeigt die Aufteilung auf die Töpfe unter der Gesamtsumme', () => {
		const screen = render(BestellHistorieKopf, {
			...basis,
			aufteilung: [
				{ mittel: 'land', gesamt: 2, gesamtbetrag: 200, gesamt_exemplare: 50 },
				{ mittel: '', gesamt: 1, gesamtbetrag: 10, gesamt_exemplare: 2 }
			]
		});

		expect(screen.getByText('255,50 €')).toBeTruthy();
		expect(screen.getByText(/Lernmittelfreiheit/)).toBeTruthy();
		expect(screen.getByText('200,00 €')).toBeTruthy();
		expect(screen.getByText(/ohne Zuordnung:/)).toBeTruthy();
	});

	// Bei nur einem Topf sagt die Aufteilung nichts, was die Zeile darüber nicht schon
	// sagt — dann bleibt sie weg.
	it('schweigt, wenn alles aus einem Topf kommt', () => {
		const screen = render(BestellHistorieKopf, {
			...basis,
			aufteilung: [{ mittel: 'land', gesamt: 3, gesamtbetrag: 255.5, gesamt_exemplare: 57 }]
		});

		expect(screen.queryByText(/Lernmittelfreiheit:/)).toBeNull();
	});

	// Das Auswahlfeld ist das hauseigene Select (Tastaturbedienung, ui/Select.svelte) —
	// eine Liste, die erst beim Öffnen entsteht. Geprüft wird deshalb, was nach dem Klick
	// zur Wahl steht.
	it('bietet den Filter samt der Alt-Bestellungen ohne Zuordnung an', async () => {
		const screen = render(BestellHistorieKopf, { ...basis, aufteilung: [] });

		screen.getByRole('combobox').click();
		await Promise.resolve();

		expect(screen.getByText('Lernmittelfreiheit (Land)')).toBeTruthy();
		expect(screen.getByText('Schülerbücherei (Schulträger)')).toBeTruthy();
		expect(screen.getByText('ohne Zuordnung')).toBeTruthy();
	});
});
