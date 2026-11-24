import { describe, it, expect } from 'vitest';
import { render, fireEvent, waitFor } from '@testing-library/svelte';
import BuchakteAusleiher from './BuchakteAusleiher.svelte';

// Dieselbe Regel wie an der Theke (utils/suchnorm.js): ohne Akzente, ohne Bindestrich, Vor- und
// Nachname in beliebiger Reihenfolge.
describe('Buchakte, Reiter „Ausleiher“: die Suche', () => {
	/** @param {string} nachname @param {string} ausweis */
	const ausleiherMit = (nachname, ausweis) => ({
		schueler_name: 'Mia',
		schueler_nachname: nachname,
		klasse: '7b',
		schueler_barcode: ausweis,
		exemplar_barcode: `E-${ausweis}`,
		ausgeliehen_am: '2026-01-02T10:00:00Z',
		rueckgabe_frist: '2026-02-01T10:00:00Z'
	});

	it.each([
		['ozturk', 'Öztürk', 'Al-Kahil'],
		['alkahil', 'Al-Kahil', 'Öztürk'],
		['kahil mia', 'Al-Kahil', 'Öztürk']
	])('findet zu „%s“ %s', async (suche, gesucht, anderer) => {
		const screen = render(BuchakteAusleiher, {
			ausleiher: [ausleiherMit('Öztürk', 'A-1'), ausleiherMit('Al-Kahil', 'A-2')],
			titel: { title: 'Die Räuber' },
			onBack: () => {}
		});
		await fireEvent.input(screen.getByRole('searchbox', { name: 'Nach Name filtern' }), {
			target: { value: suche }
		});
		await waitFor(() => expect(screen.container.textContent).not.toContain(anderer));
		expect(screen.container.textContent).toContain(gesucht);
	});
});

// Der Knopf druckt alle, die den Titel gerade haben, auch die ohne Verzug. Die Mahnliste
// mit nur den Überfälligen steht im Mahnwesen.
describe('Buchakte, Reiter „Ausleiher“: Druckknopf', () => {
	const ausleiher = {
		schueler_name: 'Lena',
		schueler_nachname: 'Groß',
		klasse: '7b',
		schueler_barcode: 'A-1',
		exemplar_barcode: 'B-1',
		ausgeliehen_am: '2026-01-02T10:00:00Z',
		rueckgabe_frist: '2026-02-01T10:00:00Z'
	};

	it('heißt „Liste drucken“ und nennt keine Mahnliste', () => {
		const screen = render(BuchakteAusleiher, {
			ausleiher: [ausleiher],
			titel: { title: 'Die Räuber' },
			onBack: () => {}
		});
		expect(screen.getByRole('button', { name: 'Liste drucken' })).toBeTruthy();
		expect(screen.container.textContent ?? '').not.toContain('Mahnliste');
	});

	it('schreibt die Daten zweistellig wie jedes Datum im Programm (utils/format.js)', () => {
		const screen = render(BuchakteAusleiher, {
			ausleiher: [ausleiher],
			titel: { title: 'Die Räuber' },
			onBack: () => {}
		});
		const text = screen.container.textContent ?? '';
		expect(text).toContain('02.01.2026');
		expect(text).toContain('01.02.2026');
	});
});
