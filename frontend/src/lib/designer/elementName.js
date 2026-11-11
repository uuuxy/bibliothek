// Der Name eines Ausweis-Elements, wie ihn die Oberfläche nennt: im Kopf des
// Eigenschaften-Felds und in der Liste der Elemente. Die Kennung (el.id) ist ein interner
// Schlüssel („header", „el-101") und sagt der Verwaltung nichts.

const NAMEN = /** @type {Record<string, string>} */ ({
	header: 'Schulname',
	address: 'Adresse',
	logo: 'Logo',
	photo: 'Passbild',
	name: 'Name',
	validity: 'Gültigkeit',
	dokumenttyp: 'Ausweisart',
	barcode: 'Strichcode',
	image: 'Bild',
	box: 'Fläche'
});

/**
 * Ein freier Text heißt nach seinem Inhalt, damit zwei Texte unterscheidbar sind.
 * @param {{ type: string, content?: string }} el
 */
export function elementName(el) {
	if (el.type === 'text') return (el.content ?? '').trim() || 'Text';
	return NAMEN[el.type] ?? 'Element';
}
