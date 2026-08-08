// Gate für die Trennung von „Dokument" und „Eingriff" im Schülerprofil.
//
// Die Sperre stand bis zum 08.08.2026 im Kasten „Dokumente & Aktionen", zwischen vier
// Knöpfen, die nur ein PDF erzeugen. Ihre Position kam aus einem ml-auto in einem
// flex-wrap-Container — das schiebt nur innerhalb der aktuellen ZEILE nach rechts.
// Ergebnis: Bei breitem Fenster stand die folgenreichste Aktion des Bildschirms
// unabgesetzt am Zeilenende, bei schmalem rutschte sie allein nach unten rechts, also
// genau dorthin, wo ein Dialog seinen Bestätigen-Knopf hat.
//
// Deshalb prüft dieser Test bei ZWEI Fensterbreiten. Eine einzelne Breite hätte den
// Fehler nie gezeigt — er bestand ja darin, dass das Ergebnis von der Breite abhing.
import { test, expect } from '@playwright/test';
import { uiLogin } from './helpers.js';

const BREITEN = [
	{ name: 'breit', viewport: { width: 1680, height: 1000 } },
	{ name: 'schmal', viewport: { width: 1180, height: 900 } }
];

/** Öffnet das erste Schülerprofil der Schülerdatei. */
async function ersterSchueler(page) {
	await page.goto('/schuelerdatei');
	const treffer = page.locator('tbody tr').first();
	await treffer.waitFor();
	await treffer.click();
	await expect(page.getByText('Konto-Status')).toBeVisible();
}

for (const { name, viewport } of BREITEN) {
	test(`${name}: die Sperre steht beim Konto-Status, nicht bei den Dokumenten`, async ({
		page
	}) => {
		await page.setViewportSize(viewport);
		await uiLogin(page);
		await ersterSchueler(page);

		const sperren = page.getByRole('button', { name: /Schüler sperren|Sperre aufheben/ });
		await expect(sperren).toBeVisible();

		// Der Dokumente-Kasten darf die Sperre NICHT mehr enthalten.
		//
		// Über die Überschrift und deren Elternelement, NICHT über einen Textfilter auf
		// 'div': Ein solcher Filter trifft jeden Vorfahren, der den Text irgendwo
		// enthält — bis hinauf zum Seitenrumpf, der die Sperre natürlich mit umfasst.
		// Der erste Anlauf dieses Tests war deshalb rot, obwohl der Umbau stimmte.
		const dokumenteKasten = page
			.getByRole('heading', { name: 'Dokumente', exact: true })
			.locator('xpath=..');
		await expect(
			dokumenteKasten.getByRole('button', { name: /Schüler sperren|Sperre aufheben/ })
		).toHaveCount(0);

		// Und sie steht wirklich beim Status: derselbe Block trägt beides.
		const statusBereich = page.getByText('Konto-Status').locator('xpath=ancestor::div[2]');
		await expect(
			statusBereich.getByRole('button', { name: /Schüler sperren|Sperre aufheben/ })
		).toHaveCount(1);

		// Gegenprobe gegen einen stillen Nulllauf: Die Dokumente-Knöpfe gibt es noch.
		await expect(page.getByRole('button', { name: 'Kontoauszug' })).toBeVisible();
	});
}

// Der graue Knopf muss sagen, WARUM er grau ist. Der native title tut das nicht:
// Ein disabled-Element bekommt keine Zeigerereignisse, die Blase erschien nie — die
// Begründung stand im Code und erreichte niemanden.
test('Ersatzforderung erklärt im gesperrten Zustand, woran es liegt', async ({ page }) => {
	await uiLogin(page);
	await ersterSchueler(page);

	const knopf = page.getByRole('button', { name: 'Ersatzforderung' });
	await expect(knopf).toBeVisible();
	await expect(knopf, 'Testdaten: Schüler ohne Schadensfall erwartet').toBeDisabled();

	await knopf.hover({ force: true });
	const blase = page.locator('[data-tooltip-blase]');
	await expect(blase).toBeVisible({ timeout: 3000 });
	await expect(blase).toContainText('Kein offener Schadensfall');
});
