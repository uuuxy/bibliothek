import { describe, it, expect } from 'vitest';
import { render } from '@testing-library/svelte';
import BuchakteHistorie from './BuchakteHistorie.svelte';

// Der Reiter nennt je Ausleihe Leser, Klasse, Exemplar und ob das Buch zurück ist. Der Test
// hält das fest, damit eine Umstellung der Oberfläche keinen Wert verliert.
describe('BuchakteHistorie', () => {
	it('nennt Leser, Klasse, Exemplar und den Stand der Rückgabe', () => {
		const screen = render(BuchakteHistorie, {
			verlauf: [
				{
					schueler_name: 'Ida',
					schueler_nachname: 'Imker',
					klasse: '07A',
					exemplar_barcode: 'B-1',
					ausgeliehen_am: '2026-09-01',
					rueckgabe_am: '2026-09-20'
				},
				{
					schueler_name: 'Ole',
					schueler_nachname: 'Ohm',
					klasse: '08B',
					exemplar_barcode: 'B-2',
					ausgeliehen_am: '2026-09-15',
					rueckgabe_am: null
				}
			]
		});

		expect(screen.getByText('Letzte 2 Ausleihen')).toBeTruthy();
		// Wie der Browser: Leerraum aus dem Markup zählt als ein Leerzeichen.
		const [ida, ole] = screen
			.getAllByRole('listitem')
			.map((z) => (z.textContent ?? '').replace(/\s+/g, ' '));
		expect(ida).toContain('Ida Imker (07A)');
		expect(ida).toContain('Exemplar: B-1');
		expect(ida).toContain('Zurück 20.9.2026');
		expect(ole).toContain('Noch ausgeliehen');
	});

	it('sagt, wenn es noch keine Ausleihe gab', () => {
		const screen = render(BuchakteHistorie, { verlauf: [] });
		expect(screen.getByText('Noch keine Ausleihen in der Datenbank vorhanden.')).toBeTruthy();
	});
});
