import { test, expect } from '@playwright/test';
import { uiLogin, csrfToken, seedBenutzer, seedSQL, uniqueSuffix, gehZu } from './helpers.js';

// LMF-Plan als Reihenfolge (Peter, 05.09.2026, am echten Plan der Schule): Der Planer
// bekommt Rahmen und Reihenfolge, der Server gießt sie auf Schultage × Stunden. Geprüft
// über den echten Klickpfad: Klasse aus „Nicht im Plan" holen → ersten Tag setzen →
// Vorschau zeigt Wochentag/Datum/Stunde passend zur Position → Zeile davor einfügen
// rückt sie eine Stunde weiter → speichern → Tabelle im Portal einer Lehrkraft → PDF →
// 403 fürs Schreiben.
//
// Geplant wird die AUSGABE: Sie setzt keine Fristen. Ein Rückgabe-Plan über die
// Seed-Klassen würde die Fristen der Testausleihen umschreiben, und die anderen Läufe
// dieser Suite rechnen mit dem Stichtag. Die Frist-Kopplung misst
// api/lmf_termine_frist_pg_test.go am Postgres.
const LEHRER_EMAIL = 'e2e-lehrer-lmfplan@test.local';
const ERSTER_TAG = new Date(2027, 7, 9); // Montag 09.08.2027
const STUNDEN_JE_TAG = 6;

/** Der Platz, den der Server der Zeile mit dieser Nummer geben muss (Mo–Fr, 6 je Tag). */
function erwarteterPlatz(/** @type {number} */ nummer) {
	const schultag = Math.floor((nummer - 1) / STUNDEN_JE_TAG);
	const datum = new Date(ERSTER_TAG);
	datum.setDate(datum.getDate() + schultag + 2 * Math.floor(schultag / 5));
	return {
		wochentag: new Intl.DateTimeFormat('de-DE', { weekday: 'long' }).format(datum),
		datum: new Intl.DateTimeFormat('de-DE', {
			day: '2-digit',
			month: '2-digit',
			year: '2-digit'
		}).format(datum),
		stunde: `${((nummer - 1) % STUNDEN_JE_TAG) + 1}. Std.`
	};
}

