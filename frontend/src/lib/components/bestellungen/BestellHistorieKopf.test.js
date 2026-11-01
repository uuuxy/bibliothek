import { describe, it, expect, vi } from 'vitest';
import { render } from '@testing-library/svelte';

vi.mock('../../stores/bestellStore.svelte.js', () => ({
	bestellStore: { preiseErfassen: true }
}));

import BestellHistorieKopf from './BestellHistorieKopf.svelte';

// Der Kopf der Bestellhistorie zeigt, woraus die Gesamtzahl besteht (#596): Für das
// Schulamt zählt der Landes-Anteil, für den Schulträger seiner. Und er trägt den Filter,
// mit dem sich die Liste auf einen Topf einengen lässt — auch auf die Alt-Bestellungen
// ohne Zuordnung, die einzige Stelle, an der man sie zum Nachtragen findet.

const euro = (/** @type {number} */ n) => `${n.toFixed(2).replace('.', ',')} €`;
const topfName = (/** @type {string} */ w) =>
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
	topfName,
	onFilterWechsel: () => {}
};

describe('BestellHistorieKopf', () => {
	// Die Zahl sinkt ohne Zutun, wenn der Händler bestätigt. Ein Bildschirmleser liest nur vor, was
	// in einer Region dazukommt, die schon dastand; ein bloßes Verschwinden der Zeile bliebe stumm.
	it('liest vor, wenn die Zahl der wartenden Bestellungen sinkt, auch auf null', async () => {
		const screen = render(BestellHistorieKopf, {
			...basis,
			offeneBestaetigungen: 2,
			aufteilung: []
		});
		const region = screen.container.querySelector('[aria-live="polite"]');
		expect(region?.textContent).toContain('2 Bestellungen warten noch');

		await screen.rerender({ offeneBestaetigungen: 1 });
		expect(screen.container.querySelector('[aria-live="polite"]')).toBe(region);
		expect(region?.textContent).toContain('1 Bestellung wartet noch');

		await screen.rerender({ offeneBestaetigungen: 0 });
		expect(screen.container.querySelector('[aria-live="polite"]')).toBe(region);
		expect(region?.textContent?.trim()).toBe(
			'Keine Bestellung wartet auf die Bestätigung des Händlers'
		);
		expect(region?.querySelector('.sr-only')).toBeTruthy();
	});

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
