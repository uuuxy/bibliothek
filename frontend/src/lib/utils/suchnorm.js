/**
 * Die eine Normalform für Namensvergleiche, Zwilling der SQL-Funktion suchnorm und von suchnorm in
 * internal/repository/suchnorm.go: Kleinschreibung, Akzente weg, die deutschen Ersatzschreibungen
 * auf einen Nenner (ss→s, ue→u, oe→o, ae→a). So findet „ozturk" auch „Öztürk" und „Mueller" auch
 * „Müller", im Browser wie am Server. suchnorm.faelle.json ist aus der Datenbank erzeugt; der Test
 * hier und der in Go lesen sie, damit alle drei dasselbe rechnen.
 */

// Wo unaccent anders rechnet als die Zerlegung in Grundbuchstabe und Akzent: dieselben Paare wie
// unaccentTabelle in suchnorm.go, aus ihr übernommen.
const AUSNAHMEN = new Map([
	['×', '*'], // U+00D7
	['ß', 'ss'], // U+00DF
	['æ', 'ae'], // U+00E6
	['ð', 'd'], // U+00F0
	['÷', '/'], // U+00F7
	['ø', 'o'], // U+00F8
	['þ', 'th'], // U+00FE
	['đ', 'd'], // U+0111
	['ħ', 'h'], // U+0127
	['ı', 'i'], // U+0131
	['ĳ', 'ij'], // U+0133
	['ĸ', 'q'], // U+0138
	['ŀ', 'l'], // U+0140
	['ł', 'l'], // U+0142
	['ŉ', "'n"], // U+0149
	['ŋ', 'n'], // U+014B
	['œ', 'oe'], // U+0153
	['ŧ', 't'], // U+0167
	['ſ', 's'], // U+017F
	['ƀ', 'b'], // U+0180
	['ƃ', 'b'], // U+0183
	['ƈ', 'c'], // U+0188
	['ƌ', 'd'], // U+018C
	['ƒ', 'f'], // U+0192
	['ƕ', 'hv'], // U+0195
	['ƙ', 'k'], // U+0199
	['ƚ', 'l'], // U+019A
	['ƞ', 'n'], // U+019E
	['ƣ', 'oi'], // U+01A3
	['ƥ', 'p'], // U+01A5
	['ƫ', 't'], // U+01AB
	['ƭ', 't'], // U+01AD
	['ƴ', 'y'], // U+01B4
	['ƶ', 'z'], // U+01B6
	['ǆ', 'dz'], // U+01C6
	['ǉ', 'lj'], // U+01C9
	['ǌ', 'nj'], // U+01CC
	['ǣ', 'ǣ'], // U+01E3
	['ǥ', 'g'], // U+01E5
	['ǯ', 'ǯ'], // U+01EF
	['ǳ', 'dz'], // U+01F3
	['ǽ', 'ǽ'], // U+01FD
	['ǿ', 'ǿ'], // U+01FF
	['ȡ', 'd'], // U+0221
	['ȥ', 'z'], // U+0225
	['ȴ', 'l'], // U+0234
	['ȵ', 'n'], // U+0235
	['ȶ', 't'], // U+0236
	['ȷ', 'j'], // U+0237
	['ȸ', 'db'], // U+0238
	['ȹ', 'qp'], // U+0239
	['ȼ', 'c'], // U+023C
	['ȿ', 's'], // U+023F
	['ɀ', 'z'], // U+0240
	['ɇ', 'e'], // U+0247
	['ɉ', 'j'], // U+0249
	['ɍ', 'r'], // U+024D
	['ɏ', 'y'], // U+024F
	['ɓ', 'b'], // U+0253
	['ɖ', 'd'], // U+0256
	['ɗ', 'd'], // U+0257
	['ɛ', 'e'], // U+025B
	['ɠ', 'g'], // U+0260
	['ɨ', 'i'], // U+0268
	['ɲ', 'n'], // U+0272
	['ʀ', 'R'], // U+0280
	['ʈ', 't'], // U+0288
	['ʉ', 'u'], // U+0289
	['ʋ', 'v'], // U+028B
	['ẚ', 'a'], // U+1E9A
	['ẛ', 'ẛ'], // U+1E9B
	['ẜ', 's'], // U+1E9C
	['ẝ', 's'], // U+1E9D
	['ỻ', 'll'], // U+1EFB
	['ỽ', 'v'], // U+1EFD
	['ỿ', 'y'], // U+1EFF
	['ⱥ', 'a'], // U+2C65
	['ⱦ', 't'] // U+2C66
]);

/** @param {unknown} text */
export function suchnorm(text) {
	let ergebnis = '';
	for (const zeichen of String(text ?? '').toLowerCase()) {
		// ASCII hat weder Akzent noch Ausnahme; das spart das Zerlegen bei fast jedem Namen.
		if (zeichen < '\u0080') {
			ergebnis += zeichen;
			continue;
		}
		ergebnis +=
			AUSNAHMEN.get(zeichen) ??
			zeichen
				.normalize('NFD')
				.replace(/\p{Mn}/gu, '')
				.normalize('NFC');
	}
	return ergebnis
		.replaceAll('ss', 's')
		.replaceAll('ue', 'u')
		.replaceAll('oe', 'o')
		.replaceAll('ae', 'a');
}

// Bindestrich, Apostroph und Leerzeichen zählen beim Namen nicht, wie am Server (sqlNameKompakt in
// leser_suche.go): „alkahil" findet „Al-Kahil", „annalena" findet „Anna-Lena". Die Datenbank macht
// typografische Striche und Apostrophe vorher zu „-" und „'"; hier stehen sie deshalb mit drin.
const NAMEN_TRENNER = /[\s\-‐‑–—'’ʼ`´]/gu;

/** @param {unknown} text */
function kompakt(text) {
	return suchnorm(text).replace(NAMEN_TRENNER, '');
}

/**
 * Trifft jedes Wort der Suche eines der Felder? Dieselbe Regel wie bei der Suche nach Lesern an
 * der Theke: Akzente, Groß und klein, Bindestrich, Apostroph und Leerzeichen zählen nicht, die
 * Reihenfolge der Wörter auch nicht („Müller Anna" findet „Anna Müller").
 * @param {string} suche
 * @param {...unknown} felder
 */
export function passtZurSuche(suche, ...felder) {
	const woerter = String(suche ?? '')
		.split(/\s+/)
		.map(kompakt)
		.filter(Boolean);
	if (woerter.length === 0) return true;
	const kompakteFelder = felder.map(kompakt);
	return woerter.every((wort) => kompakteFelder.some((feld) => feld.includes(wort)));
}
