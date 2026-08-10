// Gate gegen die Rückkehr des Buch-Karussells im Medienkatalog → Jahrgänge.
//
// Bis zum 10.08.2026 lagen die Bücher eines Jahrgangs in einem waagerecht scrollenden
// Streifen. Drei Fehler auf einmal:
//   1. Die Kacheln waren 176 px breit (w-40 + gap-4). Sechzehn Bücher brauchten 2.816 px,
//      sichtbar war rund die Hälfte — der Rest lag außerhalb des Bildes.
//   2. Die Blätterpfeile standen auf opacity:0 und erschienen erst bei :hover. Am Tablet
//      am Pult gibt es kein :hover, dort waren sie NIE zu sehen.
//   3. Der Verlauf über den Rändern war rgba(9,9,11,.95) — ein fast schwarzer Schleier
//      auf weißem Grund, übrig aus einem dunklen Entwurf.
//
// Warum das gemessen und nicht gegrept wird: Punkt 1 steht nirgends im Markup. Er ergibt
// sich erst aus Kachelbreite mal Anzahl gegen die Fensterbreite, und genau deshalb ist er
// beim Lesen der Quelle zweimal durchgerutscht — dieselbe Ansicht existierte ein zweites
// Mal unter Verwaltung → Schulklassen und wurde dort schon am 08.08. umgebaut.
//
// Die Assertion ist bewusst scrollWidth gegen clientWidth und nicht „Pfeil sichtbar":
// Playwrights toBeVisible() wertet opacity:0 NICHT als unsichtbar, ein Test auf die
// Pfeile wäre also grün gewesen, während niemand sie sah.
import { test, expect } from '@playwright/test';
import { uiLogin, seedSQL } from './helpers.js';

const ZWEIG = 'ZTEST';
const GROSS = { stufe: 9, buecher: 16 };
const KLEIN = { stufe: 8, buecher: 2 };

const nameGross = `Klasse ${GROSS.stufe} ${ZWEIG}`;
const rasterGross = `#jahrgang-${nameGross.replace(/\s+/g, '-')}`;

// Zwei Jahrgänge, nicht einer: Bleibt nach dem Filtern genau EIN Jahrgang übrig, rendert
// die Ansicht ihn absichtlich als Überschrift ohne Aufklapp-Schalter (ein Schalter, der
// nur zwischen Inhalt und leerer Seite umlegt, sagt nichts). Mit einem einzigen Jahrgang
// prüfte dieser Test also den Sonderfall statt den Normalfall.
function seed() {
	for (const { stufe, buecher } of [GROSS, KLEIN]) {
		seedSQL(`
			UPDATE buecher_titel SET grade_level = ${stufe}, track = '${ZWEIG}'
			WHERE id IN (
				SELECT id FROM buecher_titel
				WHERE grade_level IS NULL AND track IS NULL
				ORDER BY id LIMIT ${buecher}
			);
		`);
	}
}

// Nur die Titel zurücksetzen, die dieser Test angefasst hat — erkennbar am Zweig-Marker,
// und ausgewählt wurden ohnehin nur Titel, die vorher auf NULL standen. Ein Teardown hat
// in diesem Projekt schon einmal echte Konfiguration mitgenommen.
function aufraeumen() {
	seedSQL(`UPDATE buecher_titel SET grade_level = NULL, track = NULL WHERE track = '${ZWEIG}';`);
}

test.beforeAll(() => {
	aufraeumen();
	seed();
});
test.afterAll(aufraeumen);

test('Jahrgänge: Bücher stehen im Raster, nicht in einem Streifen außerhalb des Bildes', async ({
	page
}) => {
	await uiLogin(page);
	await page.goto('/medienkatalog');
	await page.getByRole('tab', { name: 'Jahrgänge' }).click();

	const zeile = page.getByRole('button', { name: new RegExp(nameGross) });
	await expect(zeile).toBeVisible();

	// Zugeklappt: Die Kacheln sind nicht bloß ausgeblendet, sie stehen gar nicht im DOM.
	// Sonst blieben bei zwanzig Jahrgängen mehrere hundert Kacheln im Baum stehen.
	await expect(zeile).toHaveAttribute('aria-expanded', 'false');
	await expect(page.locator(rasterGross)).toHaveCount(0);

	await zeile.click();
	await expect(zeile).toHaveAttribute('aria-expanded', 'true');

	const raster = page.locator(rasterGross);
	await expect(raster).toBeVisible();

	// Gegenprobe: Ohne sie wäre der Test auch dann grün, wenn gar keine Bücher kämen.
	const kacheln = raster.locator('> div');
	await expect(kacheln).toHaveCount(GROSS.buecher);

	// Der Kern. Im Karussell war scrollWidth ~2.816 gegen clientWidth ~1.100.
	const mass = await raster.evaluate((el) => ({
		scrollWidth: el.scrollWidth,
		clientWidth: el.clientWidth
	}));
	expect(
		mass.scrollWidth,
		`Das Buch-Raster scrollt waagerecht (${mass.scrollWidth} px Inhalt auf ${mass.clientWidth} px ` +
			`Fläche). Damit liegen Bücher außerhalb des Bildes und sind nur über eine Blätter-Geste ` +
			`erreichbar — genau der Zustand, den der Umbau vom 10.08.2026 beendet hat.`
	).toBeLessThanOrEqual(mass.clientWidth + 1);

	// Und jede einzelne Kachel liegt innerhalb der Fläche. scrollWidth allein würde einen
	// Streifen mit overflow:hidden durchlassen — dann ist nichts scrollbar UND nichts zu sehen.
	const rasterBox = await raster.boundingBox();
	for (let i = 0; i < GROSS.buecher; i++) {
		const box = await kacheln.nth(i).boundingBox();
		expect(box, `Kachel ${i} hat keine Fläche`).not.toBeNull();
		expect(box.x + box.width, `Kachel ${i} ragt rechts aus dem Raster heraus`).toBeLessThanOrEqual(
			rasterBox.x + rasterBox.width + 1
		);
	}

	// Wieder zuklappen: Die Übersicht über die Jahrgänge ist der Zweck der Liste.
	await zeile.click();
	await expect(zeile).toHaveAttribute('aria-expanded', 'false');
	await expect(page.locator(rasterGross)).toHaveCount(0);
});
