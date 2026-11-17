import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { join } from 'node:path';
import { srcRoot, sammleQuelldateien, relPfad, ohneKommentare } from './hygiene-quellen.js';

// Ratsche: Schriftrollen nur, wenn die Skala sie kennt.
//
// Anlass (23.09.2026): Zwölf Klassen wie `text-title-medium`, `text-body-small` und
// `text-label-large` in sieben Dateien (LMF-Plan, Sommerferien, Erreichbarkeit, Portal,
// Klassendruck). Die Skala in styles/theme-mass.css kennt von den M3-Rollen nur
// `text-label-small`; alle anderen Rollen stehen dort unter den Tailwind-Namen (text-sm =
// body-medium, text-base = body-large, text-lg = title-large …, CLAUDE.md). Eine Klasse
// ohne Definition erzeugt kein CSS — die Überschrift erbte still die Größe ihres
// Behälters (im Browser an zehn Stellen gemessen: 16 px, auch wo 12 px gemeint waren). CLAUDE.md
// nannte die Regel („gibt es nicht und wäre stumm"), aber nichts prüfte sie.
//
// Die erlaubten Rollen liest der Test aus theme-mass.css (`--text-<rolle>:`): Wer dort eine
// Rolle anlegt, darf sie ab dann benutzen, ohne diesen Test anzufassen.
//
// BLINDHEIT: Nur literale Klassen der Form text-<display|headline|title|body|label>-<small|
// medium|large>. Eine zusammengesetzte Klasse (`'text-' + rolle`), eine Schriftgröße in
// einer .css-Datei oder im style-Attribut sieht er nicht.
//
// Rot bewiesen am 23.09.2026 gegen den Bestand vor der Umstellung (12 Fundstellen).
const ROLLE = /\btext-(?:display|headline|title|body|label)-(?:small|medium|large)\b/g;

function bekannteRollen() {
	const theme = readFileSync(join(srcRoot, 'styles', 'theme-mass.css'), 'utf8');
	const namen = [...theme.matchAll(/--text-([a-z]+-(?:small|medium|large)):/g)].map((m) => m[1]);
	return new Set(namen.map((n) => `text-${n}`));
}

describe('Schriftrollen nur aus der Skala', () => {
	it('liest die Skala: text-label-small ist bekannt', () => {
		// Nicht-leer-Garantie: Fände der Leser keine Rolle, wäre jede Klasse ein Fund — und
		// ein zerbrochenes Muster sähe aus wie ein strenges.
		expect(bekannteRollen().has('text-label-small')).toBe(true);
	});

	it('benutzt keine Rolle, die theme-mass.css nicht definiert', () => {
		const bekannt = bekannteRollen();
		const funde = [];
		let dateien = 0;
		for (const datei of sammleQuelldateien(srcRoot)) {
			if (!datei.endsWith('.svelte') && !datei.endsWith('.js')) continue;
			dateien++;
			const quelle = ohneKommentare(readFileSync(datei, 'utf8'));
			for (const [klasse] of quelle.matchAll(ROLLE)) {
				if (!bekannt.has(klasse)) funde.push(`${relPfad(datei)}: ${klasse}`);
			}
		}
		expect(dateien, 'Der Sammler findet kaum Quelldateien').toBeGreaterThan(200);
		expect(
			funde,
			'Diese Schriftrolle gibt es nicht und wirkt nicht. Die Skala: text-xs = body-small, ' +
				'text-sm = body-medium, text-base = body-large, text-lg = title-large, ' +
				'text-xl = headline-small, dazu text-label-small (CLAUDE.md). Überschriften wie ' +
				'ui/Abschnitt (text-base font-semibold), Knopftext wie ui/Button (text-sm font-semibold).'
		).toEqual([]);
	});
});

// Die Beschriftung eines Felds steht in body-medium (text-sm) wie in ui/Feld.svelte, die
// Hinweiszeile unter einer Überschrift ebenso (CLAUDE.md, „Hinweiszeile“). Standen 12 px und
// 14 px in derselben Maske, sahen gleiche Dinge verschieden aus. Ausgenommen sind Zeilen unter
// dem Titel eines Buchs in einer Liste oder Karte: Das ist eine Angabe, kein Hinweis.
//
// Sieht nicht: Klassen aus Variablen, eine Hinweiszeile, die nicht direkt auf die Überschrift
// folgt, und eine Beschriftung, die kein <label> ist.
const LABEL_KLEIN = /<label\b(?:[^>{]|\{(?:[^{}]|\{[^{}]*\})*\})*?\bclass="[^"]*\btext-xs\b/g;
const HINWEIS_KLEIN = /<\/h[1-6]>\s*<p\b[^>]*?\bclass="[^"]*\btext-xs\b/g;
const ANGABE_UNTER_BUCHTITEL = new Set([
	'src/lib/components/bestellungen/BedarfZeile.svelte',
	'src/lib/components/bestellungen/WarenkorbPosition.svelte',
	'src/lib/components/portal/PortalTrefferkarte.svelte'
]);

