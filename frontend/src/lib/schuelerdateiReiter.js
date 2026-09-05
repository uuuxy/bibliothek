// Die Reiter der Schülerdatei — welche es gibt, hängt an den Rechten des Lesers:
// Papierkorb an delete_students. Reine Funktion, damit StudentDirectory schlank bleibt.

/**
 * @param {{ loeschen: boolean }} rechte
 * @returns {{ id: string, label: string }[]}
 */
export function schuelerdateiReiter(rechte) {
	const liste = [
		{ id: 'active', label: 'Aktive Schüler' },
		// „Ehemalige", nicht „Abgänger": Abgänger sind die Abschlussklassen, die noch da sind
		// (eigene Ansicht /abgaenger); hier stehen die, die laut LUSD schon weg sind.
		{ id: 'graduates', label: 'Ehemalige / Archiv' }
	];
	if (rechte.loeschen) liste.push({ id: 'deleted', label: 'Papierkorb' });
	return liste;
}
