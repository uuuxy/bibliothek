// Gate gegen die Rückkehr der doppelten Klassen-Ansicht.
//
// Bis zum 08.08.2026 trug die Oberfläche „Schulklassen" zweimal: in der Seitenleiste
// (Verwaltung → Schulklassen) und als dritter Reiter im Medienkatalog. Beide zeigten
// dieselben Daten — beide GET-Routen landen im selben handleClassBooks — in derselben
// Darstellung, aus zwei getrennt gepflegten Komponentenpaaren. Ein Umbenennen des
// Reiters hätte nur den Namen entschärft; die Dopplung blieb.
//
// Warum E2E und kein Grep: Der Reiter war nur sichtbar, WENN er ausgewählt war, und
// sein Suchfeld hing an genau dieser Bedingung — deshalb ist dieses Feld auch dem
// 36-px-Gate entgangen, das /medienkatalog längst misst. Ein Grep über die Quelle
// findet Namen, aber nicht, was ein Benutzer auf dem Bildschirm vor sich hat.
import { test, expect } from '@playwright/test';
import { uiLogin, apiPost, csrfToken } from './helpers.js';

const KLASSE_A = '09z1';
const KLASSE_B = '09z2';

test('Klassensätze stehen an genau EINEM Ort', async ({ page }) => {
	await uiLogin(page);

	// --- Medienkatalog: der Klassen-Reiter ist weg, die anderen beiden sind da ---
	// Die Gegenprobe ist der Punkt: Ohne sie wäre der Test auch dann grün, wenn die
	// Seite überhaupt nicht mehr lüde.
	await page.goto('/medienkatalog');
	const reiter = page.getByRole('tab');
	await expect(reiter.filter({ hasText: 'Buch-Suche' })).toBeVisible();
	await expect(reiter.filter({ hasText: 'Jahrgänge' })).toBeVisible();
	await expect(reiter.filter({ hasText: /Klassensätze|Schulklassen/ })).toHaveCount(0);

	// --- Zwei Klassensätze anlegen, damit „filtern" überhaupt etwas bedeutet ---
	const buecher = await (await page.request.get('/api/books')).json();
	const ids = (buecher.data ?? []).slice(0, 2).map((/** @type {any} */ b) => b.id);
	expect(ids.length, 'Testdaten: mindestens zwei Bücher nötig').toBe(2);

	await apiPost(page, '/api/admin/class-books/add', {
		classNames: [KLASSE_A],
		bookIds: [ids[0]]
	});
	await apiPost(page, '/api/admin/class-books/add', {
		classNames: [KLASSE_B],
		bookIds: [ids[1]]
	});

	try {
		// --- Schulklassen: die Seite trägt Liste UND die Klassensuche des Reiters ---
		await page.goto('/schulklassen');
		const suchfeld = page.getByLabel('Klasse suchen');
		await expect(suchfeld).toBeVisible();

		const karteA = page.locator('.class-group').filter({ hasText: KLASSE_A });
		const karteB = page.locator('.class-group').filter({ hasText: KLASSE_B });
		await expect(karteA).toHaveCount(1);
		await expect(karteB).toHaveCount(1);

		// Die Fähigkeit des alten Reiters ist mitgewandert — am Verhalten geprüft,
		// nicht am Vorhandensein des Feldes.
		await suchfeld.fill(KLASSE_A);
		await expect(karteA).toHaveCount(1);
		await expect(karteB).toHaveCount(0);

		await suchfeld.fill('');
		await expect(karteB).toHaveCount(1);
	} finally {
		// Nur die beiden angelegten Zuordnungen zurücknehmen (DELETE trifft
		// class_books, nicht die Bücher). Kein Rundumschlag: Ein Teardown hat in
		// diesem Projekt schon einmal echte Konfiguration mitgenommen.
		const token = await csrfToken(page);
		for (const klasse of [KLASSE_A, KLASSE_B]) {
			await page.request.delete(`/api/admin/class-books?className=${klasse}`, {
				headers: { 'X-CSRF-Token': token }
			});
		}
	}
});
