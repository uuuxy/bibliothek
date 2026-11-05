import { describe, it, expect, vi } from 'vitest';
import { render, fireEvent, within } from '@testing-library/svelte';

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
	standBekannt: true,
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

	// Scheitert der Abruf oder läuft er noch, wartet vielleicht eine Bestellung.
	it('sagt nichts über wartende Bestellungen, solange der Stand nicht bekannt ist', async () => {
		const screen = render(BestellHistorieKopf, { ...basis, standBekannt: false, aufteilung: [] });
		const region = screen.container.querySelector('[aria-live="polite"]');
		expect(region?.textContent?.trim()).toBe('');

		await screen.rerender({ standBekannt: true });
		expect(screen.container.querySelector('[aria-live="polite"]')).toBe(region);
		expect(region?.textContent).toContain('Keine Bestellung wartet');
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
		expect(screen.getByText(/Lernmittelfreiheit:/)).toBeTruthy();
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

	// Filter-Chips wie im Bestandsbuch: je Topf ein Chip, auch für die Alt-Bestellungen ohne
	// Zuordnung. Alle aus heißt kein Filter, deshalb gibt es keinen Chip „Alle Mittel".
	it('bietet je Topf einen Chip an, auch für die Alt-Bestellungen ohne Zuordnung', () => {
		const screen = render(BestellHistorieKopf, { ...basis, aufteilung: [] });
		const chips = within(screen.getByRole('group', { name: 'Nach Mittelherkunft filtern' }));

		expect(chips.getAllByRole('button').map((b) => b.textContent?.trim())).toEqual([
			'Lernmittelfreiheit (Land)',
			'Schülerbücherei (Schulträger)',
			'ohne Zuordnung'
		]);
	});

	// Jede Wahl lädt die Liste neu, weil am Server gefiltert wird; ein zweiter Klick hebt sie auf.
	it('filtert mit einem Chip und hebt den Filter mit einem zweiten Klick auf', async () => {
		const onFilterWechsel = vi.fn();
		const screen = render(BestellHistorieKopf, { ...basis, aufteilung: [], onFilterWechsel });
		const land = screen.getByRole('button', { name: 'Lernmittelfreiheit (Land)' });

		await fireEvent.click(land);
		expect(land.getAttribute('aria-pressed')).toBe('true');
		expect(onFilterWechsel).toHaveBeenCalledTimes(1);

		await fireEvent.click(land);
		expect(land.getAttribute('aria-pressed')).toBe('false');
		expect(onFilterWechsel).toHaveBeenCalledTimes(2);
	});
});
