import { describe, it, expect } from 'vitest';
import { readdirSync, readFileSync, statSync } from 'node:fs';
import { join } from 'node:path';
import { srcRoot, relPfad, ohneKommentare } from './hygiene-quellen.js';

// Wer die Theke im Test bedient, räumt ihre Zeitgeber weg.
//
// Die Bugklasse dahinter (CI am 16.09.2026, davor am 11.09.2026 mit keyboardNav): Der
// Lauf war ROT bei 128 grünen Dateien und 697 grünen Tests. Grund war kein Test, sondern
// ein Zeitgeber — `scanfeldWiederScharfstellen` plant einen Fokussprung über 50 ms, der
// `document` anfasst. Endet die Testdatei vorher, baut Vitest jsdom ab, und der Rückruf
// läuft ohne Seite: „Unhandled Errors: ReferenceError: document is not defined". Ein
// einziger solcher Rückruf färbt den GANZEN Lauf, und die Meldung zeigt auf die
// Testdatei, in der es zufällig knallte, nicht auf den Timer.
//
// Der Fix in der Quelle (ein Handle je Zeitgeber, `stoppeZeitgeber`) macht das Aufräumen
// MÖGLICH — getan werden muss es trotzdem. Genau daran scheiterte die Lösung vom
// 11.09.2026 ein zweites Mal: Sie stand in einer Datei, und die nächste wusste nichts
// davon. Deshalb dieser Detektor statt einer Erinnerung.
//
// Erlaubt sind zwei Wege, und beide beenden das Problem wirklich:
//   - `vi.useFakeTimers()` — dann gibt es nach dem Test keine echten Timer mehr.
//   - `stoppeZeitgeber()` — der Store löscht sie selbst (Vorbild: omniboxZeitgeber.test.js).
const ERLAUBT = ['useFakeTimers', 'stoppeZeitgeber'];

/** @param {string} p @returns {string[]} */
function sammleTestdateien(p) {
	/** @type {string[]} */
	const out = [];
	for (const eintrag of readdirSync(p)) {
		if (eintrag === 'node_modules') continue;
		const voll = join(p, eintrag);
		if (statSync(voll).isDirectory()) out.push(...sammleTestdateien(voll));
		else if (eintrag.endsWith('.test.js')) out.push(voll);
	}
	return out;
}

describe('Zeitgeber der Theke in Tests', () => {
	it('jede Testdatei, die den Omnibox-Store lädt, räumt seine Zeitgeber weg', () => {
		/** @type {string[]} */
		const ohneAufraeumen = [];
		let gepruefte = 0;

		for (const datei of sammleTestdateien(srcRoot)) {
			// Kommentare zuerst weg: Ein Detektor, der die Begründung für die Sache hält,
			// meldet ewig „alles gut" (Bugklasse „Lügende Ratsche").
			const quelle = ohneKommentare(readFileSync(datei, 'utf8'));
			// Nur ECHTE Importe zählen — eine Ratsche, die den Dateinamen bloss erwähnt,
			// bedient den Store nicht und plant keinen Zeitgeber.
			if (!/import\s*\{[^}]*\}\s*from\s*'[^']*omnibox\.svelte\.js'/.test(quelle)) continue;
			gepruefte++;
			if (!ERLAUBT.some((wort) => quelle.includes(wort))) ohneAufraeumen.push(relPfad(datei));
		}

		// Ohne diese Zusicherung wäre ein umbenannter Store ein still grüner Test.
		expect(
			gepruefte,
			'keine einzige Testdatei lädt den Store — der Detektor misst nichts'
		).toBeGreaterThan(0);

		expect(
			ohneAufraeumen,
			`Diese Testdateien bedienen die Theke, räumen ihre Zeitgeber aber nicht weg:\n` +
				ohneAufraeumen.map((f) => `  ${f}`).join('\n') +
				`\nEntweder vi.useFakeTimers() setzen oder afterEach(() => store.stoppeZeitgeber()).\n` +
				`Sonst feuert ein Rückruf nach dem Abbau von jsdom und färbt den GANZEN Lauf rot,\n` +
				`obwohl jeder Test grün ist (CI 16.09.2026, siehe stores/omniboxZeitgeber.test.js).`
		).toEqual([]);
	});
});