describe('Beschriftung und Hinweiszeile in body-medium', () => {
	const dateien = sammleQuelldateien(srcRoot).filter((f) => f.endsWith('.svelte'));
	/** @param {RegExp} muster @param {Set<string>} [ausnahmen] */
	const funde = (muster, ausnahmen = new Set()) =>
		dateien.flatMap((f) => {
			const quelle = ohneKommentare(readFileSync(f, 'utf8'));
			return ausnahmen.has(relPfad(f)) ? [] : [...quelle.matchAll(muster)].map(() => relPfad(f));
		});

	it('keine Beschriftung eines Felds in 12 px', () => {
		expect(funde(LABEL_KLEIN), '<label> in text-xs — text-sm wie ui/Feld.svelte').toEqual([]);
	});

	it('keine Hinweiszeile unter einer Überschrift in 12 px', () => {
		expect(
			funde(HINWEIS_KLEIN, ANGABE_UNTER_BUCHTITEL),
			'Hinweiszeile in text-xs — text-sm text-on-surface-variant (CLAUDE.md)'
		).toEqual([]);
		for (const datei of ANGABE_UNTER_BUCHTITEL) {
			const quelle = readFileSync(`${srcRoot}/../${datei}`, 'utf8');
			expect([...quelle.matchAll(HINWEIS_KLEIN)].length, `${datei}: Ausnahme ohne Fundstelle`).toBe(
				1
			);
		}
	});

	it('Selbstprobe: die Detektoren fassen die Formen', () => {
		const zaehl = (/** @type {RegExp} */ m, /** @type {string} */ s) => [...s.matchAll(m)].length;
		expect(zaehl(LABEL_KLEIN, '<label for="x" class="text-xs font-medium">')).toBe(1);
		expect(zaehl(LABEL_KLEIN, '<label\n\tfor={id}\n\tclass="block text-xs"\n>')).toBe(1);
		expect(zaehl(LABEL_KLEIN, '<label for="x" class="text-sm font-medium">')).toBe(0);
		expect(
			zaehl(HINWEIS_KLEIN, '<h3 class="x">Titel</h3>\n\t<p class="mt-1 text-xs">Satz</p>')
		).toBe(1);
		expect(zaehl(HINWEIS_KLEIN, '<h3>Titel</h3>\n<p class="text-sm">Satz</p>')).toBe(0);
	});
});

// Googles Skala gibt Überschriften (title-medium, 16 px) und der kleinsten Beschriftung
// (label-small, 11 px) das Gewicht Medium: „textAppearanceTitleMedium | Medium 16sp“,
// „textAppearanceLabelSmall | Medium 11sp“ (material-components-android, Typography.md). In
// diesem Haus zeigt font-medium auf 400 (theme-mass.css); Medium heißt font-semibold. Die Rolle
// label-small trägt ihr Gewicht selbst.
//
// Sieht nicht: Überschriften, deren Größe nicht text-base heißt, und Klassen aus Variablen.
const UEBERSCHRIFT_REGULAR =
	/<h[1-6]\b[^>]*?\bclass="[^"]*\btext-base\b[^"]*\bfont-(?:medium|normal)\b/g;
const UEBERSCHRIFT_REGULAR_VORN =
	/<h[1-6]\b[^>]*?\bclass="[^"]*\bfont-(?:medium|normal)\b[^"]*\btext-base\b/g;
const LABEL_SMALL_REGULAR =
	/class="(?=[^"]*\btext-label-small\b)[^"]*\bfont-(?:medium|normal)\b[^"]*"/g;

describe('Überschrift und kleinste Beschriftung in Medium', () => {
	const dateien = sammleQuelldateien(srcRoot).filter((f) => /\.(svelte|js)$/.test(f));
	/** @param {RegExp[]} muster */
	const funde = (...muster) =>
		dateien.flatMap((f) => {
			const q = ohneKommentare(readFileSync(f, 'utf8'));
			return muster.flatMap((m) => [...q.matchAll(m)].map(() => relPfad(f)));
		});

	it('keine Überschrift in 16 px mit 400', () => {
		expect(
			funde(UEBERSCHRIFT_REGULAR, UEBERSCHRIFT_REGULAR_VORN),
			'Überschrift text-base mit font-medium (= 400) — font-semibold (= 500, M3 Medium)'
		).toEqual([]);
	});

	it('keine kleinste Beschriftung mit 400', () => {
		expect(
			funde(LABEL_SMALL_REGULAR),
			'text-label-small mit font-medium (= 400) — das Gewicht weglassen, die Rolle trägt 500'
		).toEqual([]);
	});

	it('Selbstprobe: die Detektoren fassen die Formen', () => {
		const z = (/** @type {RegExp} */ m, /** @type {string} */ s) => [...s.matchAll(m)].length;
		expect(z(UEBERSCHRIFT_REGULAR, '<h3 class="mb-4 text-base font-medium text-on-surface">')).toBe(
			1
		);
		expect(z(UEBERSCHRIFT_REGULAR_VORN, '<h2 class="font-medium text-base">')).toBe(1);
		expect(z(UEBERSCHRIFT_REGULAR, '<h3 class="text-base font-semibold">')).toBe(0);
		expect(z(LABEL_SMALL_REGULAR, 'class="text-label-small font-medium text-on-error"')).toBe(1);
		expect(z(LABEL_SMALL_REGULAR, 'class="font-medium text-label-small"')).toBe(1);
		expect(z(LABEL_SMALL_REGULAR, 'class="text-label-small text-on-surface-variant"')).toBe(0);
	});
});
