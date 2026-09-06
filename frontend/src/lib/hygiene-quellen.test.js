// Gate für hygiene-quellen.ohneKommentare — die eine Stelle, an der die Quelltext-Ratschen
// Kommentare ausblenden, bevor sie prüfen. CodeQL #27–#29 („Incomplete multi-character
// sanitization"): eine Regex-Ersetzung ließ aus `<!-- <!-- -->` wieder einen Kommentar
// entstehen. Der Fall steht hier als Test, damit ein Rückbau auf `replace` rot wird.
import { describe, expect, it } from 'vitest';
import { ohneKommentare } from './hygiene-quellen.js';

describe('ohneKommentare', () => {
	// Rasterdurchgang 06.09.2026 (Frage 7), am Rückbau GEMESSEN: Diese drei Zeilen sind
	// KEIN Beleg für den Scanner. Die verworfene Regex-Fassung
	// (`/<!--[\s\S]*?-->/g` und Geschwister) besteht sie ebenfalls — non-greedy frisst bei
	// `<!-- <!-- -->` die ganze Zeichenkette, es bleibt kein Kommentar stehen. Rot wird
	// die Regex-Fassung an den beiden Tests darunter (Leerraum vor `//` bleibt erhalten;
	// ein Zeilenkommentar frisst keinen Blockanfang der nächsten Zeile) — DIE halten den
	// Scanner fest. Der Fall hier bleibt als Beschreibung stehen, nicht als Beweis.
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
