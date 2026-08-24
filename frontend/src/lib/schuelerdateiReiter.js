// Die Reiter der Schülerdatei — welche es gibt, hängt an den Rechten des Lesers:
// Papierkorb an delete_students, Schuljahreswechsel (LUSD, Versetzung; seit 24.08.2026
// hier statt unter Einstellungen → Datenverwaltung) an import_students bzw.
// manage_students_admin. Reine Funktion, damit StudentDirectory schlank bleibt.

/**
 * @param {{ loeschen: boolean, schuljahreswechsel: boolean }} rechte
 * @returns {{ id: string, label: string }[]}
 */
export function schuelerdateiReiter(rechte) {
	const liste = [
		{ id: 'active', label: 'Aktive Schüler' },
		{ id: 'graduates', label: 'Abgänger / Archiv' }
	];
	if (rechte.loeschen) liste.push({ id: 'deleted', label: 'Papierkorb' });
	if (rechte.schuljahreswechsel) liste.push({ id: 'schuljahr', label: 'Schuljahreswechsel' });
	return liste;
}
