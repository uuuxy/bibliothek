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
const ERSTER_TAG = new Date(2027, 7, 9); // Montag 09.08.2027 — erster Schultag nach den Ferien
const STARTSTUNDE = 2; // Vorgabe des Planers (Peters Plan 2026: Mo 10.08., 2. Std.)
const STUNDEN_JE_TAG = 6;

/**
 * Wählt eine Aktion aus dem Überlaufmenü einer Zeile des Planers.
 * @param {import('@playwright/test').Page} page
 * @param {import('@playwright/test').Locator} zeile
 * @param {number} nummer
 * @param {string} eintrag
 */
async function zeilenAktion(page, zeile, nummer, eintrag) {
	await zeile.getByRole('button', { name: `Aktionen Zeile ${nummer}` }).click();
	await page.getByRole('menuitem', { name: eintrag }).click();
}

/** Der Platz, den der Server der Zeile mit dieser Nummer geben muss (Mo–Fr, 6 je Tag,
 *  am ersten Tag ab der Startstunde). */
function erwarteterPlatz(/** @type {number} */ nummer) {
	const platz = nummer - 1 + (STARTSTUNDE - 1);
	const schultag = Math.floor(platz / STUNDEN_JE_TAG);
	const datum = new Date(ERSTER_TAG);
	datum.setDate(datum.getDate() + schultag + 2 * Math.floor(schultag / 5));
	return {
		wochentag: new Intl.DateTimeFormat('de-DE', { weekday: 'long' }).format(datum),
		datum: new Intl.DateTimeFormat('de-DE', {
			day: '2-digit',
			month: '2-digit',
			year: '2-digit'
		}).format(datum),
		stunde: `${(platz % STUNDEN_JE_TAG) + 1}. Std.`
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
	await page.getByRole('button', { name: 'Bücherausgabe nach den Sommerferien' }).click();
	await expect(page.getByTestId('lmf-plan-hinweis')).toContainText('noch nicht gespeichert');

	// Die Klasse in den Plan holen — über das Dialogfenster „Andere Klasse eintragen"
	// (seit 06.09.2026 kein Dauerformular mehr). Sie landet NICHT am Ende, sondern nach
	// der Nachbar-Regel hinter der letzten Klasse ihres Jahrgangs und Zweigs (07G6 im
	// Seed), mitten im Regel-Vorschlag.
	await page.getByRole('button', { name: 'Andere Klasse eintragen' }).click();
	await page.getByLabel('Klasse', { exact: true }).fill(klasse);
	await page.getByRole('button', { name: 'In den Plan' }).click();
	const tabelle = page.getByTestId('lmf-reihenfolge');
	const zeile = tabelle.getByRole('row').filter({ hasText: klasse });
	await expect(zeile).toBeVisible();
	const nummer = Number(await zeile.getByRole('cell').first().innerText());
	expect(nummer).toBeGreaterThan(0);
	// Die Klasse, die gleich dieselbe Stunde teilt: der Nachbar darüber — derselbe
	// Jahrgang (den Zweig belegt lmfplanZeilen.test.js; die e2e-Datenbank trägt
	// Klassen anderer Läufe wie „07E29", und der Suffix hier kann Buchstaben enthalten).
	const vorherige = (
		await tabelle
			.getByRole('row')
			.nth(nummer - 1)
			.getByRole('cell')
			.nth(4)
			.innerText()
	).trim();
	// Jahrgang 7, egal in welcher Schreibweise: Die e2e-Datenbank sammelt Klassen aus
	// früheren Läufen („07E29", „7e0f") und folgt nicht dem Seed-Vokabular. Ein Test, der
	// auf „07" besteht, misst die Datenlage statt der Regel (06.09.2026 rot geworden).
	expect(vorherige, 'Nachbarklasse nach der Nachbar-Regel (Jahrgang 7)').toMatch(/^0?7/i);
	expect(
		Number(await tabelle.getByRole('row').last().getByRole('cell').first().innerText()),
		'nicht die letzte Zeile'
	).toBeGreaterThan(nummer);

	// Ersten Tag setzen (vorbelegt ist der erste Schultag nach den nächsten Ferien, hier
	// fest 2027, damit die Rechnung steht): Die Vorschau vom Server gibt der Zeile den
	// Platz, der ihrer Nummer entspricht — Wochenende übersprungen, 6 Stunden je Tag,
	// am ersten Tag ab der 2. Stunde (Vorgabe).
	await page.getByLabel('Erster Tag').fill('2027-08-09');
	await expect(page.getByLabel('Beginn am ersten Tag')).toContainText('2. Stunde');
	let soll = erwarteterPlatz(nummer);
	await expect(zeile).toContainText(soll.wochentag);
	await expect(zeile).toContainText(soll.datum);
	await expect(zeile).toContainText(soll.stunde);
	await zeile.getByLabel(`Besonderheiten Zeile ${nummer}`).fill(vermerk);

	// Escape im Überlaufmenü schließt NUR das Menü — bis 06.09.2026 sprang derselbe
	// Tastendruck zusätzlich an die Theke (Router hört vor dem Overlay auf window).
	await zeile.getByRole('button', { name: `Aktionen Zeile ${nummer}` }).click();
	await expect(page.getByRole('menu')).toBeVisible();
	await page.keyboard.press('Escape');
	await expect(page.getByRole('menu')).toHaveCount(0);
	await expect(page).toHaveURL(/\/schuljahr$/);

	// Eine Zeile ohne Klasse davor: die Klasse rückt eine Stunde weiter. Die Aktion liegt
	// im Überlaufmenü der Zeile (M3: Overflow für Aktionen, die nicht ständig sichtbar sein
	// müssen).
	await zeilenAktion(page, zeile, nummer, 'Zeile davor einfügen');
	let verschoben = tabelle.getByRole('row').filter({ hasText: klasse });
	soll = erwarteterPlatz(nummer + 1);
	await expect(verschoben).toContainText(soll.datum);
	await expect(verschoben).toContainText(soll.stunde);

	// Erst die Klassenzeile mit der Leerzeile darüber zusammenlegen — sie bekommt deren
	// Stunde zurück und trägt beide Vermerke.
	await zeilenAktion(
		page,
		tabelle.getByRole('row').filter({ hasText: klasse }),
		nummer + 1,
		'Mit der Zeile davor zusammenlegen'
	);
	verschoben = tabelle.getByRole('row').filter({ hasText: klasse });
	soll = erwarteterPlatz(nummer);
	await expect(verschoben).toContainText(soll.stunde);
	// Der Vermerk steht in einem Eingabefeld (kein Text) — geprüft wird er im Portal,
	// wo er nach dem Speichern als Text in der Zeile steht.

	// Und dann mit der Klassenzeile darüber: ZWEI Klassen in EINER Stunde — so stehen
	// „10R1/10R2" und „6F1/6F2" im Plan der Schule (Peter, 05.09.: „das muss alles super
	// flexibel ablaufen und planbar sein").
	await zeilenAktion(page, verschoben, nummer, 'Mit der Zeile davor zusammenlegen');
	const geteilt = tabelle.getByRole('row').filter({ hasText: klasse });
	soll = erwarteterPlatz(nummer - 1);
	await expect(geteilt).toContainText(soll.stunde);
	await expect(geteilt).toContainText(vorherige);
	await expect(geteilt).toContainText(klasse);

	// Die getippte Klasse hat noch keine Schüler — der Planer sagt es (Peter, 06.09.2026:
	// Klassen wechseln mit dem Schuljahr; was übrig bleibt, gehört raus oder kommt mit dem Import).
	await expect(zeile.getByText('ohne Schüler')).toBeVisible();
	await expect(page.getByTestId('lmf-ohne-schueler')).toContainText(klasse);

	await page.getByRole('button', { name: 'Plan speichern' }).click();
	// Gespeichert ist ein ENTWURF (Migration 100): sichtbar nur hier, nicht im Portal.
	await expect(page.getByTestId('lmf-plan-hinweis')).toContainText('Entwurf vom 09.08.27');
	await expect(page.getByTestId('lmf-plan-hinweis')).toContainText('noch nicht veröffentlicht');

	// Das Kollegium sieht denselben Plan im Portal — ohne edit_books, nur mit Sitzung —
	// aber erst nach dem Veröffentlichen.
	const lehrerKontext = await browser.newContext();
	const lehrer = await lehrerKontext.newPage();
	try {
		await uiLogin(lehrer, LEHRER_EMAIL);
		await lehrer.getByTitle('Mein Portal').click();
		await lehrer.getByRole('tab', { name: 'LMF-Plan' }).click();
		const portal = lehrer.getByRole('region', { name: 'Bücherausgabe nach den Sommerferien' });
		const portalZeile = portal.getByRole('row').filter({ hasText: vermerk });
		await expect(lehrer.getByText('Noch kein Plan für dieses Schuljahr')).toBeVisible();
		await expect(portalZeile).toHaveCount(0);
		// Der Entwurf ist auch über die Leitung unsichtbar, nicht nur in der Oberfläche.
		const entwurfListe = await lehrer.request.get('/api/lmf-termine');
		expect(
			(await entwurfListe.json()).termine.some((/** @type {any} */ t) => t.vermerk === vermerk),
			'Entwurf steht in der Portal-Liste'
		).toBe(false);
		expect(
			(await lehrer.request.get('/api/lmf-termine/entwurf/pdf')).status(),
			'Kollegium lädt das Entwurfs-PDF'
		).toBe(403);

		await page.getByRole('button', { name: 'Veröffentlichen' }).click();
		// Rückfrage als M3-Dialog (seit 08.09.2026 statt window.confirm).
		await page
			.getByRole('dialog', { name: 'Plan veröffentlichen?' })
			.getByRole('button', { name: 'Veröffentlichen' })
			.click();
		await expect(page.getByTestId('lmf-plan-hinweis')).toContainText('veröffentlicht am');
		await expect(page.getByRole('button', { name: 'Veröffentlichen' })).toHaveCount(0);

		await lehrer.reload();
		await lehrer.getByRole('tab', { name: 'LMF-Plan' }).click();
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
	await page.getByRole('button', { name: 'Bücherausgabe nach den Sommerferien' }).click();
	await expect(page.getByTestId('lmf-plan-hinweis')).toContainText('noch nicht gespeichert');
	await page.getByRole('button', { name: 'Andere Klasse eintragen' }).click();
	await page.getByLabel('Klasse', { exact: true }).fill(klasse);
	await page.getByRole('button', { name: 'In den Plan' }).click();

	const tabelle = page.getByTestId('lmf-reihenfolge');
	const erste = tabelle.getByRole('row').nth(1);
	await page.getByLabel('Erster Tag').fill('2027-08-09'); // Montag
	await expect(erste).toContainText('09.08.27');

	// Der erste Tag wird freigehalten — Chip „Tag freihalten", Dialogfenster, „Freihalten"
	// (06.09.2026): Der Plan beginnt am Dienstag, der Grund steht da.
	await page.getByRole('button', { name: 'Tag freihalten' }).click();
	await page.getByLabel('Freier Tag').fill('2027-08-09');
	await page.getByLabel('Grund', { exact: true }).fill('Pädagogischer Tag');
	await page.getByRole('button', { name: 'Freihalten', exact: true }).click();
	await expect(page.getByRole('dialog')).toHaveCount(0);
	await expect(erste).toContainText('Dienstag');
	await expect(erste).toContainText('10.08.27');
	await expect(page.getByTestId('lmf-ausfaelle')).toContainText('Pädagogischer Tag');

	// Unsere Klasse hat am Freitag 20.08. ihren Termin — fest, egal wo sie in der
	// Reihenfolge steht. Der Weg ist die Zelle selbst (Peter, 06.09.2026: „einfach
	// anklicken um es zu ändern … statt immer über die 3 Punkte rechts"): Klick auf das
	// Datum legt die Zeile fest, vorbelegt mit ihrem Platz, der Fokus liegt im Datumsfeld.
	// Das Zeilenmenü kennt den Weg weiterhin (LmfPlanReihenfolge.test.js).
	const zeile = tabelle.getByRole('row').filter({ hasText: klasse });
	const nummer = Number(await zeile.getByRole('cell').first().innerText());
	const vorher = await zeile.getByRole('cell').nth(2).innerText();
	await zeile.getByRole('button', { name: new RegExp(`^Datum Zeile ${nummer}:`) }).click();
	const festerTag = zeile.getByLabel(`Fester Tag Zeile ${nummer}`);
	await expect(festerTag).toBeFocused();
	expect(
		new Intl.DateTimeFormat('de-DE', { day: '2-digit', month: '2-digit', year: '2-digit' }).format(
			new Date(`${await festerTag.inputValue()}T12:00:00`)
		),
		'vorbelegt mit dem Platz der Zeile'
	).toBe(vorher.trim());
	await festerTag.fill('2027-08-20');
	await expect(zeile).toContainText('Freitag');
	// Die Zeile darunter fließt weiter: Sie bekommt den Platz, den die feste Zeile hatte.
	const naechste = tabelle.getByRole('row').nth(nummer + 1);
	await expect(naechste).toContainText(vorher.trim());

	// Klick auf die Klasse tauscht sie gegen eine aus „Noch nicht im Plan" — die Auswahl
	// öffnet sich sofort, die alte Klasse liegt danach im Vorrat, die Zeile behält den Platz.
	await page.getByRole('button', { name: /Klassen bleiben draußen/ }).click();
	const andere = (
		await page.getByTestId('lmf-vorrat-draussen').getByRole('button').first().innerText()
	)
		.replace(/\s*einplanen.*$/, '')
		.trim();
	await page.getByRole('button', { name: /Klassen bleiben draußen/ }).click();
	await zeile.getByTitle(`${klasse} gegen eine andere Klasse tauschen`).click();
	const auswahl = zeile.getByRole('combobox', { name: `Klasse Zeile ${nummer} tauschen` });
	await expect(auswahl).toHaveAttribute('aria-expanded', 'true');
	await page.getByRole('option', { name: andere, exact: true }).click();
	const getauscht = tabelle.getByRole('row').nth(nummer);
	await expect(getauscht).toContainText(andere);
	await expect(getauscht).toContainText('Freitag');
	await expect(page.getByRole('button', { name: `${klasse} einplanen` })).toBeVisible();
	// Und zurück, damit der Rest des Tests unsere Klasse wiederfindet — im Vorrat trägt
	// sie den Zusatz „ohne Schüler", deshalb kein exakter Name.
	await getauscht.getByTitle(`${andere} gegen eine andere Klasse tauschen`).click();
	await page.getByRole('option', { name: `${klasse} · ohne Schüler` }).click();
	await expect(zeile).toContainText('Freitag');

	await page.getByRole('button', { name: 'Plan speichern' }).click();
	await expect(page.getByTestId('lmf-plan-hinweis')).toContainText('Entwurf vom 09.08.27');
	try {
		const stand = await (await page.request.get('/api/lmf-plan/ausgabe')).json();
		const gespeichert = stand.zeilen.find((/** @type {any} */ z) => z.klassen.includes(klasse));
		expect(gespeichert?.fest, 'fest-Marke im gespeicherten Plan').toBe(true);
		expect(gespeichert?.datum).toBe('2027-08-20');
		expect(stand.plan.freie_tage).toEqual([{ datum: '2027-08-09', grund: 'Pädagogischer Tag' }]);

		// Nach dem Neuladen ist der feste Platz wieder ein Eingabefeld mit seinem Datum.
		await page.reload();
		await page.getByRole('button', { name: 'Bücherausgabe nach den Sommerferien' }).click();
		await expect(
			tabelle.getByRole('row').filter({ hasText: klasse }).getByLabel(`Fester Tag Zeile ${nummer}`)
		).toHaveValue('2027-08-20');
		await expect(page.getByTestId('lmf-freie-tage')).toContainText('09.08.27 Pädagogischer Tag');
	} finally {
		seedSQL(`DELETE FROM lmf_plaene WHERE art = 'ausgabe';`);
	}
});

// Der Büchertausch vor den Sommerferien hängt am ENDE (Peter, 06.09.2026: „es endet immer
// am gleichen Tag — Donnerstags vor den Ferien zur vierten Stunde"): Der Planer belegt
// den letzten Tag aus der Ferientabelle Hessen vor (ein Donnerstag), die letzte Zeile
// liegt in der 4. Stunde dieses Tages, und der Satz unter dem Rahmen nennt den
// gerechneten Beginn. Nur Vorschau, nichts wird gespeichert — ein Rückgabe-Plan über die
// Seed-Klassen würde nach dem Veröffentlichen Fristen setzen. Läuft die Ferientabelle
// aus (nach 2030), wird dieser Test rot — so wie TestSommerferienHessen_Horizont im Go.
test('LMF-Plan: Büchertausch endet am Donnerstag vor den Ferien in der 4. Stunde', async ({
	page
}) => {
	seedSQL(`DELETE FROM lmf_plaene WHERE art = 'rueckgabe';`);
	await uiLogin(page);
	await gehZu(page, '/schuljahr');
	await expect(page.getByTestId('lmf-plan-hinweis')).toContainText('noch nicht gespeichert');

	const letzterTag = page.getByLabel('Letzter Tag');
	const wert = await letzterTag.inputValue();
	expect(wert, 'letzter Tag vorbelegt').toMatch(/^\d{4}-\d{2}-\d{2}$/);
	expect(new Date(`${wert}T12:00:00`).getDay(), 'ein Donnerstag').toBe(4);
	await expect(page.getByLabel('Ende am letzten Tag')).toContainText('4. Stunde');
	await expect(page.getByLabel('Erster Tag')).toHaveCount(0);
	await expect(page.getByTestId('lmf-zeitraum-hinweis')).toContainText('Sommerferien');

	// Die letzte Zeile der Reihenfolge liegt auf dem Anker, die Vorschau nennt den Beginn.
	const zeilen = page.getByTestId('lmf-reihenfolge').getByRole('row');
	const letzte = zeilen.last();
	await expect(letzte).toContainText('4. Std.');
	await expect(letzte).toContainText(
		new Intl.DateTimeFormat('de-DE', { day: '2-digit', month: '2-digit', year: '2-digit' }).format(
			new Date(`${wert}T12:00:00`)
		)
	);
	await expect(page.getByTestId('lmf-zeitraum-hinweis')).toContainText('Der Plan beginnt');

	// Eine Zeile mehr: das Ende bleibt, der Beginn rückt nach vorn.
	const beginnVorher = await page.getByTestId('lmf-zeitraum-hinweis').innerText();
	const erste = zeilen.nth(1);
	await erste.getByRole('button', { name: 'Aktionen Zeile 1' }).click();
	await page.getByRole('menuitem', { name: 'Zeile davor einfügen' }).click();
	await expect(zeilen.last()).toContainText('4. Std.');
	await expect(page.getByTestId('lmf-zeitraum-hinweis')).not.toHaveText(beginnVorher);

	// „Noch nicht im Plan" (06.09.2026): Die Oberstufe liegt eingeklappt hinter „bleiben
	// draußen" — offen steht nichts, die Regel hat jede Klasse eingeordnet. Ausgeklappt
	// plant ein Klick 12T1 ein; ohne Jahrgang 12 im Plan ans Ende. „An den Anfang" holt
	// die Zeile mit einem Klick nach oben — vorher 60 Pfeilklicks.
	await expect(page.getByText('Noch nicht im Plan:')).toHaveCount(0);
	await page.getByRole('button', { name: /Klassen bleiben draußen/ }).click();
	const draussen = page.getByTestId('lmf-vorrat-draussen').getByRole('button');
	const namen = (await draussen.allInnerTexts()).map((t) =>
		t.replace(/\s*einplanen.*$/, '').trim()
	);
	const klasseA = namen[0];
	const klasseB = namen[namen.length - 1]; // der letzte Chip: der Tabelle am nächsten
	await draussen.first().click();
	const zeileA = zeilen.filter({ hasText: klasseA });
	await expect(zeileA).toHaveCount(1);
	await zeileA.getByRole('button', { name: /Aktionen Zeile/ }).click();
	await page.getByRole('menuitem', { name: 'An den Anfang' }).click();
	await expect(zeilen.nth(1)).toContainText(klasseA);
	await expect(zeilen.nth(1).getByRole('cell').first()).toHaveText('1');

	// Ein Chip auf eine Zeile ziehen setzt die Klasse DAVOR: auf Zeile 2 → Zeile 2.
	// Von Hand mit der Maus, nicht locator.dragTo: hover() scrollt das Ziel erst in den
	// Blick, und dann liegt unter der gedrückten Maus nicht mehr der Chip, sondern eine
	// Zeile — der Drag startete auf Zeile 3 statt auf dem Chip (Wegwerf-Probe 06.09.2026).
	// Deshalb der letzte Chip (der Tabelle am nächsten), einmal in die Mitte gescrollt,
	// dann die Koordinaten beider nehmen und die Maus ohne weiteres Scrollen führen.
	const chip = page.getByRole('button', { name: `${klasseB} einplanen` });
	await chip.evaluate((el) => el.scrollIntoView({ block: 'center' }));
	const von = await chip.boundingBox();
	const nach = await zeilen.nth(2).boundingBox();
	if (!von || !nach) throw new Error('Chip oder Zielzeile ohne Geometrie');
	const fenster = page.viewportSize();
	expect(nach.y + nach.height, 'Zielzeile im Fenster').toBeLessThan(fenster?.height ?? 720);
	await page.mouse.move(von.x + von.width / 2, von.y + von.height / 2);
	await page.mouse.down();
	await page.mouse.move(von.x + von.width / 2 + 10, von.y + von.height / 2 + 10);
	await page.mouse.move(nach.x + nach.width / 2, nach.y + nach.height / 2, { steps: 8 });
	await page.mouse.move(nach.x + nach.width / 2, nach.y + nach.height / 2 + 1);
	await page.mouse.up();
	await expect(zeilen.nth(2)).toContainText(klasseB);
	await expect(page.getByRole('button', { name: `${klasseB} einplanen` })).toHaveCount(0);

	// Die Kopfleiste haftet beim Scrollen oben: „Plan speichern" bleibt im Fenster, auch
	// wenn man in Zeile 60 arbeitet (06.09.2026; vorher zwei Bildschirmhöhen entfernt).
	// Der Scroll-Container ist <main class="overflow-y-auto"> (App.svelte), nicht window;
	// die Backup-Warnung steht DARÜBER, deshalb zählt der Abstand zur Oberkante von main.
	// Gemessen wird auch, dass wirklich GESCROLLT wurde: Bei einer kurzen Tabelle bliebe
	// „Plan speichern" auch ohne Haftung im Bild, und das Gate wäre grün ohne Messung
	// (Rasterdurchgang 06.09.2026, Frage 7).
	const { oben, gescrollt } = await page.getByTestId('lmf-reihenfolge').evaluate((el) => {
		let p = el.parentElement;
		while (
			p &&
			!(
				p.scrollHeight > p.clientHeight + 50 &&
				['auto', 'scroll'].includes(getComputedStyle(p).overflowY)
			)
		)
			p = p.parentElement;
		if (!p) throw new Error('kein Scroll-Container');
		p.scrollTop = 600;
		return { oben: p.getBoundingClientRect().top, gescrollt: p.scrollTop };
	});
	expect(gescrollt, 'der Bereich wurde wirklich gescrollt').toBeGreaterThan(400);
	const speichern = page.getByRole('button', { name: 'Plan speichern' });
	await expect(speichern).toBeInViewport();
	const lage = await speichern.boundingBox();
	expect((lage?.y ?? 999) - oben, 'Leiste haftet an der Oberkante des Scrollbereichs').toBeLessThan(
		100
	);
});

// Verlassen-Schutz (Register B, 07.09.2026): Bis dahin ging eine ungespeicherte
// Änderung beim Klick auf einen anderen Menüpunkt wortlos verloren. Jetzt hält der
// uiStore den Wechsel an, der Planer fragt — „Bleiben" lässt alles stehen, „Verwerfen
// und weiter" führt den Wechsel aus. Gemessen an Adresse UND Feldinhalt.
test('LMF-Plan: Menüwechsel mit ungespeicherter Änderung fragt nach', async ({ page }) => {
	await uiLogin(page);
	await gehZu(page, '/schuljahr');
	await page.getByRole('button', { name: 'Bücherausgabe nach den Sommerferien' }).click();
	const tabelle = page.getByTestId('lmf-reihenfolge');
	await expect(tabelle.getByRole('row').nth(1)).toBeVisible();
	const nummer = Number(
		await tabelle.getByRole('row').nth(1).getByRole('cell').first().innerText()
	);
	const feld = tabelle.getByLabel(`Besonderheiten Zeile ${nummer}`);
	await feld.fill('E2E ungespeichert');

	await page.getByTitle('Ausleihe').click();
	const dialog = page.getByRole('dialog', { name: 'Ungespeicherte Änderungen' });
	await expect(dialog).toBeVisible();
	await dialog.getByRole('button', { name: 'Bleiben' }).click();
	await expect(dialog).toBeHidden();
	await expect(page).toHaveURL(/\/schuljahr$/);
	await expect(feld, 'die Eingabe steht noch').toHaveValue('E2E ungespeichert');

	await page.getByTitle('Ausleihe').click();
	await dialog.getByRole('button', { name: 'Verwerfen und weiter' }).click();
	await expect(page).toHaveURL(/\/kiosk$/);
});
