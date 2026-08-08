import { describe, it, expect } from 'vitest';
import { readFileSync, existsSync } from 'node:fs';
import { join } from 'node:path';
import { srcRoot, sammleQuelldateien } from './hygiene-quellen.js';

// Fuenfte Struktur-Invariante: Eine Route bringt kein eigenes Seitengeruest mit.
//
// Am 07.08.2026 tat das jede zweite: VIER Routen malten mit `bg-slate-50` eine zweite
// Flaeche ueber die Leinwand der App-Huelle, VIER verschiedene `max-w-*` legten vier
// Inhaltsbreiten fest. Auf dem Bildschirm sah dadurch jede Seite anders aus — mal
// getoente Arbeitsflaeche, mal weisse; mal 896 px breit, mal 1152, mal randlos.
//
// Die Zustaendigkeiten seitdem:
//   Flaeche  -> App.svelte (weiss, edge-to-edge)
//   Kopf     -> PageShell.svelte (Titel, Unterzeile, Aktionen)
// KEINE Breitenbegrenzung und KEINE Karten-Wrapper: f2320e1, e81ce75 und 95d5d33
// haben das Floating-Card-Muster und die max-w-Einengung ausdruecklich abgeschafft.
// Ich hatte beides am 07.08. wieder eingefuehrt — genau das ist Peter aufgefallen.
// Eine Route, die davon etwas selbst setzt, bricht genau eine dieser drei.

// Geprueft wird gegen die Routen, die Router.svelte tatsaechlich rendert — nicht gegen
// eine gepflegte Liste. Eine neue Route ist damit ab ihrem ersten Tag mit erfasst.
const ROUTER = join(srcRoot, 'lib/Router.svelte');

/** @returns {string[]} Pfade der Komponenten, die Router.svelte einbindet. */
function routenKomponenten() {
	const quelle = readFileSync(ROUTER, 'utf8');
	const treffer = [...quelle.matchAll(/from '\.\/([A-Za-z0-9/_-]+\.svelte)'/g)];
	return treffer.map((m) => join(srcRoot, 'lib', m[1])).filter((p) => existsSync(p));
}

// Nur Elemente auf der OBERSTEN Ebene: Was in einer Karte, einem Dialog oder einem
// Snippet steht, darf sehr wohl eine Flaeche haben — das ist ja der Sinn einer Karte.
// (Ein frueherer Anlauf las die erste `class=`-Zeile nach dem </script> und griff
// deshalb daneben: bei Mahnwesen traf er einen Toast, bei StatsDashboard ein <svg>.)
const OBERSTE_ELEMENTE = /^<[a-zA-Z][^>]*>/gm;
const VERBOTEN = /\bbg-|\bmax-w-/;

/** @param {string} pfad */
function verstoesse(pfad) {
	const quelle = readFileSync(pfad, 'utf8');
	const markup = quelle.slice(quelle.lastIndexOf('</script>') + 1);
	return [...markup.matchAll(OBERSTE_ELEMENTE)]
		.map((m) => m[0].replace(/\s+/g, ' '))
		.filter((tag) => VERBOTEN.test(tag));
}