test('LMF-Plan: Reihenfolge planen, im Kollegiums-Portal sehen, PDF laden', async ({
	page,
	browser
}) => {
	const s = uniqueSuffix();
	const klasse = `07G${s.slice(-2)}`.toUpperCase();
	const vermerk = `E2E-Plan-${s}`;
	seedBenutzer(LEHRER_EMAIL, 'kollegium');
	// Sauberer Ausgangspunkt: kein Ausgabe-Plan, sonst erbt der Entwurf fremde Zeilen.
	seedSQL(`DELETE FROM lmf_plaene WHERE art = 'ausgabe';`);

	await uiLogin(page);
	await gehZu(page, '/schuljahr');
	await page.getByRole('button', { name: 'Bücherausgabe' }).click();
	await expect(page.getByTestId('lmf-plan-hinweis')).toContainText('Noch kein Plan');

	// Die Klasse in den Plan holen — sie landet am Ende, hinter dem Regel-Vorschlag.
	await page.getByLabel('Weitere Klasse').fill(klasse);
	await page.getByRole('button', { name: 'In den Plan' }).click();
	const tabelle = page.getByTestId('lmf-reihenfolge');
	const zeile = tabelle.getByRole('row').filter({ hasText: klasse });
	await expect(zeile).toBeVisible();
	const nummer = Number(await zeile.getByRole('cell').first().innerText());
	expect(nummer).toBeGreaterThan(0);
	// Die Klasse, die gleich dieselbe Stunde teilt: der Nachbar darüber.
	const vorherige = (
		await tabelle
			.getByRole('row')
			.nth(nummer - 1)
			.getByRole('cell')
			.nth(4)
			.innerText()
	).trim();
	expect(vorherige, 'Nachbarklasse für die geteilte Stunde').not.toBe('');

	// Ersten Tag setzen: Die Vorschau vom Server gibt der Zeile den Platz, der ihrer
	// Nummer entspricht — Wochenende übersprungen, 6 Stunden je Tag.
	await page.getByLabel('Erster Tag').fill('2027-08-09');
	let soll = erwarteterPlatz(nummer);
	await expect(zeile).toContainText(soll.wochentag);
	await expect(zeile).toContainText(soll.datum);
	await expect(zeile).toContainText(soll.stunde);
	await zeile.getByLabel(`Besonderheiten Zeile ${nummer}`).fill(vermerk);

	// Eine Zeile ohne Klasse davor: die Klasse rückt eine Stunde weiter.
	await zeile.getByLabel(`Vor Zeile ${nummer} einfügen`).click();
	let verschoben = tabelle.getByRole('row').filter({ hasText: klasse });
	soll = erwarteterPlatz(nummer + 1);
	await expect(verschoben).toContainText(soll.datum);
	await expect(verschoben).toContainText(soll.stunde);

	// Erst die Klassenzeile mit der Leerzeile darüber zusammenlegen — sie bekommt deren
	// Stunde zurück und trägt beide Vermerke.
	await tabelle
		.getByRole('row')
		.filter({ hasText: klasse })
		.getByLabel(`Zeile ${nummer + 1} mit voriger zusammenlegen`)
		.click();
	verschoben = tabelle.getByRole('row').filter({ hasText: klasse });
	soll = erwarteterPlatz(nummer);
	await expect(verschoben).toContainText(soll.stunde);
	// Der Vermerk steht in einem Eingabefeld (kein Text) — geprüft wird er im Portal,
	// wo er nach dem Speichern als Text in der Zeile steht.

	// Und dann mit der Klassenzeile darüber: ZWEI Klassen in EINER Stunde — so stehen
	// „10R1/10R2" und „6F1/6F2" im Plan der Schule (Peter, 05.09.: „das muss alles super
	// flexibel ablaufen und planbar sein").
	await verschoben.getByLabel(`Zeile ${nummer} mit voriger zusammenlegen`).click();
	const geteilt = tabelle.getByRole('row').filter({ hasText: klasse });
	soll = erwarteterPlatz(nummer - 1);
	await expect(geteilt).toContainText(soll.stunde);
	await expect(geteilt).toContainText(vorherige);
	await expect(geteilt).toContainText(klasse);

	await page.getByRole('button', { name: 'Plan speichern' }).click();
	await expect(page.getByTestId('lmf-plan-hinweis')).toContainText('Plan vom 09.08.27');

	// Das Kollegium sieht denselben Plan im Portal — ohne edit_books, nur mit Sitzung.
	const lehrerKontext = await browser.newContext();
	const lehrer = await lehrerKontext.newPage();
	try {
		await uiLogin(lehrer, LEHRER_EMAIL);
		await lehrer.getByTitle('Mein Portal').click();
		await lehrer.getByRole('tab', { name: 'LMF-Plan' }).click();
		const portal = lehrer.getByRole('region', { name: 'Bücherausgabe' });
		const portalZeile = portal.getByRole('row').filter({ hasText: vermerk });
		await expect(portalZeile).toBeVisible();
		await expect(portalZeile).toContainText(soll.wochentag);
		await expect(portalZeile).toContainText(soll.stunde);
		// Beide Klassen der geteilten Stunde stehen in EINER Zeile (Reihenfolge macht der
		// Server über den Normschlüssel, deshalb einzeln geprüft).
		await expect(portalZeile).toContainText(vorherige);
		await expect(portalZeile).toContainText(klasse);
		// Lesend: keine Planer-Aktionen im Portal.
		await expect(lehrer.getByRole('button', { name: /Plan speichern/ })).toHaveCount(0);

		const download = lehrer.waitForEvent('download');
		await lehrer.getByRole('button', { name: /Als PDF/ }).click();
		expect((await download).suggestedFilename()).toBe('LMF-Plan.pdf');

		// Schreiben ist der Rolle verwehrt — auch die Vorschau.
		const verboten = await lehrer.request.put('/api/lmf-plan/ausgabe', {
			data: {
				erster_tag: '2027-08-10',
				startstunde: 2,
				stunden_je_tag: 6,
				vorschau: true,
				zeilen: [{ klassen: [klasse], vermerk: 'verboten' }]
			},
			headers: { 'X-CSRF-Token': await csrfToken(lehrer) }
		});
		expect(verboten.status(), 'Kollegium schreibt den Plan').toBe(403);
	} finally {
		await lehrerKontext.close();
		seedSQL(`DELETE FROM lmf_plaene WHERE art = 'ausgabe';`);
	}
});

