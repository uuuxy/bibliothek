// Gate für hygiene-quellen.ohneKommentare — die eine Stelle, an der die Quelltext-Ratschen
// Kommentare ausblenden, bevor sie prüfen. CodeQL #27–#29 („Incomplete multi-character
// sanitization"): eine Regex-Ersetzung ließ aus `<!-- <!-- -->` wieder einen Kommentar
// entstehen. Der Fall steht hier als Test, damit ein Rückbau auf `replace` rot wird.
import { describe, expect, it } from 'vitest';
import { ohneKommentare } from './hygiene-quellen.js';

describe('ohneKommentare', () => {
	it('lässt aus verschachtelten Anfängen keinen neuen Kommentar entstehen (CodeQL #29)', () => {
		expect(ohneKommentare('<!-- <!-- -->')).toBe('');
		expect(ohneKommentare('/* /* */')).toBe('');
		expect(ohneKommentare('a<!-- <!-- -->b')).toBe('ab');
	});

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
