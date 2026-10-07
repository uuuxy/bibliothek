// Gate für „Auf anderem Weg bestellt" an einer Bestellung, deren Mail nicht rausging.
//
// Warum E2E: Block, Rückfrage und DELETE /api/bestellungen/{id}/mail sind einzeln getestet.
// Ob der Klick durch den Dialog bis in die Datenbank kommt, zeigt nur der Weg im Browser.
// Gesendet wird dabei nichts: Die Bestellung entsteht per SQL, die Tür verschickt keine Mail.
import { test, expect } from '@playwright/test';
import { uiLogin, seedBestellung, seedSQL, querySQL, oeffneBestellungsDetail } from './helpers.js';

test('„Auf anderem Weg bestellt" entfernt den Hinweis nach der Rückfrage', async ({ page }) => {
	const { marke, aufraeumen } = seedBestellung();
	const dieBestellung = `bestellungen_verlauf WHERE lieferant_name = '${marke}'`;
	seedSQL(`UPDATE ${dieBestellung.replace(' WHERE', ' SET mail_gescheitert_am = now() WHERE')}`);

	try {
		await uiLogin(page);
		await page.goto('/bestellungen');
		await oeffneBestellungsDetail(page, marke);

		const hinweis = page.getByText('Die Bestellmail ist nicht rausgegangen');
		await expect(hinweis).toBeVisible();

		// Beide Knöpfe stehen in einer Zeile, der leisere links, gleich hoch.
		const anders = page.getByRole('button', { name: 'Auf anderem Weg bestellt' });
		const erneut = page.getByRole('button', { name: 'Erneut senden' });
		const a = await anders.boundingBox();
		const e = await erneut.boundingBox();
		expect(a && e, 'beide Knöpfe sind messbar').toBeTruthy();
		if (a && e) {
			expect(Math.abs(a.y - e.y), 'eine Zeile').toBeLessThan(1);
			expect(a.x + a.width, 'der Text-Knopf steht links').toBeLessThanOrEqual(e.x);
			expect(a.height, 'gleiche Höhe').toBe(e.height);
		}

		// Der Fokus steht auf „Abbrechen": Ein Enter, etwa aus dem Handscanner, lässt alles, wie
		// es ist.
		await anders.click();
		const dialog = page.getByRole('dialog');
		await expect(dialog.getByRole('heading', { name: 'Auf anderem Weg bestellt?' })).toBeVisible();
		await expect(dialog.getByRole('button', { name: 'Abbrechen' })).toBeFocused();
		await page.keyboard.press('Enter');
		await expect(dialog).toBeHidden();
		await expect(hinweis).toBeVisible();
		expect(querySQL(`SELECT mail_gescheitert_am IS NOT NULL FROM ${dieBestellung}`)).toBe('t');

		// Die Bestätigung entfernt den Hinweis, an der Bestellung und in der Datenbank.
		await anders.click();
		await dialog.getByRole('button', { name: 'Hinweis entfernen' }).click();
		await expect(hinweis).toBeHidden();
		await expect(page.getByRole('button', { name: 'Erneut senden' })).toHaveCount(0);
		expect(querySQL(`SELECT mail_gescheitert_am IS NULL FROM ${dieBestellung}`)).toBe('t');
		expect(
			querySQL(`SELECT count(*) FROM audit_logs
				WHERE aktion = 'BESTELLMAIL_VERMERK_ENTFERNT'
				  AND details->>'bestellung_id' = (SELECT id::text FROM ${dieBestellung})`)
		).toBe('1');
	} finally {
		seedSQL(`DELETE FROM audit_logs
			WHERE aktion = 'BESTELLMAIL_VERMERK_ENTFERNT'
			  AND details->>'bestellung_id' = (SELECT id::text FROM ${dieBestellung})`);
		aufraeumen();
	}
});