// Feiertage und Ausflüge (Peter, 05.09.2026 abends): Ein freier Tag des Plans verschiebt
// den Beginn, der Hinweis nennt ihn mit Grund; eine Zeile mit festem Platz behält Datum
// und Stunde über das Speichern hinweg — im API-Stand als fest markiert, nach dem
// Neuladen wieder als Eingabefeld. Auch hier die AUSGABE, sie setzt keine Fristen.
test('LMF-Plan: freier Tag verschiebt den Beginn, fester Platz überlebt das Speichern', async ({
	page
}) => {
	const s = uniqueSuffix();
	const klasse = `07H${s.slice(-2)}`.toUpperCase();
	seedSQL(`DELETE FROM lmf_plaene WHERE art = 'ausgabe';`);

	await uiLogin(page);
	await gehZu(page, '/schuljahr');
	await page.getByRole('button', { name: 'Bücherausgabe' }).click();
	await expect(page.getByTestId('lmf-plan-hinweis')).toContainText('Noch kein Plan');
	await page.getByLabel('Weitere Klasse').fill(klasse);
	await page.getByRole('button', { name: 'In den Plan' }).click();

	const tabelle = page.getByTestId('lmf-reihenfolge');
	const erste = tabelle.getByRole('row').nth(1);
	await page.getByLabel('Erster Tag').fill('2027-08-09'); // Montag
	await expect(erste).toContainText('09.08.27');

	// Der erste Tag wird freigehalten: Der Plan beginnt am Dienstag, der Grund steht da.
	await page.getByLabel('Freier Tag').fill('2027-08-09');
	await page.getByLabel('Grund', { exact: true }).fill('Pädagogischer Tag');
	await page.getByRole('button', { name: 'Tag freihalten' }).click();
	await expect(erste).toContainText('Dienstag');
	await expect(erste).toContainText('10.08.27');
	await expect(page.getByTestId('lmf-ausfaelle')).toContainText('Pädagogischer Tag');

	// Unsere Klasse hat am Freitag 20.08. ihren Termin — fest, egal wo sie in der
	// Reihenfolge steht.
	const zeile = tabelle.getByRole('row').filter({ hasText: klasse });
	const nummer = Number(await zeile.getByRole('cell').first().innerText());
	await zeile.getByLabel(`Zeile ${nummer} festlegen`).click();
	await zeile.getByLabel(`Fester Tag Zeile ${nummer}`).fill('2027-08-20');
	await expect(zeile).toContainText('Freitag');

	await page.getByRole('button', { name: 'Plan speichern' }).click();
	await expect(page.getByTestId('lmf-plan-hinweis')).toContainText('Plan vom 09.08.27');
	try {
		const stand = await (await page.request.get('/api/lmf-plan/ausgabe')).json();
		const gespeichert = stand.zeilen.find((/** @type {any} */ z) => z.klassen.includes(klasse));
		expect(gespeichert?.fest, 'fest-Marke im gespeicherten Plan').toBe(true);
		expect(gespeichert?.datum).toBe('2027-08-20');
		expect(stand.plan.freie_tage).toEqual([{ datum: '2027-08-09', grund: 'Pädagogischer Tag' }]);

		// Nach dem Neuladen ist der feste Platz wieder ein Eingabefeld mit seinem Datum.
		await page.reload();
		await page.getByRole('button', { name: 'Bücherausgabe' }).click();
		await expect(
			tabelle.getByRole('row').filter({ hasText: klasse }).getByLabel(`Fester Tag Zeile ${nummer}`)
		).toHaveValue('2027-08-20');
		await expect(page.getByTestId('lmf-freie-tage')).toContainText('09.08.27 Pädagogischer Tag');
	} finally {
		seedSQL(`DELETE FROM lmf_plaene WHERE art = 'ausgabe';`);
	}
});
