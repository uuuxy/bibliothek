import { describe, it, expect } from 'vitest';
import { readFileSync, existsSync } from 'node:fs';
import { join } from 'node:path';
import { srcRoot } from './hygiene-quellen.js';

// Fuenfte Struktur-Invariante: Eine Route bringt kein eigenes Seitengeruest mit.
//
// Am 07.08.2026 tat das jede zweite: VIER Routen malten mit `bg-slate-50` eine zweite
// Flaeche ueber die Leinwand der App-Huelle, VIER verschiedene `max-w-*` legten vier
// Inhaltsbreiten fest. Auf dem Bildschirm sah dadurch jede Seite anders aus — mal
// getoente Arbeitsflaeche, mal weisse; mal 896 px breit, mal 1152, mal randlos.
//
// Die Zustaendigkeiten seitdem:
//   Flaeche  -> App.svelte (`bg-surface`, die getoente Leinwand)
//   Karte    -> Sheet.svelte (`surface-container-lowest`, das Weiss darauf)
//   Breite   -> PageShell.svelte (`voll` oder `inhalt`)
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
				'Die Flaeche gehoert App.svelte, das Weiss darauf Sheet.svelte, die Breite\n' +
				'PageShell.svelte (breite="voll" | "inhalt"):\n  ' +
				gefunden.join('\n  ')
		).toEqual([]);
	});

	it('findet die Routen ueberhaupt (sonst prueft der Test oben ins Leere)', () => {
		// Ohne diese Zusicherung waere ein umbenannter Router ein still gruener Test.
		expect(routenKomponenten().length).toBeGreaterThan(10);
	});
});
