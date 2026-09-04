import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { srcRoot, sammleQuelldateien, relPfad, vergleicheMitBestand } from './hygiene-quellen.js';

// Buchcover kommen aus components/ui/BuchCover.svelte — dieselbe Invariante wie bei
// Suchfeldern, Reitern und Symbolen.
//
// Anlass (04.09.2026): Peter fragte, warum der Bestellbedarf keine Cover zeigt. Beim
// Nachsehen fanden sich FÜNF Größen im Haus (w-7, w-8, w-10, w-12, w-16) und VIER
// Kopien derselben Ausweich-Logik — BuchKarte, KlassenBuchKachel, BookTableZeile und
// IsbnLookupDialog liefen jede für sich durch dieselbe Kandidatenliste, mit eigener
// Fehlerbehandlung und eigenem Platzhalter. Genau die Ausgangslage der Suchfelder,
// die in zehn Fassungen mit sieben Maßen endeten.
//
// Warum eine Komponente und kein Snippet: Ein Cover trägt Zustand JE VORKOMMEN (Index
// in der Kandidatenliste, „alle Quellen gescheitert"). Snippets sind Markup-Vorlagen
// ohne eigenen Zustand.
//
// Der Bestand ist bewusst NICHT in einem Rutsch umgestellt: Es sind siebzehn Dateien,
// darunter täglich benutzte Bildschirme, und ein Massen-Refactoring an einem Tag hat
// in diesem Projekt schon zweimal eine Regression erzeugt. Sie stehen hier eingefroren
// und werden bei ihrem nächsten fachlichen Anfassen nachgezogen — so wie es
// OrderRecommendations am 04.09. als erster Verwender vorgemacht hat.
const EIGENES_COVER = /coverKandidaten|coverSrc\(/;

const BESTAND = [
	'src/lib/BorrowedBooksList.svelte',
	'src/lib/OpacSearch.svelte',
	'src/lib/StatsDashboard.svelte',
	'src/lib/StudentPrintReceipt.svelte',
	'src/lib/components/bestellungen/BestellDetailPositionen.svelte',
	'src/lib/components/bestellungen/OrderCart.svelte',
	'src/lib/components/bestellungen/OrderSearch.svelte',
	'src/lib/components/bestellungen/WareneingangTable.svelte',
	'src/lib/components/portal/PortalTrefferkarte.svelte',
	'src/lib/components/stats/StatistikDetailPage.svelte',
	'src/inventur/lib/components/BuchKarte.svelte',
	'src/inventur/lib/components/admin/BookTableZeile.svelte',
	'src/inventur/lib/components/admin/ClassAssignmentSummary.svelte',
	'src/inventur/lib/components/admin/KlassenBuchKachel.svelte',
	'src/lib/useBookAkte.svelte.js',
	'src/lib/monitor/FolieBeliebt.svelte',
	'src/lib/monitor/FolieBuchDesMonats.svelte',
	'src/lib/monitor/FolieNeuEingetroffen.svelte'
];

// Die beiden Bauteile selbst greifen zu Recht auf die Helfer zu: BuchCover zeichnet das
// Bild, CoverPeek die Großansicht. Sie sind die Quelle, nicht ein Verstoß.
const QUELLEN = [
	'src/lib/components/ui/BuchCover.svelte',
	'src/lib/components/ui/CoverPeek.svelte',
	'src/lib/utils/coverSrc.js'
];

describe('Cover-Hygiene', () => {
	it('baut keine neuen Cover von Hand (sie kommen aus ui/BuchCover.svelte)', () => {
		const betroffen = sammleQuelldateien(srcRoot)
			.filter((f) => EIGENES_COVER.test(readFileSync(f, 'utf8')))
			.map(relPfad)
			.filter((f) => !QUELLEN.includes(f))
			.sort();

		const { neu, inzwischenSauber } = vergleicheMitBestand(betroffen, BESTAND);

		expect(
			neu,
			'Neue Datei mit eigenem Cover-Markup. Es gab davon schon fünf Größen und vier ' +
				'Kopien der Ausweich-Logik — bitte ui/BuchCover.svelte benutzen.'
		).toEqual([]);

		expect(
			inzwischenSauber,
			'Der Bestand ist überholt — bitte aus BESTAND austragen. Eine Liste, die ' +
				'Erledigtes weiterführt, verliert ihre Aussage.'
		).toEqual([]);
	});

	it('das Bauteil lädt verzögert und prüft die Bildgröße', () => {
		const datei = readFileSync(`${srcRoot}/lib/components/ui/BuchCover.svelte`, 'utf8');

		// OHNE KOMMENTARE geprüft. Beim Rot-Beweis am 04.09.2026 blieb dieser Test grün,
		// obwohl `loading="lazy"` aus dem <img> entfernt war — der String stand weiterhin
		// im Kopfkommentar, der ihn erklärt. Ein Detektor, der die Begründung für die Sache
		// hält, meldet ewig „alles gut" (Bugklasse „Lügende Ratsche", docs/sweeps.md).
		const code = datei
			.replace(/<!--[\s\S]*?-->/g, '')
			.replace(/\/\*[\s\S]*?\*\//g, '')
			.replace(/^\s*\/\/.*$/gm, '');

		// loading="lazy" ist der Grund, warum das Cover überhaupt in die Zeile zurückdarf:
		// Der Bestellbedarf hat auf dem Zielsystem 247 Zeilen, ohne lazy wären das 247
		// Anfragen beim Laden. Genau diese Last war das Argument dagegen.
		expect(code, 'BuchCover ohne loading="lazy" holt jede Zeile sofort').toContain(
			'loading="lazy"'
		);

		// Der Cover-Proxy antwortet auf nicht bediente Adressen mit einem 1×1-GIF und
		// Status 200 — onerror feuert dafür NIE. Ohne naturalWidth-Prüfung stünde ein
		// leeres Kästchen statt des Platzhalters.
		expect(
			code,
			'BuchCover ohne naturalWidth-Prüfung zeigt das 1×1-GIF des Proxys als leeres Bild'
		).toContain('naturalWidth');
	});
});
