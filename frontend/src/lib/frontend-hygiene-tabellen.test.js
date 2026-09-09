import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { srcRoot, sammleQuelldateien, relPfad, ohneKommentare } from './hygiene-quellen.js';

// Ratsche: Datentabellen kommen aus ui/Tabelle.svelte, und ihre Zellen sagen nichts
// über ihr Aussehen.
//
// Anlass (08.09.2026): 24 Tabellen mit 24 Rezepten — das Aussehen hing mal am <thead>,
// mal an der Kopfzeile, mal am <th>; sechs Innenabstände, zwei Schriftgrößen, drei
// Hover-Farben, zwei Farben für die gewählte Zeile. Seither steht das Rezept EINMAL im
// <style> des Bauteils; die Zellen behalten nur Layout (Breite, Ausrichtung, Umbruch,
// Ziffern) und im Inhalt die Betonung (font-*).
//
// Zwei bewusste Ausnahmen (je Kommentar am Ort): die Druckquittung (Papier, kein M3)
// und die Bildschirmleser-Tabelle des Trend-Diagramms (sr-only).
//
// Rot bewiesen am 08.09.2026 gegen den Bestand vor der Umstellung (24 Dateien).
const AUSNAHMEN = new Set([
	'src/lib/StudentPrintReceipt.svelte',
	'src/lib/components/stats/StatsTrendChart.svelte'
]);
const ROHE_TABELLE = /<table\b/g;
// Aussehen in Zellen, Zeilen und Köpfen: Fläche, Linie, Farbe, Größe, Abstand, Hover.
const OPTIK =
	/\b(bg-|border(-[a-z]|\b)|divide-|text-(slate|on-surface|xs|sm|base|lg)|hover:|uppercase|tracking-|p[xy]?-[0-9.]+|transition)/;
const ZELLE = /<(thead|tbody|tr|th|td)\b[^>]*\bclass="([^"]*)"/g;

describe('Tabellen kommen aus ui/', () => {
	it('kennt kein rohes <table> ausserhalb von ui/Tabelle.svelte', () => {
		const treffer = [];
		for (const datei of sammleQuelldateien(srcRoot)) {
			if (!datei.endsWith('.svelte')) continue;
			const rel = relPfad(datei);
			if (rel === 'src/lib/components/ui/Tabelle.svelte' || AUSNAHMEN.has(rel)) continue;
			const n = (ohneKommentare(readFileSync(datei, 'utf8')).match(ROHE_TABELLE) || []).length;
			if (n) treffer.push(`${rel} (${n})`);
		}
		expect(treffer, 'Rohes <table> — bitte <Tabelle> aus ui/Tabelle.svelte nehmen').toEqual([]);
	});

	it('gibt Zellen, Zeilen und Köpfen keine Optik (nur Breite, Ausrichtung, Umbruch, Ziffern)', () => {
		const treffer = [];
		for (const datei of sammleQuelldateien(srcRoot)) {
			if (!datei.endsWith('.svelte')) continue;
			const rel = relPfad(datei);
			if (AUSNAHMEN.has(rel)) continue;
			const quelle = ohneKommentare(readFileSync(datei, 'utf8'));
			if (!quelle.includes('<Tabelle') && !quelle.includes('<table')) continue;
			for (const m of quelle.matchAll(ZELLE)) {
				const boese = m[2].split(/\s+/).filter((k) => OPTIK.test(k));
				if (boese.length) treffer.push(`${rel}: <${m[1]} …${boese.join(' ')}>`);
			}
		}
		expect(treffer, 'Optik an Tabellenzellen — das Rezept steht in ui/Tabelle.svelte').toEqual([]);
	});

	// Barrierefreiheit (09.09.2026): Jede Tabelle sagt, was in ihr steht — als unsichtbare
	// <caption>, die ui/Tabelle aus `beschriftung` setzt. Ohne sie hört ein Screenreader
	// nur „Tabelle, 6 Spalten, 1482 Zeilen". Rot bewiesen gegen den Bestand (26 Tabellen).
	it('gibt jeder <Tabelle> eine beschriftung', () => {
		const treffer = [];
		for (const datei of sammleQuelldateien(srcRoot)) {
			if (!datei.endsWith('.svelte')) continue;
			const rel = relPfad(datei);
			if (rel === 'src/lib/components/ui/Tabelle.svelte') continue;
			const quelle = ohneKommentare(readFileSync(datei, 'utf8'));
			for (const m of quelle.matchAll(/<Tabelle\b([^>]*)>/g)) {
				if (!/\bbeschriftung=/.test(m[1])) treffer.push(`${rel}: <Tabelle${m[1]}>`);
			}
		}
		expect(treffer, '<Tabelle> ohne beschriftung — der Screenreader braucht den Titel').toEqual([]);
	});
});
