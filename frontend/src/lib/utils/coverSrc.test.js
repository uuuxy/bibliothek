import { describe, it, expect } from 'vitest';
import { coverSrc } from './coverSrc.js';

/**
 * Der Fehler, den diese Tests festhalten: Lokal abgelegte Cover wurden durch den
 * Cover-Proxy geschickt, dessen Host-Allowlist nur fremde Server kennt. Er antwortete
 * mit einem transparenten 1×1-GIF — die Großansicht meldete "Kein Coverbild hinterlegt",
 * während das Miniaturbild derselben Zeile das Cover zeigte.
 *
 * Da der Cover-Sync JEDES gefundene Cover auf einen lokalen Pfad migriert, war das nicht
 * der Sonderfall, sondern der Regelfall.
 */
describe('coverSrc', () => {
	it('liefert einen lokalen Pfad unverändert aus — der Proxy kennt ihn nicht', () => {
		expect(coverSrc('/uploads/covers/9783123456789.webp', '9783123456789')).toBe(
			'/uploads/covers/9783123456789.webp'
		);
	});

	it('nimmt für den lokalen Pfad auch ohne ISBN den direkten Weg', () => {
		expect(coverSrc('/uploads/covers/abc.webp')).toBe('/uploads/covers/abc.webp');
	});

	it('schickt eine externe URL über den Proxy', () => {
		expect(
			coverSrc('https://portal.dnb.de/opac/mvb/cover?isbn=9783123456789', '9783123456789')
		).toBe(
			'/api/images/cover?isbn=9783123456789&url=https%3A%2F%2Fportal.dnb.de%2Fopac%2Fmvb%2Fcover%3Fisbn%3D9783123456789'
		);
	});

	it('meldet ohne Cover-URL nichts an — der Aufrufer zeigt den Platzhalter', () => {
		expect(coverSrc('', '9783123456789')).toBe('');
		expect(coverSrc(undefined, '9783123456789')).toBe('');
		expect(coverSrc('   ', '9783123456789')).toBe('');
	});

	it('verzichtet bei externer URL ohne ISBN auf den Proxy', () => {
		// Ohne ISBN fehlt dem Proxy der Cache-Schlüssel; er lieferte das 1×1-GIF und damit
		// ein leeres Kästchen statt des Platzhalters.
		expect(coverSrc('https://covers.openlibrary.org/b/id/1-L.jpg')).toBe('');
		expect(coverSrc('https://covers.openlibrary.org/b/id/1-L.jpg', '  ')).toBe('');
	});
});
