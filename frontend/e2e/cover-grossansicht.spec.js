import { test, expect } from '@playwright/test';
import { uiLogin, seedSQL, seedCoverDatei, uniqueSuffix } from './helpers.js';

// Die Großansicht eines Covers zeigte "Kein Coverbild hinterlegt", obwohl das
// Miniaturbild derselben Zeile das Cover anzeigte.
//
// Grund: Die Großansicht lud über den Cover-Proxy (/api/images/cover). Der ist dafür da,
// ein Bild von einem FREMDEN Server zu holen, und lässt nur DNB, OpenLibrary und Google
// Books zu; alles andere — auch ein lokaler "/uploads/…"-Pfad — beantwortet er mit einem
// transparenten 1×1-GIF (bewusst Status 200 gegen Konsolen-Spam).
//
// Und lokal ist der Regelfall: Der Cover-Sync migriert jedes gefundene Cover auf einen
// lokalen Pfad, damit nichts mehr extern lädt. Die Großansicht war also nicht in einem
// Sonderfall leer, sondern bei praktisch jedem Titel mit Cover.
//
// Der Test prüft deshalb naturalWidth, nicht Sichtbarkeit: Das Fehlerbild ist ein
// erfolgreich geladenes Bild — nur eben 1 Pixel breit.
test('Cover-Großansicht zeigt ein lokal abgelegtes Cover', async ({ page }) => {
	const s = uniqueSuffix();
	const coverPfad = seedCoverDatei(`e2e-cover-${s}`);
	const titel = `E2E-Cover-Titel ${s}`;

	seedSQL(`
		WITH t AS (
			INSERT INTO buecher_titel (titel, isbn, cover_url)
			VALUES ('${titel}', '978${s.slice(0, 10)}', '${coverPfad}')
			RETURNING id
		),
		ex AS (
			INSERT INTO buecher_exemplare (titel_id, barcode_id)
			SELECT id, 'E2E-COV-B-${s}' FROM t RETURNING id
		),
		sch AS (
			INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr, ist_abgaenger)
			VALUES ('E2E-COV-S-${s}', 'Cover${s}', 'Testschueler', '10c',
			        EXTRACT(YEAR FROM CURRENT_DATE)::int, true)
			RETURNING id
		)
		INSERT INTO ausleihen (exemplar_id, schueler_id, rueckgabe_frist)
		SELECT ex.id, sch.id, CURRENT_DATE - 5 FROM ex, sch;
	`);

	await uiLogin(page);

	// Über die Abgänger-Ansicht ins Profil — dieser Weg öffnet direkt den Reiter
	// "Ausleihen & Historie", in dem die Ausleihliste mit den Covern steht.
	await page.getByTitle('Abgänger').click();
	await page.getByRole('button', { name: new RegExp(`Profil von Cover${s} Testschueler`) }).click();
	await expect(page.getByText(titel).first()).toBeVisible();

	// Das Miniaturbild IST der Auslöser der Großansicht.
	const ausloeser = page.getByRole('button', { name: `Cover von ${titel} anzeigen` });
	await ausloeser.hover();

	const grossansicht = page.getByRole('tooltip').getByRole('img', {
		name: `Cover von ${titel}`
	});
	await expect(grossansicht).toBeVisible();
	await expect(page.getByText('Kein Coverbild hinterlegt')).toHaveCount(0);

	// BEWEIS: ein echtes Bild, kein 1×1-Platzhalter. Mit dem alten Verhalten (Umweg über
	// den Cover-Proxy) stünde hier 1.
	await expect
		.poll(async () =>
			grossansicht.evaluate((el) => /** @type {HTMLImageElement} */ (el).naturalWidth)
		)
		.toBeGreaterThan(1);
});
