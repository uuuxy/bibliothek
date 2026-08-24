/**
 * Gate: Welche Druck-Sektion des Ausweis-Designers ist bei welcher Betriebsart sichtbar?
 *
 * PrintPreview.svelte legt versteckte Sektionen ins Dokument — je eine für Vorder- und
 * Rückseite der Ausweiskarte. Welche davon auf dem Papier landet, entscheidet keine
 * Bedingung im Markup, sondern allein die CSS-Kaskade aus drei Quellen:
 * Tailwinds `hidden`/`print:block`, druck-grundlagen.css und druck-ausweise.css. Das
 * ist am gebauten Bundle prüfbar und sonst nirgends — jsdom kennt keine Kaskade, und
 * `@media print` sieht man am Bildschirm ohnehin nicht.
 *
 * Anlass (24.08.2026): Damals gab es zusätzlich zwei A4-Sektionen (acht Kartenabbilder
 * auf einem Blatt), und der Rückseiten-Testdruck gab BEIDE Formen zugleich aus — die
 * Auswahl nach Betriebsart fehlte den Rückseiten, und `.print-rendered-output
 * { display: block !important }` holte die übrig gebliebene aktiv hervor. Der
 * Vorderseitendruck war nie betroffen, deshalb fiel es monatelang niemandem auf.
 *
 * Der A4-Bogen ist seitdem abgeschafft (an seiner Stelle steht der Etikettenbogen, der
 * serverseitig als PDF entsteht). Das Gate bleibt: Sobald wieder eine zweite Sektion
 * dazukommt, ist die Frage „welche druckt eigentlich?" sofort wieder da.
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
	// Nur Zeichen, die in Tailwind-Klassenlisten vorkommen: Die Liste landet unten
	// wörtlich in der Prüfseite — alles andere wäre Markup-Injektion in das Gate
	// (CodeQL #22) und ohnehin ein Zeichen, dass der Regex daneben gegriffen hat.
	if (!/^[-\w :./![\]%]+$/.test(liste)) {
		abbruch(`Klassenliste mit unerwarteten Zeichen in PrintPreview.svelte: "${liste}"`);
	}
	const name = liste.match(/print-section-[a-z0-9-]+/)?.[0];
	if (name) klassen[name] = liste;
}
const ERWARTET = ['print-section-card', 'print-section-back-card'];
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
		// In dist/ einsperren (CodeQL #23): Der Server lebt zwar nur Sekunden und nur
		// auf Loopback, aber ein "/../"-Pfad las sonst beliebige Dateien des Rechners.
		const ziel = path.resolve(dist, '.' + pfad);
		if (ziel !== dist && !ziel.startsWith(dist + path.sep)) throw new Error('ausserhalb von dist');
		antwort.setHeader('Content-Type', 'text/css');
		antwort.end(fs.readFileSync(ziel));
	} catch {
		antwort.statusCode = 404;
		antwort.end('');
	}
});
// Loopback statt 0.0.0.0: Die Prüfseite geht nur den eigenen Headless-Browser an,
// nicht das Netzwerk, in dem der Rechner gerade hängt.
await new Promise((fertig) => server.listen(0, '127.0.0.1', fertig));

// Genau EINE Sektion je Fall — nicht "mindestens die richtige".
//
// 'a4' steht bewusst weiter drin, obwohl kein Bildschirm ihn mehr setzt: Ein zentral
// gespeichertes Design kann den alten Wert noch tragen, und dann darf NICHTS gedruckt
// werden statt irgendetwas. applyDesign() liest ihn zwar auf 'card' um — aber genau
// solche Umleitungen fallen bei Umbauten als Erstes weg.
// Die letzten drei Zeilen sind der eigentliche Wert: Der Kartenstapel des Designers
// darf NUR in seiner eigenen Betriebsart aufs Papier. Die Schülerdatei kann gleichzeitig
// einen markierten Stapel und ein geöffnetes Profil im DOM haben — ohne diese Zusicherung
// nimmt „Quittung drucken" (gar keine Betriebsart) oder „Ausweis drucken" im Profil
// ('card-single') den ganzen Stapel mit. 'a4' ist der abgeschaffte A4-Bogen: Ein zentral
// gespeichertes Design kann ihn noch tragen, und dann darf NICHTS gedruckt werden.
const MATRIX = [
	['card', 'front', 'print-section-card'],
	['card', 'back', 'print-section-back-card'],
	['card-single', 'front', null],
	['a4', 'front', null],
	[null, null, null]
];

const browser = await chromium.launch();
const seiteImBrowser = await browser.newPage();
await seiteImBrowser.goto(`http://127.0.0.1:${server.address().port}/pruefseite.html`);
await seiteImBrowser.emulateMedia({ media: 'print' });

const fehler = [];
for (const [modus, seitenwahl, erwartet] of MATRIX) {
	await seiteImBrowser.evaluate(
		([m, s]) => {
			// null = Attribut gar nicht gesetzt (blankes window.print(), z. B. Quittung).
			if (m === null) document.body.removeAttribute('data-print-mode');
			else document.body.setAttribute('data-print-mode', m);
			if (s === null) document.body.removeAttribute('data-print-side');
			else document.body.setAttribute('data-print-side', s);
		},
		[modus, seitenwahl]
	);

	const sichtbar = await seiteImBrowser.evaluate(
		(ids) => ids.filter((id) => getComputedStyle(document.getElementById(id)).display !== 'none'),
		ERWARTET
	);
	const sollen = erwartet ? [erwartet] : [];
	if (sichtbar.join(',') !== sollen.join(',')) {
		fehler.push(
			`  ${modus ?? 'ohne Modus'}/${seitenwahl ?? '–'}: erwartet [${sollen.join(', ') || 'nichts'}], gedruckt würde [${sichtbar.join(', ') || 'nichts'}]`
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
