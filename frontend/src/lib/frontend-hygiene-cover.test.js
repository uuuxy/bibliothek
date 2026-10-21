import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import {
	srcRoot,
	sammleQuelldateien,
	relPfad,
	vergleicheMitBestand,
	ohneKommentare
} from './hygiene-quellen.js';

// Buchcover kommen aus components/ui/BuchCover.svelte: ein Zuschnitt, eine Ausweich-Logik
// über den eigenen Proxy, ein Platzhalter. Ein eigenes Cover erkennt die Ratsche an den
// Helfern aus utils/coverSrc.js und an einem <img>, dessen Quelle ein Cover nennt.
//
// Warum eine Komponente und kein Snippet: Ein Cover trägt Zustand JE VORKOMMEN (Index
// in der Kandidatenliste, „alle Quellen gescheitert"). Snippets sind Markup-Vorlagen
// ohne eigenen Zustand.
const EIGENES_COVER = /coverKandidaten|coverSrc\(|<img\b[^>]*\bsrc=\{[^}]*cover/i;

// Zwei Stellen mit eigenem Aufbau um das Bild: Die Maske „Buch bearbeiten" legt den Hinweis
// zum Hochladen darüber, die Auswahlkarte des Klassensatz-Dialogs die Markierung der Wahl.
// Beide schneiden schon 2:3 und ziehen nach, wenn ihr Aufbau ohnehin geändert wird.
const BESTAND = [
	'src/inventur/lib/components/admin/BuchCoverUpload.svelte',
	'src/inventur/lib/components/admin/KlassensatzTitelRaster.svelte'
];

// Die beiden Bauteile selbst greifen zu Recht auf die Helfer zu: BuchCover zeichnet das
// Bild, CoverVorschau die Großansicht. Sie sind die Quelle, nicht ein Verstoß.
const QUELLEN = [
	'src/lib/components/ui/BuchCover.svelte',
	'src/lib/components/ui/CoverVorschau.svelte',
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
		const code = ohneKommentare(datei);

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
