// Initialen-Kachel als Passbild-Ersatz (StudentProfileCard). Eine leere graue Box liest
// sich wie „kaputt"; fehlt das Foto, steht — wie bei Apple Kontakte / Google — eine
// Initialen-Kachel auf farbigem Verlauf. Die Farbe wird deterministisch aus dem Namen
// abgeleitet, damit derselbe Schüler immer dieselbe Kachel bekommt.
const VERLAEUFE = [
	'from-blue-500 to-indigo-600',
	'from-emerald-500 to-teal-600',
	'from-fuchsia-500 to-purple-600',
	'from-amber-500 to-orange-600',
	'from-rose-500 to-pink-600',
	'from-sky-500 to-cyan-600',
	'from-violet-500 to-purple-600',
	'from-lime-500 to-emerald-600'
];

/** @param {{ vorname?: string, nachname?: string }} p */
export const initialen = (p) =>
	((p.vorname?.[0] ?? '') + (p.nachname?.[0] ?? '')).toUpperCase() || '?';

/** @param {{ vorname?: string, nachname?: string }} p */
export function avatarVerlauf(p) {
	const key = `${p.vorname ?? ''} ${p.nachname ?? ''}`;
	let h = 0;
	for (let i = 0; i < key.length; i++) h = (h * 31 + key.charCodeAt(i)) >>> 0;
	return VERLAEUFE[h % VERLAEUFE.length];
}
