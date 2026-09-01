import { describe, it, expect } from 'vitest';
import { readFileSync, readdirSync } from 'node:fs';
import { join } from 'node:path';
import { srcRoot } from './hygiene-quellen.js';

// Ratsche gegen stille Wächter in den e2e-Läufen.
//
// `if (await knopf.isVisible()) { ... }` fragt einen Zustand in genau dem Moment ab, in
// dem die Antwort des Servers vielleicht noch unterwegs ist. Playwright wartet hier
// NICHT — anders als bei `expect(...)`. Der Zweig wird übersprungen, und der Test läuft
// weiter, als wäre nichts gewesen. Zwei Folgen, beide schlecht:
//
//   * Der erwartete Zustand kommt eine Sekunde später: Der Test scheitert drei Zeilen
//     tiefer mit einer Meldung, die vom falschen Ort erzählt.
//   * Der Zweig gehört gar nicht mehr in den Ablauf (die Bremse greift nicht mehr, der
//     Dialog ist weg): Der Test läuft grün durch und behauptet, alles sei in Ordnung.
//     Das ist die teurere Variante — ein Gate, das nicht mehr rot werden kann.
//
// Erlaubt ist die Verzweigung deshalb nur NACH einem echten Warten auf eines von beiden
// möglichen Ergebnissen:
//
//     await expect(erfolg.or(bremse)).toBeVisible();
//     if (await bremse.isVisible()) { await bremse.click(); }
//
// Gefunden am 23.08.2026 beim Raster-Durchgang (Frage 7, Gate-Ehrlichkeit): zwei solche
// Wächter in admin-lusd.spec.js.
//
// Die Regex fasste bis zum 01.09.2026 nur die historische Form `await bremse.isVisible()`:
// Ihr `[^)]*` kam nicht über die innere Klammer von `page.locator('#x')` hinweg, und die
// Negation `if (!(await …))` begann nicht mit `await`. Der Rot-Beweis-Sweep belegte vier
// durchrutschende Formen (verkettet, getByRole, negiert, negiert mit .catch). Jetzt zählt
// jede if-Bedingung mit einem awaited isVisible() — die Selbstprobe unten hält alle Formen.
const VERZWEIGUNG = /if\s*\(.*\bawait\b.*\.isVisible\(\)/;
const WARTEN = /\.or\(/;

// Begründete Ausnahmen — jede ist eine bewusste Entscheidung, kein Freifahrtschein.
const AUSNAHMEN = [
	{
		datei: 'e2e/icon-tooltips.spec.js',
		muster: /isVisible\(\)\.catch\(\(\) => false\)/,
		grund:
			'Scan-Schleife über ALLE Symbol-Kandidaten eines Bildschirms: unsichtbare überspringen ist ' +
			'dort der Zweck, kein stiller Wächter. Gegen das Ins-Leere-Laufen schützt die Mindestzahl ' +
			'expect(geprueft).toBeGreaterThan(5) am Testende.'
	}
];

/** Alle e2e-Spezifikationen einlesen. */
function specDateien() {
	const verzeichnis = join(srcRoot, '..', 'e2e');
	return readdirSync(verzeichnis)
		.filter((n) => n.endsWith('.spec.js'))
		.map((n) => ({ name: `e2e/${n}`, inhalt: readFileSync(join(verzeichnis, n), 'utf8') }));
}

describe('e2e-Wächter', () => {
	it('findet überhaupt Spezifikationen (Gegenprobe am Detektor)', () => {
		const dateien = specDateien();
		expect(dateien.length).toBeGreaterThan(10);
	});

	it('erkennt alle Formen der Verzweigung (Selbstprobe am Muster)', () => {
		// Die vier unteren Formen rutschten bis zum 01.09.2026 durch (siehe Kopfkommentar).
		const verboten = [
			'if (await bremse.isVisible()) {',
			"if (await page.locator('#x').isVisible()) {",
			"if (await page.getByRole('button', { name: 'X' }).isVisible()) {",
			'if (!(await b.isVisible())) continue;',
			'if (!(await b.isVisible().catch(() => false))) continue;'
		];
		for (const zeile of verboten) expect(VERZWEIGUNG.test(zeile), zeile).toBe(true);

		// Kein Treffer ohne Verzweigung bzw. ohne await — expect(...) wartet ja.
		const erlaubt = ['await expect(blase).toBeVisible();', 'const sichtbar = b.isVisible();'];
		for (const zeile of erlaubt) expect(VERZWEIGUNG.test(zeile), zeile).toBe(false);
	});

	it('verzweigt nie auf isVisible(), ohne vorher auf eines von zwei Ergebnissen zu warten', () => {
		const verstoesse = [];

		for (const { name, inhalt } of specDateien()) {
			const zeilen = inhalt.split('\n');
			zeilen.forEach((zeile, i) => {
				// Kommentarzeilen sind kein Code — dieser Test erklärt sich selbst mit
				// genau dem Muster, das er verbietet, und würde sich sonst selbst treffen.
				const nackt = zeile.trim();
				if (nackt.startsWith('//') || nackt.startsWith('*')) return;
				if (!VERZWEIGUNG.test(zeile)) return;
				if (AUSNAHMEN.some((a) => a.datei === name && a.muster.test(zeile))) return;
				// Die zehn Zeilen davor müssen ein `.or(`-Warten enthalten.
				const davor = zeilen.slice(Math.max(0, i - 10), i).join('\n');
				if (!WARTEN.test(davor)) {
					verstoesse.push(`${name}:${i + 1}  ${zeile.trim()}`);
				}
			});
		}

		expect(
			verstoesse,
			`Verzweigung auf isVisible() ohne vorheriges Warten — der Zweig kann still ` +
				`übersprungen werden:\n${verstoesse.join('\n')}`
		).toEqual([]);
	});
});
