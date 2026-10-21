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

/** Alle `.test.js` unter p: das Gegenstück zu sammleQuelldateien, das sie auslässt.
 * @param {string} p @returns {string[]} */
export function sammleTestdateien(p) {
	/** @type {string[]} */
	const out = [];
	for (const entry of readdirSync(p)) {
		if (entry === 'node_modules') continue;
		const full = join(p, entry);
		if (statSync(full).isDirectory()) out.push(...sammleTestdateien(full));
		else if (entry.endsWith('.test.js')) out.push(full);
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
 * Alle Elemente eines Tags mit Kopf und Inhalt. Der Kopf endet am ersten „>" außerhalb von
 * Klammern und Anführungszeichen: In onclick={() => …} steht selbst eines.
 * @param {string} quelle
 * @param {'button' | 'Button'} tag
 */
export function elemente(quelle, tag) {
	/** @type {{ start: number, kopf: string, inhalt: string }[]} */
	const gefunden = [];
	for (let start = quelle.indexOf(`<${tag}`); start !== -1;) {
		let i = start + tag.length + 1;
		const istTag = !/[A-Za-z0-9_]/.test(quelle[i] ?? '');
		let tiefe = 0;
		let zitat = '';
		for (; istTag && i < quelle.length; i++) {
			const z = quelle[i];
			if (zitat) zitat = z === zitat ? '' : zitat;
			else if (z === '"' || z === "'" || z === '`') zitat = z;
			else if (z === '{') tiefe++;
			else if (z === '}') tiefe--;
			else if (z === '>' && tiefe === 0) break;
		}
		const ende = quelle.indexOf(`</${tag}>`, i);
		if (istTag && ende !== -1 && quelle[i - 1] !== '/') {
			gefunden.push({ start, kopf: quelle.slice(start, i), inhalt: quelle.slice(i + 1, ende) });
		}
		start = quelle.indexOf(`<${tag}`, i);
	}
	return gefunden;
}

/**
 * Quelltext ohne Kommentare. Die Ratschen prüfen erst danach: Ein Detektor, der die Begründung
 * für die Sache hält, meldet sonst „alles gut" (Bugklasse „Lügende Ratsche", docs/sweeps.md).
 * @param {string} quelle
 * @returns {string}
 */
export function ohneKommentare(quelle) {
	return zerlege(quelle)
		.filter((t) => !t.kommentar)
		.map((t) => t.text)
		.join('');
}

/**
 * Die Zeilen der Kommentare, je Zeile ein Eintrag; die erste Zeile eines Kommentars beginnt mit
 * seinem Zeichen (`//`, `/*` oder `<!--`).
 * @param {string} quelle
 * @returns {string[]}
 */
export function kommentarZeilen(quelle) {
	return zerlege(quelle)
		.filter((t) => t.kommentar)
		.flatMap((t) => t.text.split('\n'));
}

/**
 * Zerlegt den Quelltext in Code und Kommentare (HTML-, Block- und ganze Zeilenkommentare), in
 * der Reihenfolge des Textes.
 *
 * Ein Scanner statt `replace`: CodeQL bewertet jede Ersetzung für sich und meldet sie als
 * unvollständig („Incomplete multi-character sanitization"); was der Scanner übersprungen hat,
 * kann keinen neuen Kommentaranfang bilden. Ein Kommentar ohne Ende bleibt Code, und `//` zählt
 * nur am Zeilenanfang, damit `https://…` in einer Code-Zeile stehen bleibt.
 * @param {string} quelle
 * @returns {{ kommentar: boolean, text: string }[]}
 */
function zerlege(quelle) {
	/** @type {{ kommentar: boolean, text: string }[]} */
	const teile = [];
	let code = '';
	let zeilenanfang = true;
	let i = 0;
	while (i < quelle.length) {
		const sprung = kommentarEnde(quelle, i, zeilenanfang);
		if (sprung > i) {
			teile.push(
				{ kommentar: false, text: code },
				{ kommentar: true, text: quelle.slice(i, sprung) }
			);
			code = '';
			i = sprung;
			zeilenanfang = false;
			continue;
		}
		const z = quelle[i++];
		code += z;
		zeilenanfang = z === '\n' || (zeilenanfang && /\s/.test(z));
	}
	teile.push({ kommentar: false, text: code });
	return teile;
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
