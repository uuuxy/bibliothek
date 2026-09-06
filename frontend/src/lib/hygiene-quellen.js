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
 * Quelltext ohne Kommentare — HTML-Kommentare, Blockkommentare, ganze Zeilenkommentare.
 *
 * Ein Detektor, der die Begründung für die Sache hält, meldet ewig „alles gut"
 * (Bugklasse „Lügende Ratsche", docs/sweeps.md). Bis 06.09.2026 stand dieselbe
 * Regex-Ersetzung in zwei Tests je einmal; CodeQL (#27, #28) meldete, dass EIN
 * `replace`-Durchlauf im Allgemeinen nicht reicht. (Das Beispiel `<!-- <!-- -->`, das
 * hier bis zum 06.09.2026 als Begründung stand, belegt es NICHT: non-greedy frisst die
 * ganze Zeichenkette. Nachgemessen im Rasterdurchgang desselben Tages — eine Begründung,
 * die man nicht nachrechnet, ist selbst eine lügende Ratsche.) Eine Schleife bis zum
 * Fixpunkt half auch nicht (#29) — CodeQL bewertet
 * jeden `replace`-Aufruf für sich und sieht die Schleife nicht. Deshalb kein `replace`
 * mehr: Ein Scanner läuft einmal von vorn nach hinten und ÜBERSPRINGT jeden Kommentar
 * vom Anfang bis zum ersten Ende. Was übersprungen ist, kann keinen neuen Anfang bilden.
 * Ein Kommentar ohne Ende bleibt stehen (wie bisher). Zeilenkommentare zählen nur, wenn
 * vor dem `//` nichts als Leerraum steht — `https://…` in einer Code-Zeile bleibt.
 * @param {string} quelle
 * @returns {string}
 */
export function ohneKommentare(quelle) {
	let out = '';
	let zeilenanfang = true;
	let i = 0;
	while (i < quelle.length) {
		const sprung = kommentarEnde(quelle, i, zeilenanfang);
		if (sprung > i) {
			i = sprung;
			zeilenanfang = false;
			continue;
		}
		const z = quelle[i++];
		out += z;
		zeilenanfang = z === '\n' || (zeilenanfang && /\s/.test(z));
	}
	return out;
}

/**
 * Index hinter dem Kommentar, der bei `i` beginnt — oder `i`, wenn dort keiner beginnt
 * (oder er kein Ende hat).
 * @param {string} q @param {number} i @param {boolean} zeilenanfang @returns {number}
 */
function kommentarEnde(q, i, zeilenanfang) {
	if (q.startsWith('<!--', i)) {
		const ende = q.indexOf('-->', i + 4);
		return ende === -1 ? i : ende + 3;
	}
	if (q.startsWith('/*', i)) {
		const ende = q.indexOf('*/', i + 2);
		return ende === -1 ? i : ende + 2;
	}
	if (zeilenanfang && q.startsWith('//', i)) {
		const ende = q.indexOf('\n', i);
		return ende === -1 ? q.length : ende;
	}
	return i;
}
