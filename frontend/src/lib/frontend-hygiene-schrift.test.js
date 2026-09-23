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
				'ui/Abschnitt (text-base font-medium), Knopftext wie ui/Button (text-sm font-semibold).'
		).toEqual([]);
	});
});