describe('Seitengeruest', () => {
	it('laesst keine Route eine eigene Flaeche oder Breite setzen', () => {
		const gefunden = routenKomponenten().flatMap((pfad) =>
			verstoesse(pfad).map((tag) => `${pfad.slice(srcRoot.length + 1)}: ${tag.slice(0, 100)}`)
		);

		expect(
			gefunden,
			'Eine Route setzt auf oberster Ebene eine eigene Flaeche (bg-*) oder Breite (max-w-*).\n' +
				'Die Flaeche gehoert App.svelte, der Kopf PageShell.svelte. Inhalt laeuft\n' +
				'edge-to-edge ueber die volle Breite:\n  ' +
				gefunden.join('\n  ')
		).toEqual([]);
	});

	it('findet die Routen ueberhaupt (sonst prueft der Test oben ins Leere)', () => {
		// Ohne diese Zusicherung waere ein umbenannter Router ein still gruener Test.
		expect(routenKomponenten().length).toBeGreaterThan(10);
	});

	// Seiten tragen KEINEN eigenen Titel. a3e4184 (20.06.2026) hat "redundant page
	// headers and description subtitles" aus fuenf Arbeitsbereichen entfernt: Die
	// Seitenleiste sagt bereits, wo man ist, und ein Satz wie "Bedarf erfassen,
	// bestellen und Zulauf verbuchen" erklaert einem Sekretariat nichts, das die Seite
	// taeglich benutzt. Ich hatte beides am 07.08. auf allen 14 Seiten wieder eingebaut
	// — Peter hat es am 08.08. beanstandet. Diese Regel merkt es beim naechsten Mal.
	it('gibt keiner Route einen eigenen Seitentitel', () => {
		// Genau eine Ausnahme, und die ist keine: Die Statistik-Detailseite sagt, WELCHE
		// Liste man sieht (Renner oder Ladenhueter). Das kann die Navigation nicht.
		const AUSNAHMEN = ['lib/components/stats/StatistikDetailPage.svelte'];

		const gefunden = routenKomponenten()
			.filter((pfad) => /<h1\b/.test(readFileSync(pfad, 'utf8')))
			.map((pfad) => pfad.slice(srcRoot.length + 1))
			.filter((p) => !AUSNAHMEN.includes(p));

		expect(
			gefunden,
			'Diese Route rendert eine eigene <h1>. Der Seitenname steht in der\n' +
				'Seitenleiste; ein zweites Mal darueber ist redundant (siehe a3e4184):\n  ' +
				gefunden.join('\n  ')
		).toEqual([]);
	});

	// Blinder Fleck des Tests darueber: Er prueft die Routen-KOMPONENTEN. Die Huelle, in
	// die Router.svelte sie stellt, sah er nie — und genau dort stand fuer LMF-Aktionen
	// ein `p-6 max-w-6xl mx-auto`, also eine vierte Inhaltsbreite an der einzigen Stelle,
	// die niemand als Seite liest.
	it('stellt jede Route in dieselbe Huelle', () => {
		const quelle = readFileSync(ROUTER, 'utf8');
		const markup = quelle.slice(quelle.lastIndexOf('</script>') + 1);
		const gefunden = [...markup.matchAll(/<div class="[^"]*"/g)]
			.map((m) => m[0])
			.filter((tag) => /\bmax-w-|\bbg-|\bp-\d|\bpx-\d|\bpy-\d/.test(tag));

		expect(
			gefunden,
			'Router.svelte gibt einer Route eine eigene Breite, Flaeche oder Polsterung.\n' +
				'Die Huelle ist fuer alle Routen gleich; Breite und Kopf gehoeren in PageShell:\n  ' +
				gefunden.join('\n  ')
		).toEqual([]);
	});
});

// ── Form-Skala ───────────────────────────────────────────────────────────────
// styles/theme-mass.css ordnet jeder BAUTEILROLLE einen Radius zu: Menues 4 px,
// Chips 8, Karten 12, Navigationsflaechen 16, Dialoge 28, Buttons Pille. Wer eine
// weisse Flaeche aufzieht, baut eine Karte, ein Eingabefeld oder einen Dialog —
// also 12 px, 28 px oder rund. 16 px (`rounded-2xl`) gehoert dort NICHT hin.
//
// Am 07.08.2026 standen im Kartenkontext sieben Radien nebeneinander (2xl 23x,
// xl 14x, 3xl 13x, lg 7x, full 5x, t 1x, md 1x) — dieselbe Sache, verschiedene
// Ecken, je nachdem wer sie zuletzt angefasst hatte.
//
// Geprueft wird NUR `rounded-2xl` (16 px). Das ist die Navigationsflaechen-Rolle
// und auf einer Karte schlicht falsch. Ausdruecklich NICHT geprueft:
//   rounded-lg  — `--radius-lg` und `--radius-xl` sind BEIDE 12 px, die Fundstellen
//                 sind also bereits richtig, nur anders benannt.
//   rounded-md  — 8 px ist die Chip-Rolle und auf einem Badge korrekt.
// Ein Test, der auch die beiden meldet, faende 27 Dateien ohne einen einzigen
// sichtbaren Fehler — und wuerde nach der dritten Falschmeldung abgeschaltet.
const KARTENFLAECHE = /class="[^"]*\bbg-(?:white|surface-container-lowest)\b[^"]*"/g;
const FALSCHE_ROLLE = /\brounded-(?:2xl|4xl)\b/;

describe('Form-Skala', () => {
	it('gibt keiner weissen Flaeche den Radius der Navigationsflaechen', () => {
		const gefunden = sammleQuelldateien(srcRoot)
			.filter((f) => f.endsWith('.svelte'))
			.flatMap((f) =>
				[...readFileSync(f, 'utf8').matchAll(KARTENFLAECHE)]
					.map((m) => m[0])
					.filter((tag) => FALSCHE_ROLLE.test(tag))
					.map((tag) => `${f.slice(srcRoot.length + 1)}: ${tag.slice(0, 110)}`)
			);

		expect(
			gefunden,
			'Weisse Flaeche mit 16 px Radius. 16 px gehoert den Navigationsflaechen;\n' +
				'Karten und Eingabefelder sind 12 px (rounded-xl), Dialoge 28 px (rounded-3xl),\n' +
				'Menues stehen auf bg-surface-container mit 4 px (siehe SelectListe.svelte).\n' +
				'Die Zuordnung steht in styles/theme-mass.css:\n  ' +
				gefunden.join('\n  ')
		).toEqual([]);
	});
});
