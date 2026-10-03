// Die Rollen eines Kontos mit ihrem Namen für Menschen: eine Liste für das Auswahlfeld im
// Bearbeiten-Dialog und für die Spalte „Rolle" der Benutzerliste, damit beide dasselbe Wort
// zeigen.
//
// Kollegium steht oben, weil es keine Rolle ist, sondern der Grundzustand: Wer sich übers
// Portal selbst anmeldet und freigeschaltet wird, ist Kollegium. Darunter die Erhebungen, die
// der Administrator vornimmt, aufsteigend. Stünde Kollegium zwischen Helfer und Mitarbeiter,
// läse sich die Liste als Rangfolge mit dem Kollegium als Stufe darin.
/** @type {{ value: string, name: string, zusatz?: string }[]} */
const ROLLEN = [
	{ value: 'kollegium', name: 'Kollegium', zusatz: 'keine Rolle, nur Portal' },
	{ value: 'helfer', name: 'Helfer' },
	{ value: 'mitarbeiter', name: 'Mitarbeiter' },
	{ value: 'leitung', name: 'Leitung' },
	{ value: 'admin', name: 'Administrator' }
];

/** Die Einträge für das Auswahlfeld. */
export const ROLLEN_AUSWAHL = ROLLEN.map((r) => ({
	value: r.value,
	label: r.zusatz ? `${r.name} (${r.zusatz})` : r.name
}));

/**
 * Name einer Rolle für die Anzeige. Eine Rolle, die diese Liste nicht kennt, steht mit ihrem
 * Schlüssel da statt als leeres Feld.
 * @param {string} rolle
 */
export const rollenName = (rolle) => ROLLEN.find((r) => r.value === rolle)?.name ?? rolle;
