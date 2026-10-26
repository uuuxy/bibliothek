import { test, expect } from '@playwright/test';
import { uiLogin, apiPost, seedSQL, uniqueSuffix, gehZu } from './helpers.js';

// RBAC-Negativpfad: Wir testen sonst nur den voll autorisierten Admin.
// Hier der Beweis, dass die Kette Server-403 → UI-Ausblendung für
// Nicht-Admins hält und nichts leakt. Die Benutzer werden idempotent
// geseedet (Mock-IMAP akzeptiert beim Login jedes Passwort).

const MITARBEITER_EMAIL = 'e2e-mitarbeiter@test.local';
const LEHRER_EMAIL = 'e2e-lehrer@test.local';

function seedUsers() {
	seedSQL(`
        INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv)
        VALUES ('E2E', 'Mitarbeiter', '${MITARBEITER_EMAIL}', 'mitarbeiter', true),
               ('E2E', 'Lehrer', '${LEHRER_EMAIL}', 'kollegium', true)
        ON CONFLICT DO NOTHING;
    `);
}

test('Mitarbeiter: Admin-Endpoints (manage_settings/manage_students_admin) liefern 403, Admin-UI bleibt verborgen', async ({
	page
}) => {
	seedUsers();
	await uiLogin(page, MITARBEITER_EMAIL);

	// Mitarbeiter dürfen Schüler anlegen (create_students) — als Testobjekt
	const suffix = uniqueSuffix();
	const created = await apiPost(page, '/api/schueler', {
		geburtsdatum: '2012-06-15', // Pflicht seit 21.08.2026: Schlüssel für den LUSD-Abgleich
		vorname: 'E2E',
		nachname: `Rbac-${suffix}`,
		klasse: '6A',
		barcode_id: `S-${suffix}`
	});
	expect(created.ok(), `Schüler-Seeding als Mitarbeiter: ${created.status()}`).toBeTruthy();
	const { id: studentId } = await created.json();

	// DSGVO-Auskunft bündelt ALLE Daten eines Kindes → nur manage_students_admin
	const auskunft = await page.request.get(`/api/schueler/${studentId}/dsgvo-auskunft`);
	expect(auskunft.status(), 'DSGVO-Auskunft für Mitarbeiter').toBe(403);
	expect(await auskunft.text()).not.toContain(`Rbac-${suffix}`);

	// Zusammenführen ist seit 03.09.2026 ein eigenes Recht (merge_students) — ab Werk nur Admin;
	// die Kandidatensuche liefert Namen über ALLE Schüler und muss darum hart zu sein.
	const kandidaten = await page.request.get(
		`/api/schueler/${studentId}/zusammenfuehren-kandidaten?q=Rbac`
	);
	expect(kandidaten.status(), 'Zusammenführen-Kandidaten für Mitarbeiter').toBe(403);
	expect(await kandidaten.text()).not.toContain(`Rbac-${suffix}`);

	// Backup-Status verlangt manage_settings — Mitarbeiter haben es ab Werk nicht
	const backup = await page.request.get('/api/admin/system/backup-status');
	expect(backup.status(), 'Backup-Status für Mitarbeiter').toBe(403);

	// …und deshalb darf das Backup-Alert-Badge im UI nicht auftauchen
	await expect(page.getByText('Backup-Verschlüsselungs-Key fehlt')).toHaveCount(0);
});

test('Lehrer: /abgaenger direkt aufgerufen leakt keine Schülerdaten', async ({ page }) => {
	seedUsers();
	await uiLogin(page, LEHRER_EMAIL);

	// Server blockt hart (view_graduates: false)
	const api = await page.request.get('/api/abgaenger');
	expect(api.status(), 'Abgänger-API für Lehrer').toBe(403);

	// Direkter URL-Aufruf: kein Crash, keine Datenzeilen
	await page.goto('/abgaenger');
	await expect(page.locator('table')).toHaveCount(0);
	await expect(page.getByText('Barcode-ID')).toHaveCount(0);

	// Schreibende Admin-API ebenfalls dicht
	const createUser = await apiPost(page, '/api/benutzer', {
		vorname: 'Boese',
		nachname: 'Absicht',
		email: 'x@x.local',
		rolle: 'admin'
	});
	expect(createUser.status(), 'Benutzer anlegen als Lehrer').toBe(403);
});

// Mit der Leserzeile eines Kollegen fällt sein Konto. Einen Zugang löscht deshalb nur, wer auch
// Konten verwaltet (manage_users), wie in der Benutzerverwaltung; die Akte zeigt dem Mitarbeiter
// den Knopf abgeschaltet und den Grund daneben. Ob ein Zugang da ist, sagt der Server
// (`hat_zugang` der Akte): Die Lehrkraft aus Littera hat nur ein Konto mit Ersatzadresse, mit dem
// sich niemand anmeldet, und bleibt löschbar.
test('Mitarbeiter: einen Kollegen mit Zugang löscht er nicht, die Akte sagt warum', async ({
	page
}) => {
	seedUsers();
	const suffix = uniqueSuffix();
	const mitZugang = `Zugang-${suffix}`;
	const ausLittera = `Littera-${suffix}`;
	// Die Leserzeilen legt der Trigger trg_benutzer_hat_leserzeile zu jedem Konto an.
	seedSQL(`
		INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv) VALUES
			('Zora', '${mitZugang}', 'zora-${suffix}@schule.example', 'kollegium', true),
			('Lutz', '${ausLittera}', 'littera-${suffix}@littera.invalid', 'kollegium', true);
	`);
	try {
		await uiLogin(page, MITARBEITER_EMAIL);

		/** @param {string} vorname @param {string} nachname */
		const loeschknopf = async (vorname, nachname) => {
			await gehZu(page, '/schuelerdatei');
			await page.getByRole('searchbox', { name: 'Leser suchen' }).fill(nachname);
			await page
				.getByRole('button', { name: `Profil von ${vorname} ${nachname} (Lehrkraft) anzeigen` })
				.click();
			await page.getByRole('tab', { name: 'Stammdaten & Adresse' }).click();
			return page.getByRole('button', { name: 'Kollegen archivieren / löschen' });
		};

		await expect(await loeschknopf('Zora', mitZugang)).toBeDisabled();
		await expect(
			page.getByText(
				'Diese Person hat einen Zugang zu „Mein Portal“. Löschen kann sie nur, wer „Benutzer & Rechte verwalten“ darf.'
			)
		).toBeVisible();

		await expect(await loeschknopf('Lutz', ausLittera)).toBeEnabled();
		await expect(page.getByText(/Benutzer & Rechte verwalten/)).toHaveCount(0);
	} finally {
		seedSQL(`
			WITH weg AS (
				DELETE FROM benutzer
				WHERE email IN ('zora-${suffix}@schule.example', 'littera-${suffix}@littera.invalid')
				RETURNING leser_id)
			DELETE FROM leser WHERE id IN (SELECT leser_id FROM weg);
		`);
	}
});
