import { describe, it, expect } from 'vitest';
import {
	menueEintraege,
	loeschFolgen,
	zaehlSatz,
	verweisWahl,
	dialogHinweis,
	loeschFrage,
	loeschErgebnis
} from './schlagwortPflege.js';

const wort = { id: 'a', wort: 'Fantasy', titel: 12, verweise: ['Tierfantasy'], ist_filter: false };
const verweis = {
	id: 'b',
	wort: 'Tierfantasy',
	titel: 0,
	verweis_auf_id: 'a',
	verweis_auf: 'Fantasy',
	verweise: [],
	ist_filter: false
};

describe('Schlagwort-Pflege: Anzeige-Regeln', () => {
	it('bietet am Verweis nur Umbenennen und Löschen an', () => {
		expect(menueEintraege(verweis).map((e) => e.id)).toEqual(['umbenennen', 'loeschen']);
		expect(menueEintraege(wort).map((e) => e.id)).toEqual([
			'umbenennen',
			'zusammenfuehren',
			'verweis',
			'loeschen'
		]);
	});

	it('nennt in der Rückfrage die Zahl der Titel und der Verweise', () => {
		expect(loeschFolgen(wort)).toBe(
			'12 Titel verlieren das Schlagwort, der Verweis darauf fällt mit. Das lässt sich nicht rückgängig machen.'
		);
		expect(loeschFolgen({ ...wort, verweise: ['X', 'Y'] })).toMatch(
			/, 2 Verweise darauf fallen mit\. /
		);
		expect(loeschFolgen({ ...wort, verweise: [] })).toMatch(
			/^12 Titel verlieren das Schlagwort\. /
		);
		expect(loeschFolgen(verweis)).toMatch(/nicht mehr bei „Fantasy“/);
	});

	it('zählt Wörter und Verweise getrennt', () => {
		expect(zaehlSatz({ gesamt: 12, verweise: 3 }, 2)).toBe(
			'9 Schlagworte · 3 Verweise · 2 als Filter im Portal'
		);
		expect(zaehlSatz({ gesamt: 2, verweise: 1 }, 0)).toBe(
			'1 Schlagwort · 1 Verweis · 0 als Filter im Portal'
		);
		expect(zaehlSatz({ gesamt: 5, verweise: 0 }, 1)).toBe('5 Schlagworte · 1 als Filter im Portal');
	});

	it('bietet „als Verweis behalten" beim Zusammenführen und am Wort an, nicht am Verweis', () => {
		expect(verweisWahl('zusammenfuehren', wort)).toBe(true);
		expect(verweisWahl('umbenennen', wort)).toBe(true);
		// am Verweis: eine weitere Schreibweise legt „Verweis anlegen" am Ziel an
		expect(verweisWahl('umbenennen', verweis)).toBe(false);
		expect(verweisWahl('verweis', wort)).toBe(false);
	});

	it('sagt im Dialog, was mit den Titeln geschieht', () => {
		expect(dialogHinweis('umbenennen', wort, 'Fantastik')).toBe(
			'Alle 12 Titel tragen danach die neue Schreibweise.'
		);
		expect(dialogHinweis('umbenennen', wort, 'tierfantasy')).toBe(
			'„Tierfantasy“ ist bisher ein Verweis auf „Fantasy“ und wird zum Schlagwort.'
		);
		expect(dialogHinweis('zusammenfuehren', wort, '')).toBe(
			'Die 12 Titel bekommen das gewählte Wort. Das lässt sich nicht rückgängig machen.'
		);
		expect(dialogHinweis('zusammenfuehren', { ...wort, titel: 0 }, '')).toBe('');
		// Einzahl: „Die 1 Titel bekommen" stand bis zum 23.09.2026 im Dialog
		expect(dialogHinweis('zusammenfuehren', { ...wort, titel: 1 }, '')).toBe(
			'Der Titel bekommt das gewählte Wort. Das lässt sich nicht rückgängig machen.'
		);
		expect(dialogHinweis('umbenennen', { ...wort, titel: 1 }, 'Fantastik')).toBe(
			'Der Titel trägt danach die neue Schreibweise.'
		);
		expect(loeschFolgen({ ...wort, titel: 1 })).toMatch(/^1 Titel verliert das Schlagwort, /);
		expect(dialogHinweis('verweis', wort, 'Fantasie')).toMatch(/bekommt „Fantasy“/);
	});
});

describe('Schlagwort-Pflege: mehrere löschen', () => {
	const mit = (w, titel, verweise = []) => ({ id: w, wort: w, titel, verweise, ist_filter: false });

	it('fragt bei einem Wort wie bisher', () => {
		expect(loeschFrage([wort])).toEqual({ titel: '„Fantasy“ löschen?', text: loeschFolgen(wort) });
	});

	it('nennt die Wörter, die Titelzahl und nur die Verweise, die nicht selbst markiert sind', () => {
		const frage = loeschFrage([
			mit('Magie', 3, ['Zauberei', 'Hexerei']),
			mit('Mühle', 1),
			{ ...verweis, wort: 'zauberei', verweis_auf: 'Magie' }
		]);
		expect(frage.titel).toBe('3 Schlagworte löschen?');
		expect(frage.text).toBe(
			'„Magie“, „Mühle“ und „zauberei“. Sie stehen zusammen 4-mal an Titeln. ' +
				'1 Verweis darauf fällt mit. Das lässt sich nicht rückgängig machen.'
		);
	});

	it('nennt sechs Wörter alle — „und 1 weitere" wäre kein Deutsch', () => {
		const sechs = ['A', 'B', 'C', 'D', 'E', 'F'].map((w) => mit(w, 0));
		expect(loeschFrage(sechs).text).toBe(
			'„A“, „B“, „C“, „D“, „E“ und „F“. Das lässt sich nicht rückgängig machen.'
		);
	});

	it('kürzt lange Auswahlen auf fünf Namen', () => {
		const sieben = ['A', 'B', 'C', 'D', 'E', 'F', 'G'].map((w) => mit(w, 0));
		expect(loeschFrage(sieben).text).toBe(
			'„A“, „B“, „C“, „D“, „E“ und 2 weitere. Das lässt sich nicht rückgängig machen.'
		);
	});

	it('meldet danach die Zahlen des Servers', () => {
		expect(loeschErgebnis([wort], { woerter: 1, titel: 12 })).toBe('„Fantasy“ gelöscht.');
		expect(loeschErgebnis([wort, verweis], { woerter: 2, titel: 1 })).toBe(
			'2 Schlagworte gelöscht. 1 Titel hat Schlagworte verloren.'
		);
		expect(loeschErgebnis([wort, verweis], { woerter: 2, titel: 0 })).toBe(
			'2 Schlagworte gelöscht.'
		);
	});
});
