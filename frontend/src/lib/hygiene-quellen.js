// Gemeinsame Quelldatei-Suche für die Hygiene-Ratschen (frontend-hygiene.test.js,
// frontend-hygiene-icons.test.js). Liegt bewusst NICHT in einer `.test.js`:
//
// Die Ratschen führen Bestandslisten mit Komponentennamen. Der Verwaisten-Test in
// frontend-hygiene.test.js hält jede Komponente für importiert, deren Dateiname
// irgendwo im Quellbaum auftaucht — stünde eine Bestandsliste in einer gesammelten
// Datei, wären alle darin genannten Komponenten dauerhaft vor ihm versteckt.
// Deshalb: Listen nur in `.test.js` (die überspringt der Sammler), hier nur Logik.

import { readdirSync, statSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { dirname, join, relative } from 'node:path';

const libDir = dirname(fileURLToPath(import.meta.url));
export const srcRoot = join(libDir, '..');
export const repoFrontend = join(srcRoot, '..');

/** @param {string} p @returns {string[]} */
export function sammleQuelldateien(p) {
	/** @type {string[]} */
	const out = [];
	for (const entry of readdirSync(p)) {
		if (entry === 'node_modules') continue;
		const full = join(p, entry);
		if (statSync(full).isDirectory()) out.push(...sammleQuelldateien(full));
		else if (/\.(svelte|js)$/.test(entry) && !entry.endsWith('.test.js')) out.push(full);
	}
	return out;
}

/** Pfad relativ zu `frontend/`, mit Schrägstrichen — so stehen sie in den Listen.
 * @param {string} f */
export const relPfad = (f) => relative(repoFrontend, f).replaceAll('\\', '/');

/**
 * Ratschen-Vergleich: Was ist neu dazugekommen, was ist inzwischen sauber?
 * @param {string[]} betroffen @param {string[]} bestand
 */
export function vergleicheMitBestand(betroffen, bestand) {
	return {
		neu: betroffen.filter((f) => !bestand.includes(f)),
		inzwischenSauber: bestand.filter((f) => !betroffen.includes(f))
	};
}

/**
 * Quelltext ohne Kommentare — HTML-Kommentare, Blockkommentare, Zeilenkommentare.
 *
 * Ein Detektor, der die Begründung für die Sache hält, meldet ewig „alles gut"
 * (Bugklasse „Lügende Ratsche", docs/sweeps.md). Bis 06.09.2026 stand dieselbe
 * Ersetzung in zwei Tests je einmal; CodeQL (Alerts #27, #28: „Incomplete
 * multi-character sanitization") wies darauf hin, dass EIN Durchlauf nicht reicht —
 * `<!-- <!-- -->` lässt nach der ersten Runde wieder einen Kommentar stehen. Deshalb
 * wird ersetzt, bis sich nichts mehr ändert.
 * @param {string} quelle
 * @returns {string}
 */
export function ohneKommentare(quelle) {
	let code = quelle;
	for (;;) {
		const vorher = code;
		code = code
			.replace(/<!--[\s\S]*?-->/g, '')
			.replace(/\/\*[\s\S]*?\*\//g, '')
			.replace(/^\s*\/\/.*$/gm, '');
		if (code === vorher) return code;
	}
}
