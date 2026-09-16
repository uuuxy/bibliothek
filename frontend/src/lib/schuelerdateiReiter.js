// Die Reiter der Leserdatei — welche es gibt, hängt an den Rechten des Bedieners:
// Papierkorb an delete_students. Reine Funktion, damit StudentDirectory schlank bleibt.

/**
 * @param {{ loeschen: boolean }} rechte
 * @returns {{ id: string, label: string }[]}
 */
export function schuelerdateiReiter(rechte) {
	const liste = [
		// „Aktive Leser": Der Reiter zeigt seit dem 16.09.2026 Schüler UND Kollegium.
		{ id: 'active', label: 'Aktive Leser' },
		// „Ehemalige", nicht „Abgänger": Abgänger sind die Abschlussklassen, die noch da sind
		// (eigene Ansicht /abgaenger); hier stehen die, die laut LUSD schon weg sind.
		{ id: 'graduates', label: 'Ehemalige / Archiv' }
	];
	if (rechte.loeschen) liste.push({ id: 'deleted', label: 'Papierkorb' });
	return liste;
}
