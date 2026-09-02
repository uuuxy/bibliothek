// Welche Aktionen die Schüler-Ansichten anbieten, entscheidet das Recht — dasselbe,
// das der Server an der jeweiligen Route verlangt (api/routes_students*.go).
//
// Bis zum 24.08.2026 fragten StudentDirectory, StudentProfile und ihre Bauteile
// `role === 'admin'`; ein Schalter auf der Berechtigungsseite blieb dort wirkungslos,
// weil die Oberfläche ihn nie las. Die Zuordnung steht EINMAL hier, damit Profil und
// Verzeichnis nicht auseinanderlaufen (Ratsche: frontend-hygiene-rechte.test.js).
import { hatRecht } from './menu.js';

/**
 * @param {any} user  authStore.currentUser
 * @returns {{ einsehen: boolean, anlegen: boolean, bearbeiten: boolean, loeschen: boolean, endgueltigLoeschen: boolean, auskunft: boolean, foto: boolean, zusammenfuehren: boolean }}
 */
export function schuelerRechte(user) {
	return {
		// Kontoauszug, Ersatzforderung, Ausweisdruck (GET /api/print/…)
		einsehen: hatRecht(user, 'view_students'),
		// POST /api/schueler
		anlegen: hatRecht(user, 'create_students'),
		// Stammdaten, Abgangsjahr, Sperre, Schadensmeldung, Gebühren (PATCH /api/schueler/{id}, …)
		bearbeiten: hatRecht(user, 'edit_students'),
		// Papierkorb: DELETE /api/schueler/{id}, GET …/deleted, POST …/restore
		loeschen: hatRecht(user, 'delete_students'),
		// DELETE /api/schueler/deleted/{id} — sofortiges endgültiges Löschen aus dem
		// Papierkorb (Art.-17-Löschverlangen), nicht dasselbe Recht wie der Soft-Delete
		endgueltigLoeschen: hatRecht(user, 'manage_students_admin'),
		// GET /api/schueler/{id}/dsgvo-auskunft
		auskunft: hatRecht(user, 'manage_students_admin'),
		// POST /api/schueler/{id}/photo
		foto: hatRecht(user, 'upload_photos'),
		// POST /api/schueler/{id}/zusammenfuehren — zwei Datensätze, ein Mensch (Umbenennung
		// ohne Schüler-ID, Dublette). Unumkehrbar, deshalb dasselbe Recht wie das Purge.
		zusammenfuehren: hatRecht(user, 'manage_students_admin')
	};
}
