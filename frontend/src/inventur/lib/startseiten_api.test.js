import { describe, it, expect } from 'vitest';
import { buecherSuchen } from './startseiten_api.js';

const buch = (/** @type {any} */ felder) => ({ id: Math.random().toString(), ...felder });

// Filter, Sortierung und Signatur-Suche der Buch-Suche (02.09.2026): Der Reiter hieß
// „Suche & Filter", hatte aber keinen Filter; die Signatur (Regaladresse) fand die
// Suche nicht, obwohl der Payload sie trug; Titel ohne Exemplare waren von komplett
// verliehenen nicht zu unterscheiden.
const bestand = [
	buch({
		title: 'Faust',
		author: 'Goethe',
		subject: 'Deutsch',
		signatur: 'Deu Goe',
		verfuegbar: 2,
		gesamt: 3,
		jahrgangVon: 9,
		jahrgangBis: 10,
		track: 'Gymnasium'
	}),
	buch({
		title: 'Analysis',
		subject: 'Mathematik',
		signatur: 'Mat Ana',
		verfuegbar: 0,
		gesamt: 5,
		gradeLevel: 11
	}),
	buch({ title: 'Ohne Bestand', subject: 'Deutsch', signatur: '', verfuegbar: 0, gesamt: 0 }),
	buch({ title: 'Hörbuch', subject: 'Englisch', medientyp: 'CD', verfuegbar: 1, gesamt: 1 })
];

describe('buecherSuchen: Signatur', () => {
	it('findet ein Buch über seine Signatur (Regaladresse)', () => {
		expect(buecherSuchen(bestand, 'goe').map((b) => b.title)).toEqual(['Faust']);
	});
});

// Schlagworte (docs/OFFEN.md 4.20): Welche Wörter einen Titel finden, sagt der Server je
// Titel — seine Schlagworte und die Verweise darauf (`suchwoerter`, siehe
// repository.SuchwoerterDerTitel). Die Suche prüft sie wie Titel und Autor.
describe('buecherSuchen: Schlagworte und Verweise', () => {
	const katalog = [
		buch({
			title: 'Drachenreiter',
			author: 'Cornelia Funke',
			suchwoerter: ['Drachen', 'Fantasy', 'Tierfantasy']
		}),
		buch({ title: 'Krabat', author: 'Otfried Preußler' })
	];
	const titel = (/** @type {string} */ q) => buecherSuchen(katalog, q).map((b) => b.title);

	it('findet über ein Schlagwort, auch angefangen', () => {
		expect(titel('fanta')).toEqual(['Drachenreiter']);
	});
	it('findet über einen Verweis auf ein Schlagwort', () => {
		expect(titel('Tierfantasy')).toEqual(['Drachenreiter']);
	});
	it('jeder Begriff darf ein anderes Feld treffen', () => {
		expect(titel('funke fantasy')).toEqual(['Drachenreiter']);
	});
	it('ein Titel ohne Schlagworte wird weiter über seine Felder gefunden', () => {
		expect(titel('krabat')).toEqual(['Krabat']);
	});
});
