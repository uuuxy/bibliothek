import { describe, it, expect } from 'vitest';
import { readFileSync, readdirSync, statSync } from 'node:fs';
import { join } from 'node:path';
import { coverSrc, coverKandidaten } from './coverSrc.js';

// Gleiche Herkunft für ALLE Bilder — das ist seit dem 06.08.2026 keine Stilfrage mehr,
// sondern die Voraussetzung der Content-Security-Policy: img-src kennt kein https: mehr
// (internal/middleware/security.go). Eine wiedereingeführte fremde Bildadresse würde im
// Browser stumm blockiert; die Kachel bliebe einfach leer. Genau die Sorte Fehler, die
// niemand meldet, weil es aussieht wie "das Buch hat halt kein Cover".

// Vitest läuft im Paketverzeichnis (frontend/). Das ist verlässlicher als eine
// Auflösung über import.meta.url, die hier nur "/src" ergab.
const WURZEL = join(process.cwd(), 'src') + '/';

/** @param {string} verzeichnis */
function alleQuelldateien(verzeichnis) {
	/** @type {string[]} */
	const treffer = [];
	for (const eintrag of readdirSync(verzeichnis)) {
		const pfad = join(verzeichnis, eintrag);
		if (statSync(pfad).isDirectory()) {
			treffer.push(...alleQuelldateien(pfad));
		} else if (pfad.endsWith('.svelte') || (pfad.endsWith('.js') && !pfad.endsWith('.test.js'))) {
			treffer.push(pfad);
		}
	}
	return treffer;
}

describe('Cover-Herkunft', () => {
	it('durchsucht überhaupt Dateien — sonst wäre das Gate wertlos grün', () => {
		expect(alleQuelldateien(WURZEL).length).toBeGreaterThan(50);
	});

	it('bindet nirgends eine fremde Bildadresse direkt ein', () => {
		// Die einzige Datei, die fremde Adressen NENNEN darf, ist coverSrc.js selbst —
		// dort werden sie gebaut und sofort durch den Proxy geschickt.
		const verstoesse = alleQuelldateien(WURZEL)
			.filter((p) => !p.endsWith('utils/coverSrc.js'))
			.flatMap((pfad) => {
				const inhalt = readFileSync(pfad, 'utf8');
				return inhalt
					.split('\n')
					.map((zeile, i) => ({ pfad, nr: i + 1, zeile }))
					.filter(
						({ zeile }) =>
							/https?:\/\/(covers\.openlibrary\.org|books\.google\.com|portal\.dnb\.de)/.test(
								zeile
							) &&
							!zeile.includes('coverSrc(') &&
							!zeile.includes('proxyCover(') &&
							!zeile.trimStart().startsWith('//') &&
							!zeile.trimStart().startsWith('*')
					);
			})
			.map(({ pfad, nr, zeile }) => `${pfad.replace(WURZEL, '')}:${nr}: ${zeile.trim()}`);

		expect(verstoesse, `Direkte Cover-Hotlinks gefunden:\n${verstoesse.join('\n')}`).toEqual([]);
	});

	it('liefert für externe Adressen ausschließlich Proxy-Pfade', () => {
		const proxied = coverSrc('https://covers.openlibrary.org/b/isbn/123-L.jpg', '123');
		expect(proxied.startsWith('/api/images/cover?')).toBe(true);
	});

	it('gibt lokale Pfade unverändert weiter', () => {
		expect(coverSrc('/uploads/covers/abc.webp', '123')).toBe('/uploads/covers/abc.webp');
	});

	it('macht aus der Kandidatenliste lauter Quellen gleicher Herkunft', () => {
		const kandidaten = coverKandidaten('https://covers.openlibrary.org/x.jpg', '978-3-16-148410-0');
		expect(kandidaten.length).toBeGreaterThan(0);
		for (const k of kandidaten) {
			expect(k.startsWith('/')).toBe(true);
		}
	});

	it('liefert ohne ISBN keine externe Quelle', () => {
		expect(coverSrc('https://covers.openlibrary.org/x.jpg', '')).toBe('');
		expect(coverKandidaten('https://covers.openlibrary.org/x.jpg', '')).toEqual([]);
	});
});
