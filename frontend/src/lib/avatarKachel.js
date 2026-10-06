// Initialen-Kachel als Passbild-Ersatz (StudentProfileCard). Eine leere graue Box liest
// sich wie „kaputt"; fehlt das Foto, stehen dort die Initialen des Lesers.

/** @param {{ vorname?: string, nachname?: string }} p */
export const initialen = (p) =>
	((p.vorname?.[0] ?? '') + (p.nachname?.[0] ?? '')).toUpperCase() || '?';
