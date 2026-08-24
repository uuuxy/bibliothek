/**
 * Gate: Welche Druck-Sektion des Ausweis-Designers ist bei welcher Betriebsart sichtbar?
 *
 * PrintPreview.svelte legt VIER versteckte Sektionen ins Dokument — Kartendrucker und
 * A4-Bogen, je Vorder- und Rückseite. Welche davon auf dem Papier landet, entscheidet
 * keine Bedingung im Markup, sondern allein die CSS-Kaskade aus drei Quellen:
 * Tailwinds `hidden`/`print:block`, druck-grundlagen.css und druck-ausweise.css. Das
 * ist am gebauten Bundle prüfbar und sonst nirgends — jsdom kennt keine Kaskade, und
 * `@media print` sieht man am Bildschirm ohnehin nicht.
 *
 * Anlass (24.08.2026): Der Rückseiten-Testdruck gab BEIDE Formen zugleich aus. Die
 * Auswahl nach Betriebsart gab es nur für die Vorderseiten-Sektionen; die Rückseiten
 * wurden allein nach Seite gefiltert, und `.print-rendered-output { display: block
 * !important }` holte die übrig gebliebene aktiv hervor. Der Vorderseitendruck war nie
 * betroffen — deshalb ist es monatelang niemandem aufgefallen.
 *
 * Der Prüfaufbau baut das Markup NICHT nach, sondern liest die vier Klassenlisten aus
 * PrintPreview.svelte und die Svelte-Scope-Klasse aus dem gebauten CSS. Eine umbenannte
 * Sektion lässt das Gate deshalb abbrechen, statt weiter eine Aussage über Klassen zu
 * treffen, die es nicht mehr gibt.
 *
 * Aufruf:  npm run build && node scripts/druck-sektionen-gate.mjs
 */
import { chromium } from '@playwright/test';
import http from 'node:http';
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const wurzel = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const dist = path.join(wurzel, 'dist');

/** Bricht mit Erklärung ab — ein Gate darf sich nie still überspringen. */
function abbruch(grund) {
	console.error(`✗ Druck-Sektionen-Gate: ${grund}`);
	process.exit(1);
}

// --- 1. Die echten Klassenlisten aus der Komponente lesen ---------------------------
const quelle = path.join(wurzel, 'src/lib/designer/PrintPreview.svelte');
if (!fs.existsSync(quelle)) abbruch(`${quelle} gibt es nicht mehr.`);
const markup = fs.readFileSync(quelle, 'utf8');

/** @type {Record<string,string>} */
const klassen = {};
for (const [, liste] of markup.matchAll(/class="(print-rendered-output[^"]*)"/g)) {
	const name = liste.match(/print-section-[a-z0-9-]+/)?.[0];
	if (name) klassen[name] = liste;
}
const ERWARTET = [
	'print-section-card',
	'print-section-a4',
	'print-section-back-card',
	'print-section-back-a4'
];
for (const name of ERWARTET) {
	if (!klassen[name])
		abbruch(`PrintPreview.svelte führt keine Sektion "${name}" mehr — Gate veraltet.`);
}

// --- 2. Svelte-Scope-Klasse aus dem gebauten CSS holen ------------------------------
const cssDatei = fs
	.readdirSync(path.join(dist, 'assets'), { withFileTypes: false })
	.filter((f) => f.endsWith('.css'))
	.map((f) => path.join('assets', f))[0];
if (!cssDatei) abbruch('Kein gebautes CSS in dist/assets — vorher "npm run build" laufen lassen.');
const css = fs.readFileSync(path.join(dist, cssDatei), 'utf8');
const scope = css.match(/print-section-card\.(svelte-[a-z0-9]+)/)?.[1];
if (!scope) abbruch('Im gebauten CSS steht keine gescopte Regel für .print-section-card mehr.');

// --- 3. Seitengerüst wie in App.svelte, mit den echten Klassen ----------------------
const sektionen = ERWARTET.map(
	(name) =>
		`<div id="${name}" class="${klassen[name]} ${scope}"><div class="print-card-box">x</div></div>`
).join('\n');
const seite = `<!doctype html><html><head><meta charset="utf-8">
<link rel="stylesheet" href="/${cssDatei}"></head><body>
<main class="min-h-screen bg-surface"><div class="h-screen flex w-full overflow-hidden">
<aside class="no-print h-screen w-64">Navigation</aside>
<div class="flex w-full min-w-0 flex-1 flex-col overflow-y-auto px-4 py-6">
<main class="flex-1 overflow-y-auto flex flex-col w-full"><div class="w-full h-full">
<div id="bedienung" class="w-full no-print">Werkzeugleiste und Leinwand</div>
${sektionen}
</div></main></div></div></main></body></html>`;

// --- 4. Unter @media print messen ---------------------------------------------------
const server = http.createServer((anfrage, antwort) => {
	const pfad = decodeURIComponent(anfrage.url.split('?')[0]);
	if (pfad === '/pruefseite.html') {
		antwort.setHeader('Content-Type', 'text/html; charset=utf-8');
		return antwort.end(seite);
	}
	try {
		antwort.setHeader('Content-Type', 'text/css');
		antwort.end(fs.readFileSync(path.join(dist, pfad)));
	} catch {
		antwort.statusCode = 404;
		antwort.end('');
	}
});
await new Promise((fertig) => server.listen(0, fertig));

// Genau EINE Sektion je Betriebsart — nicht "mindestens die richtige".
const MATRIX = [
	['card', 'front', 'print-section-card'],
	['card', 'back', 'print-section-back-card'],
	['a4', 'front', 'print-section-a4'],
	['a4', 'back', 'print-section-back-a4']
];

const browser = await chromium.launch();
const seiteImBrowser = await browser.newPage();
await seiteImBrowser.goto(`http://127.0.0.1:${server.address().port}/pruefseite.html`);
await seiteImBrowser.emulateMedia({ media: 'print' });

const fehler = [];
for (const [modus, seitenwahl, erwartet] of MATRIX) {
	await seiteImBrowser.evaluate(
		([m, s]) => {
			document.body.setAttribute('data-print-mode', m);
			document.body.setAttribute('data-print-side', s);
		},
		[modus, seitenwahl]
	);

	const sichtbar = await seiteImBrowser.evaluate(
		(ids) => ids.filter((id) => getComputedStyle(document.getElementById(id)).display !== 'none'),
		ERWARTET
	);
	if (sichtbar.length !== 1 || sichtbar[0] !== erwartet) {
		fehler.push(
			`  ${modus}/${seitenwahl}: erwartet [${erwartet}], gedruckt würde [${sichtbar.join(', ') || 'nichts'}]`
		);
	}
}

// Die Bedienoberfläche darf nie mitdrucken — dieselbe Kaskade, andere Richtung.
const bedienungSichtbar = await seiteImBrowser.evaluate(
	() => getComputedStyle(document.getElementById('bedienung')).display !== 'none'
);
if (bedienungSichtbar)
	fehler.push('  Die Bedienoberfläche (.no-print) landet mit auf dem Ausdruck.');

await browser.close();
server.close();

if (fehler.length > 0) {
	console.error(
		'✗ Druck-Sektionen-Gate: falsche Sichtbarkeit unter @media print\n' + fehler.join('\n')
	);
	process.exit(1);
}
console.log(
	'✓ Druck-Sektionen-Gate: je Betriebsart genau eine Sektion, Bedienung bleibt außen vor.'
);
