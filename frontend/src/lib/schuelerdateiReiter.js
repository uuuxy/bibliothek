// Die Reiter der Schülerdatei — welche es gibt, hängt an den Rechten des Lesers:
// Papierkorb an delete_students. Reine Funktion, damit StudentDirectory schlank bleibt.

/**
 * @param {{ loeschen: boolean }} rechte
 * @returns {{ id: string, label: string }[]}
 */
export function schuelerdateiReiter(rechte) {
	const liste = [
		{ id: 'active', label: 'Aktive Schüler' },
		{ id: 'graduates', label: 'Abgänger / Archiv' }
	];
	if (rechte.loeschen) liste.push({ id: 'deleted', label: 'Papierkorb' });
	return liste;
}
