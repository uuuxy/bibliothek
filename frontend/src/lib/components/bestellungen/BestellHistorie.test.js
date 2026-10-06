import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, fireEvent, waitFor } from '@testing-library/svelte';

vi.mock('../../apiFetch.js', async (original) => ({
	.../** @type {any} */ (await original()),
	apiGet: vi.fn()
}));

import { apiGet } from '../../apiFetch.js';
import BestellHistorie from './BestellHistorie.svelte';

const holen = vi.mocked(apiGet);
const SUMMEN = {
	gesamt: 1,
	gesamtbetrag: 0,
	gesamt_exemplare: 2,
	offene_bestaetigungen: 0,
	nach_mittel: []
};
/** @param {string} lieferant @param {string} mittel */
const bestellung = (lieferant, mittel) => ({
	id: lieferant,
	bestelldatum: '2026-10-01T08:00:00Z',
	lieferant_name: lieferant,
	lieferant_email: 'handel@example.org',
	kundennummer: '',
	mittel,
	anzahl_exemplare: 2,
	gesamtbetrag: 0
});

describe('BestellHistorie', () => {
	beforeEach(() => vi.clearAllMocks());

	// Leer heißt leer, ein Ladefehler heißt Ladefehler: Vorher blieb „Lade Bestellhistorie…"
	// für immer stehen, und nach einem gescheiterten Nachladen stand die alte Liste unter dem
	// neuen Filter.
	it('zeigt nach einem gescheiterten Laden den Ladefehler mit dem Weg zurück', async () => {
		holen.mockRejectedValue(new Error('Server nicht erreichbar'));
		const screen = render(BestellHistorie);

		const meldung = await screen.findByRole('alert');
		expect(meldung.textContent).toContain('Bestellhistorie nicht geladen');
		expect(screen.queryByText(/Lade Bestellhistorie/)).toBeNull();
		expect(screen.queryByText(/Noch keine Bestellungen aufgegeben/)).toBeNull();

		holen.mockReset();
		holen.mockImplementation(async (url) =>
			String(url).includes('uebersicht') ? SUMMEN : [bestellung('Buchhandlung Nord', 'land')]
		);
		await fireEvent.click(screen.getByRole('button', { name: 'Erneut versuchen' }));
		expect(await screen.findByText('Buchhandlung Nord')).toBeTruthy();
	});

	// Zwei Filterwechsel kurz nacheinander: Es gilt die Antwort auf den jüngsten, auch wenn
	// die ältere später eintrifft.
	it('lässt eine späte Antwort auf den vorigen Filter die Liste nicht ersetzen', async () => {
		/** @type {(liste: any[]) => void} */
		let landFertig = () => {};
		holen.mockImplementation((url) => {
			const adresse = String(url);
			if (adresse.includes('uebersicht')) return Promise.resolve(SUMMEN);
			if (adresse.includes('mittel=land')) return new Promise((fertig) => (landFertig = fertig));
			if (adresse.includes('mittel=schultraeger'))
				return Promise.resolve([bestellung('Buchhandlung Süd', 'schultraeger')]);
			return Promise.resolve([bestellung('Buchhandlung Nord', 'land')]);
		});
		const screen = render(BestellHistorie);
		await screen.findByText('Buchhandlung Nord');

		const filter = screen.getByRole('combobox');
		await fireEvent.click(filter);
		await fireEvent.click(screen.getByRole('option', { name: 'Lernmittelfreiheit (Land)' }));
		await fireEvent.click(filter);
		await fireEvent.click(screen.getByRole('option', { name: 'Schülerbücherei (Schulträger)' }));
		await screen.findByText('Buchhandlung Süd');

		landFertig([bestellung('Späte Antwort', 'land')]);
		await waitFor(() => expect(holen.mock.calls.length).toBeGreaterThanOrEqual(6));
		await Promise.resolve();

		expect(screen.queryByText('Späte Antwort')).toBeNull();
		expect(screen.getByText('Buchhandlung Süd')).toBeTruthy();
	});
});
