import { describe, it, expect } from 'vitest';
import { render } from '@testing-library/svelte';
import LeserAusleihenListe from './LeserAusleihenListe.svelte';
import LeserAusleihenKarte from './LeserAusleihenKarte.svelte';

// Ein verliehenes Gerät steht in der Akte wie ein Buch: zurückgeben und Verlust oder Schaden
// melden gehen wie beim Buch (OFFEN 5.202 entschieden: das Gerät wird ausgesondert).
describe('Ausleihliste: Gerät', () => {
	/** @param {boolean} geraet */
	const zeile = (geraet) => ({
		id: 'x-1',
		ausleihe_id: 'a-1',
		barcode_id: 'G-1',
		titel: geraet ? 'iPad 9' : 'Der Zauberberg',
		ist_geraet: geraet,
		ausgeliehen_am: '2026-11-01T10:00:00Z',
		rueckgabe_frist: '2026-11-15T10:00:00Z'
	});
	const aktionen = { onReturnClick: () => {}, onDamageClick: () => {} };

	it('bietet am Gerät „Verlust oder Schaden melden“ und nennt die Rückgabe des Geräts', () => {
		const screen = render(LeserAusleihenListe, { ausleihen: [zeile(true)], ...aktionen });
		expect(screen.getByRole('button', { name: 'Verlust oder Schaden melden' })).toBeTruthy();
		expect(screen.getByRole('button', { name: 'Gerät zurückgeben' })).toBeTruthy();
	});

	it('behält am Buch beide Aktionen', () => {
		const screen = render(LeserAusleihenListe, { ausleihen: [zeile(false)], ...aktionen });
		expect(screen.getByRole('button', { name: 'Verlust oder Schaden melden' })).toBeTruthy();
		expect(screen.getByRole('button', { name: 'Buch zurückgeben' })).toBeTruthy();
	});

	// Zu einem Gerät gibt es kein Cover; der Knopf dafür öffnete nur einen Platzhalter.
	it('zeigt am Gerät das Gerätesymbol statt eines Covers, am Buch das Cover', () => {
		const geraet = render(LeserAusleihenListe, { ausleihen: [zeile(true)], ...aktionen });
		expect(geraet.queryByRole('button', { name: /^Cover von/ })).toBeNull();
		expect(geraet.container.querySelector('tbody [data-icon="devices"]')).not.toBeNull();
		geraet.unmount();

		const buch = render(LeserAusleihenListe, { ausleihen: [zeile(false)], ...aktionen });
		expect(buch.getByRole('button', { name: 'Cover von Der Zauberberg anzeigen' })).toBeTruthy();
	});

	it('zählt in der Überschrift Ausleihen, weil Geräte mitgezählt werden', () => {
		const screen = render(LeserAusleihenKarte, {
			ausleihen: [zeile(false), { ...zeile(true), id: 'x-2', ausleihe_id: 'a-2' }]
		});
		expect(screen.getByRole('heading', { name: '2 Ausleihen' })).toBeTruthy();
		expect(screen.getByRole('table', { name: 'Ausleihen' })).toBeTruthy();
	});
});
