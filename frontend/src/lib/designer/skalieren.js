/**
 * Die neuen Maße eines Elements beim Ziehen an einer Ecke, in mm: Die gegenüberliegende Ecke bleibt
 * stehen. Mindestens 5 mm breit und 3 mm hoch; ein proportionales Element (Foto, Logo, Strichcode)
 * behält sein Seitenverhältnis, seine Höhe folgt der Breite.
 * @param {'nw'|'ne'|'sw'|'se'} ecke
 * @param {{ x: number, y: number, width: number, height: number, proportional?: boolean }} start
 * @param {number} dx  Weg des Zeigers nach rechts in mm
 * @param {number} dy  Weg des Zeigers nach unten in mm
 * @returns {{ x: number, y: number, width: number, height: number }}
 */
export function skaliere(ecke, start, dx, dy) {
	const { x, y, width: breite, height: hoehe } = start;
	const rechts = ecke === 'ne' || ecke === 'se';
	const unten = ecke === 'sw' || ecke === 'se';
	const width = Math.max(5, rechts ? breite + dx : breite - dx);
	const height = start.proportional
		? width / (breite / hoehe)
		: Math.max(3, unten ? hoehe + dy : hoehe - dy);
	return {
		x: rechts ? x : Math.max(0, x + (breite - width)),
		y: unten ? y : Math.max(0, y + (hoehe - height)),
		width,
		height
	};
}
