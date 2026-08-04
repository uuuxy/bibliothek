// Gate gegen die verschwundene Bestellung.
//
// Die Bestellspalte lässt sich einklappen, damit die Bedarfsliste daneben mehr Breite
// bekommt — sie belegte auch mit leerem Warenkorb ein Drittel des Fensters, während
// lange Lehrmitteltitel abgeschnitten wurden.
//
// Zwei Dinge dürfen dabei nicht passieren, und beide wären im Betrieb still:
//
// 1. Der eingeklappte Streifen wird zum stummen Rechteck. Wer nicht sieht, was drin ist,
//    muss aufklappen, um nachzusehen — dann hat das Einklappen nichts gespart. Deshalb
//    trägt er Beschriftung UND die Anzahl der Positionen.
// 2. Die Bestellung ist auf dem Tablet nicht mehr erreichbar. Unterhalb von lg legt das
//    Raster gar nicht nebeneinander; der eingeklappte Zustand darf dort deshalb nicht
//    greifen. Ohne die max-lg-Ausnahme wäre die Bestellung für jeden verschwunden, der
//    am grossen Bildschirm eingeklappt und danach am Tablet weitergearbeitet hat — ohne
//    Fehlermeldung, ohne sichtbaren Weg zurück.
import { test, expect } from '@playwright/test';
import { uiLogin, seedBestellbedarf } from './helpers.js';

// Bedarf selbst mitbringen — siehe seedBestellbedarf(): ohne den Schalter
// `bestellbedarf_warnung_aktiv` ist die Liste leer, und dann gibt es nichts in den Korb
// zu legen. Der Test prüfte sonst eine Spalte, die er nie gefüllt hat.
/** @type {(() => void) | undefined} */
let aufraeumen;
test.beforeEach(() => {
	aufraeumen = seedBestellbedarf(2);
});
test.afterEach(() => {
	aufraeumen?.();
	aufraeumen = undefined;
});

const BREIT = { width: 1600, height: 1000 };
const TABLET = { width: 900, height: 1000 };

test('Bestellspalte: einklappen gibt Breite frei und nimmt die Bestellung nicht weg', async ({
	page
}) => {
	test.setTimeout(120_000);
	await page.setViewportSize(BREIT);
	await uiLogin(page);
	await page.goto('/bestellungen');

	const plus = page.getByRole('button', { name: /zur Bestellung hinzufügen/ });
	await plus.first().waitFor({ timeout: 30_000 });

	// Eine Position in den Korb — sonst prüft der Zähler auf dem Streifen nichts.
	await plus.first().click();
	const panel = page.locator('#bestellpanel');
	await expect(panel).toBeVisible();

	const bedarf = page.locator('#bestellspalte').locator('..').locator('> div').first();
	const vorher = (await bedarf.boundingBox())?.width ?? 0;
	expect(vorher, 'Die Bedarfsliste muss eine messbare Breite haben').toBeGreaterThan(0);

	// ── Einklappen ──────────────────────────────────────────────────────────────────────
	await page.getByRole('button', { name: 'Bestellspalte einklappen' }).click();

	const streifen = page.getByRole('button', { name: 'Bestellspalte ausklappen' });
	await expect(streifen).toBeVisible();
	await expect(panel).toBeHidden();

	// Die Liste muss die frei gewordene Breite tatsächlich bekommen — sonst hat das
	// Einklappen nur die Bestellung versteckt, ohne etwas zu gewinnen.
	const nachher = (await bedarf.boundingBox())?.width ?? 0;
	expect(
		nachher,
		`Die Bedarfsliste ist nach dem Einklappen ${Math.round(nachher)} px breit, vorher ` +
			`${Math.round(vorher)} px — sie hat die frei gewordene Breite nicht bekommen.`
	).toBeGreaterThan(vorher + 100);

	// Der Streifen ist beschriftet und nennt den Korbinhalt.
	await expect(streifen).toContainText('Deine Bestellung');
	await expect(
		streifen,
		'Der eingeklappte Streifen muss die Anzahl im Korb nennen — sonst muss man zum Nachsehen aufklappen'
	).toContainText('1');

	// ── Der stille Fall: schmaler Bildschirm, während eingeklappt ist ────────────────────
	await page.setViewportSize(TABLET);
	await expect(
		panel,
		'Unterhalb von lg wird nicht nebeneinander gelegt — die Bestellung muss dort sichtbar ' +
			'bleiben, auch wenn am grossen Bildschirm eingeklappt wurde. Sonst ist sie weg, ohne ' +
			'dass es einen Weg zurück gibt (der Ausklapp-Streifen ist dort ausgeblendet).'
	).toBeVisible();

	// ── Zurück auf breit, wieder aufklappen ─────────────────────────────────────────────
	await page.setViewportSize(BREIT);
	await expect(streifen).toBeVisible();
	await streifen.click();
	await expect(panel).toBeVisible();
	await expect(page.getByRole('button', { name: 'Bestellspalte einklappen' })).toBeVisible();
});
