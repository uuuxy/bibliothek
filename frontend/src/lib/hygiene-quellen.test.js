// Gate für hygiene-quellen.ohneKommentare — die eine Stelle, an der die Quelltext-Ratschen
// Kommentare ausblenden, bevor sie prüfen. CodeQL #27–#29 („Incomplete multi-character
// sanitization"): eine Regex-Ersetzung ließ aus `<!-- <!-- -->` wieder einen Kommentar
// entstehen. Der Fall steht hier als Test, damit ein Rückbau auf `replace` rot wird.
import { describe, expect, it } from 'vitest';
import { kommentarZeilen, ohneKommentare } from './hygiene-quellen.js';

describe('ohneKommentare', () => {
	// Dieser Fall beschreibt den Scanner, belegt ihn aber nicht: Eine Regex-Fassung
	// (`/<!--[\s\S]*?-->/g` und Geschwister) besteht ihn auch. Rot wird sie an den beiden
	// Tests darunter.
	it('entfernt HTML-, Block- und ganze Zeilenkommentare, auch über Zeilen hinweg', () => {
		const quelle = ['<!-- x\n y -->keep1', '/* a\n b */keep2', '  // nur Kommentar', 'keep3'].join(
			'\n'
		);
		expect(ohneKommentare(quelle)).toBe('keep1\nkeep2\n  \nkeep3');
	});

	it('lässt // mitten in einer Code-Zeile und Kommentare ohne Ende stehen', () => {
		expect(ohneKommentare("const u = 'https://x';")).toBe("const u = 'https://x';");
		expect(ohneKommentare('a <!-- offen')).toBe('a <!-- offen');
		expect(ohneKommentare('a /* offen')).toBe('a /* offen');
	});

	it('ein Zeilenkommentar frisst keinen Blockkommentar-Anfang aus der nächsten Zeile', () => {
		expect(ohneKommentare('// /*\ncode\n*/')).toBe('\ncode\n*/');
	});
});

describe('kommentarZeilen', () => {
	it('liefert die Zeilen jedes Kommentars, die erste mit seinem Zeichen', () => {
		const quelle = [
			'<!-- x\n y -->code',
			'/* a */code',
			'  // ganze Zeile',
			"const u = 'https://x'; // hinter Code"
		].join('\n');
		expect(kommentarZeilen(quelle)).toEqual(['<!-- x', ' y -->', '/* a */', '// ganze Zeile']);
	});
});
