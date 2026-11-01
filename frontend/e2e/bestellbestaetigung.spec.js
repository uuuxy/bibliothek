import { test, expect } from '@playwright/test';
import { seedSQL, querySQL, uniqueSuffix, uiLogin, menuepunkt } from './helpers.js';

// Der Bestätigungs-Link, den der Lieferant aus der Bestellmail öffnet.
//
// Warum als E2E und nicht nur als Go-Test: Die entscheidende Eigenschaft dieser Seite
// ist, dass sie OHNE ANMELDUNG funktioniert. Ein Go-Test ruft den Handler direkt auf und
// kann das gar nicht bemerken — käme versehentlich ein Auth-Wächter davor oder liefe der
// Pfad in die Login-Maske, bliebe er trotzdem grün, und der Lieferant stünde vor einem
// Anmeldebildschirm, für den er kein Konto hat.
//
// Der Test meldet sich deshalb bewusst NIE an.

/**
 * Legt Lieferant + Bestellung samt Token an und liefert den Klartext-Token zurück.
 * `mittel` ist der Topf ('land' | 'schultraeger'); ohne Angabe bleibt die Spalte NULL —
 * die Alt-Bestellung ohne Zuordnung (Migration 109).
 */
function seedBestellungMitLink(token, { gueltigTage = 30, mittel = null } = {}) {
	const s = uniqueSuffix();
	seedSQL(`
		-- Erst räumen, dann setzen — genau wie setzeHauptlieferant im Handler.
		-- Hauptlieferant darf höchstens EINER sein (idx_lieferanten_ein_hauptlieferant,
		-- Migration 066); ohne diese Zeile bräche schon der zweite Seed in dieser Datei
		-- mit einer Unique-Verletzung ab.
		UPDATE lieferanten SET ist_hauptlieferant = false WHERE ist_hauptlieferant;

		WITH l AS (
			INSERT INTO lieferanten (name, email, kundennummer, ist_hauptlieferant)
			VALUES ('E2E-Naacher ${s}', 'e2e-${s}@example.invalid', 'K-${s}', true)
			RETURNING id
		),
		t AS (
			-- isbn bewusst NULL statt '': Die Spalte ist UNIQUE, und der Leerstring IST ein
			-- Wert — zwei Seeds ohne ISBN kollidierten. Der Produktivcode schreibt an dieser
			-- Stelle ebenfalls NULL (NULLIF in UpsertBookTitle).
			INSERT INTO buecher_titel (titel, autor, isbn, signatur)
			VALUES ('E2E-Bestaetigung ${s}', 'Autor', NULL, 'LMF-E2E ${s}')
			RETURNING id
		),
		b AS (
			INSERT INTO bestellungen_verlauf
				(lieferant_id, lieferant_name, lieferant_email, kundennummer, anzahl_exemplare,
				 bestaetigungs_token_hash, token_gueltig_bis, mittel)
			SELECT l.id, 'E2E-Naacher ${s}', 'e2e-${s}@example.invalid', 'K-${s}', 2,
			       encode(sha256('${token}'::bytea), 'hex'),
			       now() + make_interval(days => ${gueltigTage}),
			       ${mittel ? `'${mittel}'` : 'NULL'}
			FROM l RETURNING id
		),
		p AS (
			INSERT INTO bestellungen_positionen
				(bestellung_id, titel_id, titel_name, isbn, menge, einzelpreis, mit_vorab_barcode)
			SELECT b.id, t.id, 'E2E-Bestaetigung ${s}', '', 2, 0, true FROM b, t
			RETURNING bestellung_id
		)
		INSERT INTO buecher_exemplare (titel_id, barcode_id, bestellung_id, ist_ausleihbar)
		SELECT t.id, 'E2E-BC-${s}-' || g, b.id, false FROM t, b, generate_series(1,2) g;
	`);
	return s;
}

test('Lieferant öffnet den Link ohne Anmeldung, druckt Etiketten und bestätigt — genau einmal', async ({
	page
}) => {
	const token = `E2E-TOKEN-${uniqueSuffix()}`;
	const s = seedBestellungMitLink(token);

	await page.goto(`/bestellung/${token}`);

	// Kein Login-Bildschirm, sondern die Bestellung.
	await expect(page.getByRole('heading', { name: /Bestellung vom/ })).toBeVisible();
	await expect(page.getByText(`E2E-Bestaetigung ${s}`)).toBeVisible();
	await expect(page.getByRole('button', { name: /Anmelden|Login/ })).toHaveCount(0);

	// Die Etiketten sind abrufbar — geprüft am echten PDF, nicht am Knopf.
	const antwort = await page.request.get(`/api/public/bestellung/${token}/etiketten/gross`);
	expect(antwort.status()).toBe(200);
	expect(antwort.headers()['content-type']).toContain('application/pdf');

	await page.getByRole('button', { name: 'Bestellung jetzt bestätigen' }).click();
	await expect(page.getByRole('heading', { name: 'Bestellung bestätigt' })).toBeVisible();

	// BEWEIS in der Datenbank: Die Bibliothek sieht, dass der LIEFERANT bestätigt hat —
	// nicht jemand aus dem Haus.
	const durch = querySQL(
		`SELECT bestaetigt_durch FROM bestellungen_verlauf WHERE kundennummer = 'K-${s}';`
	);
	expect(durch.trim()).toBe('lieferant');

	// Nach dem Neuladen bleibt es bei der Quittung — kein zweiter Bestätigen-Knopf.
	await page.reload();
	await expect(page.getByRole('button', { name: 'Bestellung jetzt bestätigen' })).toHaveCount(0);
});

// Der Händler bekommt am selben Tag zwei gleich aussehende Links (OFFEN.md 4.11, entschieden
// am 21.09.2026): Die Seite nennt den Topf, und zur Bestellung für die Schülerbücherei gibt es
// das große Lernmittel-Etikett „Eigentum des Landes" nicht — weder als Knopf noch an der Tür.
const GROSSER_KNOPF = /Große Lernmittel-Etiketten/;

test('Schülerbücherei: die Seite nennt den Topf, das große Lernmittel-Etikett gibt es nicht', async ({
	page
}) => {
	const token = `E2E-SB-${uniqueSuffix()}`;
	seedBestellungMitLink(token, { mittel: 'schultraeger' });

	await page.goto(`/bestellung/${token}`);
	await expect(page.getByRole('heading', { name: /Bestellung vom/ })).toBeVisible();
	await expect(page.getByText(/^Schülerbücherei · E2E-Naacher/)).toBeVisible();
	// Eine Größe, also kein „klein": Das Wort braucht ein Gegenstück.
	await expect(
		page.getByRole('button', { name: 'Etiketten (Bogen A4)', exact: true })
	).toBeVisible();
	await expect(page.getByLabel('Bogenraster der Etiketten', { exact: true })).toBeVisible();
	await expect(page.getByText(/klein/i)).toHaveCount(0);
	await expect(page.getByRole('button', { name: GROSSER_KNOPF })).toHaveCount(0);

	// Die Tür, nicht nur der Knopf: Wer die Adresse kennt, bekommt das Etikett auch nicht.
	const gross = await page.request.get(`/api/public/bestellung/${token}/etiketten/gross`);
	expect(gross.status()).toBe(404);
	const klein = await page.request.get(`/api/public/bestellung/${token}/etiketten/klein`);
	expect(klein.status()).toBe(200);
});

// Gegenprobe: Bei der Lernmittel-Bestellung steht der Topf ebenso da, und beide Größen gehen.
test('Lernmittelfreiheit: die Seite nennt den Topf und bietet beide Größen an', async ({
	page
}) => {
	const token = `E2E-LMF-${uniqueSuffix()}`;
	seedBestellungMitLink(token, { mittel: 'land' });

	await page.goto(`/bestellung/${token}`);
	await expect(page.getByText(/^Lernmittelfreiheit · E2E-Naacher/)).toBeVisible();
	await expect(page.getByRole('button', { name: GROSSER_KNOPF })).toBeVisible();
	await expect(page.getByRole('button', { name: /Kleine Etiketten/ })).toBeVisible();
	const gross = await page.request.get(`/api/public/bestellung/${token}/etiketten/gross`);
	expect(gross.status()).toBe(200);
});

test('Ein ungültiger Link zeigt keine Daten und keinen Anmeldebildschirm', async ({ page }) => {
	await page.goto('/bestellung/DIESEN-TOKEN-GIBT-ES-NICHT');

	await expect(page.getByText('Dieser Link ist nicht mehr gültig')).toBeVisible();
	await expect(page.getByRole('heading', { name: /Bestellung vom/ })).toHaveCount(0);
});

test('Ein abgelaufener Link ist tot — auch wenn die Bestellung existiert', async ({ page }) => {
	const token = `E2E-ALT-${uniqueSuffix()}`;
	seedBestellungMitLink(token, { gueltigTage: -1 });

	await page.goto(`/bestellung/${token}`);
	await expect(page.getByText('Dieser Link ist nicht mehr gültig')).toBeVisible();
});

// Der Kreis schliesst sich erst hier: Was nützt die Bestätigung des Händlers, wenn sie im
// Haus niemand sieht? Vorher stand der Status IN der Lieferantenspalte, die truncate
// trägt — das Chip war auf wenige Pixel zerquetscht und praktisch unlesbar.
// Der Link steht nur einmal da: In der Datenbank liegt bloß sein Hash. Baut das Neuladen danach die
// Ansicht neu auf, ist er weg und der alte schon tot; am Server allein sieht das niemand.
test('Im Haus einen neuen Link erzeugen: Er steht da und trägt, der alte ist tot', async ({
	page,
	baseURL
}) => {
	const adresseVorher = querySQL(
		`SELECT wert FROM system_einstellungen WHERE schluessel = 'oeffentliche_adresse'`
	);
	seedSQL(`INSERT INTO system_einstellungen (schluessel, wert)
		VALUES ('oeffentliche_adresse', '${baseURL}')
		ON CONFLICT (schluessel) DO UPDATE SET wert = EXCLUDED.wert;`);
	const alterToken = `E2E-ALT-${uniqueSuffix()}`;
	const s = seedBestellungMitLink(alterToken);
	try {
		await uiLogin(page);
		await menuepunkt(page, 'Bestellungen').click();
		await page.getByRole('tab', { name: 'Bestellhistorie', exact: true }).click();
		await page
			.getByRole('button', { name: new RegExp(`E2E-Naacher ${s}.*Bestellung öffnen`) })
			.click();
		await page.getByRole('button', { name: 'Neuen Link erzeugen' }).click();

		const feld = page.getByLabel('Bestätigungs-Link');
		await expect(feld).toHaveValue(new RegExp(`^${baseURL}/bestellung/.+`));
		await expect(page.getByText(/^Nur jetzt sichtbar/)).toBeVisible();
		const neuerPfad = new URL(await feld.inputValue()).pathname;

		await page.goto(`/bestellung/${alterToken}`);
		await expect(page.getByText('Dieser Link ist nicht mehr gültig')).toBeVisible();
		await page.goto(neuerPfad);
		await expect(page.getByRole('button', { name: 'Bestellung jetzt bestätigen' })).toBeVisible();
	} finally {
		seedSQL(
			adresseVorher === ''
				? `DELETE FROM system_einstellungen WHERE schluessel = 'oeffentliche_adresse';`
				: `UPDATE system_einstellungen SET wert = '${adresseVorher}'
				   WHERE schluessel = 'oeffentliche_adresse';`
		);
	}
});

test('Die Bestellung erscheint im Haus als bestätigt — ohne die Zeile aufzuklappen', async ({
	page
}) => {
	const token = `E2E-SICHT-${uniqueSuffix()}`;
	const s = seedBestellungMitLink(token);

	// Der Händler bestätigt über seinen Link.
	await page.goto(`/bestellung/${token}`);
	await page.getByRole('button', { name: 'Bestellung jetzt bestätigen' }).click();
	await expect(page.getByRole('heading', { name: 'Bestellung bestätigt' })).toBeVisible();

	// Und die Bibliothek sieht es in der Bestellhistorie, in der eingeklappten Zeile.
	await uiLogin(page);
	await menuepunkt(page, 'Bestellungen').click();
	await page.getByRole('tab', { name: 'Bestellhistorie', exact: true }).click();

	const zeile = page.locator('tbody tr', { hasText: `K-${s}` }).first();
	await expect(zeile).toContainText('Bestätigt');
	await expect(zeile.getByText('Wartet auf Händler')).toHaveCount(0);
});

test('Der Wareneingang nennt im Kopf der Bestellung, ob der Händler bestätigt hat', async ({
	page
}) => {
	const token = `E2E-WARE-${uniqueSuffix()}`;
	const s = seedBestellungMitLink(token);
	seedSQL(`UPDATE buecher_exemplare SET bestellstatus = 'im_zulauf'
	         WHERE barcode_id LIKE 'E2E-BC-${s}-%';`);

	try {
		await uiLogin(page);
		await menuepunkt(page, 'Bestellungen').click();
		await page.getByRole('tab', { name: /^Wareneingang/ }).click();
		const kopf = page.locator('[data-gruppe]', { hasText: `E2E-Naacher ${s}` });
		await expect(kopf.getByText('Wartet auf Händler')).toBeVisible();

		await page.goto(`/bestellung/${token}`);
		await page.getByRole('button', { name: 'Bestellung jetzt bestätigen' }).click();
		await expect(page.getByRole('heading', { name: 'Bestellung bestätigt' })).toBeVisible();

		// Die öffentliche Seite hat keine Navigation; die Sitzung trägt noch.
		await page.goto('/');
		await menuepunkt(page, 'Bestellungen').click();
		await page.getByRole('tab', { name: /^Wareneingang/ }).click();
		await expect(kopf.getByText(/Bestätigt · \d\d\.\d\d\./)).toBeVisible();
		await expect(kopf.getByText('Wartet auf Händler')).toHaveCount(0);
	} finally {
		// Der Zulauf ist eine gemeinsame Liste: Ohne das stünden die Exemplare dort weiter, nach dem
		// Teardown unter „Unbekannter Lieferant".
		seedSQL(`UPDATE buecher_exemplare SET bestellstatus = NULL
		         WHERE barcode_id LIKE 'E2E-BC-${s}-%';`);
	}
});

test('Eine in der Historie nachgetragene Zusage steht beim Wechsel des Reiters im Wareneingang', async ({
	page
}) => {
	const token = `E2E-NACH-${uniqueSuffix()}`;
	const s = seedBestellungMitLink(token);
	seedSQL(`UPDATE buecher_exemplare SET bestellstatus = 'im_zulauf'
	         WHERE barcode_id LIKE 'E2E-BC-${s}-%';`);

	try {
		await uiLogin(page);
		await menuepunkt(page, 'Bestellungen').click();
		await page.getByRole('tab', { name: /^Wareneingang/ }).click();
		const kopf = page.locator('[data-gruppe]', { hasText: `E2E-Naacher ${s}` });
		await expect(kopf.getByText('Wartet auf Händler')).toBeVisible();

		await page.getByRole('tab', { name: 'Bestellhistorie', exact: true }).click();
		await page
			.getByRole('button', { name: new RegExp(`E2E-Naacher ${s}.*Bestellung öffnen`) })
			.click();
		await page.getByRole('button', { name: 'Kleine Etiketten' }).click();
		await expect(page.getByText('Bestätigung in der Bibliothek nachgetragen')).toBeVisible();

		await page.getByRole('tab', { name: /^Wareneingang/ }).click();
		await expect(kopf.getByText(/Bestätigt · \d\d\.\d\d\./)).toBeVisible();
	} finally {
		seedSQL(`UPDATE buecher_exemplare SET bestellstatus = NULL
		         WHERE barcode_id LIKE 'E2E-BC-${s}-%';`);
	}
});

// Die Bestätigung kommt, während die Listen offen sind: Niemand fasst die Seite an, und sie wird
// auch nicht neu geladen. Am Go-Test allein sähe das niemand; dort fehlt der Weg bis in die Ansicht.

/** @param {string} s Kennung aus seedBestellungMitLink */
function legeInDenZulauf(s) {
	seedSQL(`UPDATE buecher_exemplare SET bestellstatus = 'im_zulauf'
	         WHERE barcode_id LIKE 'E2E-BC-${s}-%';`);
}

/** @param {string} s Kennung aus seedBestellungMitLink */
function nimmAusDemZulauf(s) {
	seedSQL(`UPDATE buecher_exemplare SET bestellstatus = NULL
	         WHERE barcode_id LIKE 'E2E-BC-${s}-%';`);
}

/**
 * Meldet an und öffnet den Reiter des Bestellwesens. Die Marke am Fenster überlebt kein Neuladen.
 * @param {import('@playwright/test').Page} seite
 * @param {{ name: string | RegExp, exact?: boolean }} reiter
 */
async function oeffneReiter(seite, reiter) {
	await uiLogin(seite);
	await menuepunkt(seite, 'Bestellungen').click();
	await seite.getByRole('tab', reiter).click();
	await seite.evaluate(() => {
		/** @type {any} */ (window).ohneNeuladen = true;
	});
}

/** @param {import('@playwright/test').Page} seite */
const nichtNeuGeladen = (seite) =>
	seite.evaluate(() => /** @type {any} */ (window).ohneNeuladen === true);

/**
 * Der Händler bestätigt in seinem eigenen Browser, ohne Anmeldung.
 * @param {import('@playwright/test').Browser} browser
 * @param {string} token
 */
async function haendlerBestaetigt(browser, token) {
	const kontext = await browser.newContext();
	const seite = await kontext.newPage();
	await seite.goto(`/bestellung/${token}`);
	await seite.getByRole('button', { name: 'Bestellung jetzt bestätigen' }).click();
	await expect(seite.getByRole('heading', { name: 'Bestellung bestätigt' })).toBeVisible();
	await kontext.close();
}

const BESTAETIGT = /Bestätigt · \d\d\.\d\d\./;
const HISTORIE = { name: 'Bestellhistorie', exact: true };
const WARENEINGANG = { name: /^Wareneingang/ };

/** @param {import('@playwright/test').Page} seite @param {string} s */
const zeileDerHistorie = (seite, s) => seite.locator('tbody tr', { hasText: `K-${s}` }).first();
/** @param {import('@playwright/test').Page} seite @param {string} s */
const kopfImWareneingang = (seite, s) =>
	seite.locator('[data-gruppe]', { hasText: `E2E-Naacher ${s}` });

test('Bestätigt der Händler, wechselt der Chip in Historie und Wareneingang von selbst', async ({
	page,
	browser
}) => {
	const token = `E2E-LIVE-${uniqueSuffix()}`;
	const s = seedBestellungMitLink(token);
	legeInDenZulauf(s);
	const zweiterPlatz = await (await browser.newContext()).newPage();

	try {
		await oeffneReiter(page, HISTORIE);
		await oeffneReiter(zweiterPlatz, WARENEINGANG);
		const zeile = zeileDerHistorie(page, s);
		const kopf = kopfImWareneingang(zweiterPlatz, s);
		await expect(zeile.getByText('Wartet auf Händler')).toBeVisible();
		await expect(kopf.getByText('Wartet auf Händler')).toBeVisible();

		await haendlerBestaetigt(browser, token);

		await expect(zeile.getByText(BESTAETIGT)).toBeVisible();
		await expect(kopf.getByText(BESTAETIGT)).toBeVisible();
		await expect(zeile.getByText('Wartet auf Händler')).toHaveCount(0);
		await expect(kopf.getByText('Wartet auf Händler')).toHaveCount(0);
		expect(await nichtNeuGeladen(page)).toBe(true);
		expect(await nichtNeuGeladen(zweiterPlatz)).toBe(true);
	} finally {
		nimmAusDemZulauf(s);
		await zweiterPlatz.context().close();
	}
});

test('Eine von Hand nachgetragene Zusage steht am zweiten Arbeitsplatz von selbst da', async ({
	page,
	browser
}) => {
	const token = `E2E-LIVE-${uniqueSuffix()}`;
	const s = seedBestellungMitLink(token);
	legeInDenZulauf(s);
	const zweiterPlatz = await (await browser.newContext()).newPage();

	try {
		await oeffneReiter(zweiterPlatz, WARENEINGANG);
		const kopf = kopfImWareneingang(zweiterPlatz, s);
		await expect(kopf.getByText('Wartet auf Händler')).toBeVisible();

		await oeffneReiter(page, HISTORIE);
		await page
			.getByRole('button', { name: new RegExp(`E2E-Naacher ${s}.*Bestellung öffnen`) })
			.click();
		await page.getByRole('button', { name: 'Kleine Etiketten' }).click();
		await expect(page.getByText('Bestätigung in der Bibliothek nachgetragen')).toBeVisible();

		await expect(kopf.getByText(BESTAETIGT)).toBeVisible();
		expect(await nichtNeuGeladen(zweiterPlatz)).toBe(true);
	} finally {
		nimmAusDemZulauf(s);
		await zweiterPlatz.context().close();
	}
});

test('Die offene Bestellung zeigt die Quittung von selbst, und der getippte Grund bleibt', async ({
	page,
	browser
}) => {
	const token = `E2E-LIVE-${uniqueSuffix()}`;
	const s = seedBestellungMitLink(token, { mittel: 'land' });

	await oeffneReiter(page, HISTORIE);
	await page
		.getByRole('button', { name: new RegExp(`E2E-Naacher ${s}.*Bestellung öffnen`) })
		.click();
	await expect(page.getByText('Warten auf den Händler')).toBeVisible();
	await page.getByRole('button', { name: 'Topf ändern' }).click();
	const grund = page.getByLabel(/Grund der Korrektur/);
	await grund.fill('falsch gekennzeichnet');

	await haendlerBestaetigt(browser, token);

	await expect(page.getByText('Der Lieferant hat die Bestellung bestätigt')).toBeAttached();
	await expect(grund).toHaveValue('falsch gekennzeichnet');
	expect(await nichtNeuGeladen(page)).toBe(true);
});

test('Im Wareneingang bleiben markierte Exemplare markiert, wenn der Händler bestätigt', async ({
	page,
	browser
}) => {
	const token = `E2E-LIVE-${uniqueSuffix()}`;
	const s = seedBestellungMitLink(token);
	legeInDenZulauf(s);

	try {
		await oeffneReiter(page, WARENEINGANG);
		const kopf = kopfImWareneingang(page, s);
		await expect(kopf.getByText('Wartet auf Händler')).toBeVisible();
		const kaestchen = page.getByRole('checkbox', { name: `E2E-Bestaetigung ${s} auswählen` });
		// Per Tastatur: Safari gibt einem Kästchen beim Anklicken keinen Fokus.
		await kaestchen.focus();
		await page.keyboard.press('Space');
		await expect(page.getByText('2 Exemplare markiert')).toBeVisible();

		await haendlerBestaetigt(browser, token);

		await expect(kopf.getByText(BESTAETIGT)).toBeVisible();
		await expect(page.getByText('2 Exemplare markiert')).toBeVisible();
		await expect(kaestchen).toBeChecked();
		await expect(kaestchen).toBeFocused();
	} finally {
		nimmAusDemZulauf(s);
	}
});

// Was während einer Unterbrechung geschah, kommt nicht über die Leitung nach. Die Fehlerantwort ist
// der harte Fall: Ein abgebrochener Abruf holt der Browser selbst nach (livesync-wiederaufbau.spec.js).
test('Nach einer Unterbrechung der Leitung holt das Fenster die Bestätigung nach', async ({
	page,
	browser
}) => {
	const token = `E2E-LIVE-${uniqueSuffix()}`;
	const s = seedBestellungMitLink(token);
	let leitungSteht = false;
	await page.route('**/events', async (route) => {
		if (leitungSteht) return route.continue();
		return route.fulfill({ status: 401, contentType: 'application/json', body: '{}' });
	});

	await oeffneReiter(page, HISTORIE);
	const zeile = zeileDerHistorie(page, s);
	await expect(zeile.getByText('Wartet auf Händler')).toBeVisible();

	await haendlerBestaetigt(browser, token);
	// Ohne Leitung erfährt dieses Fenster nichts.
	await expect(zeile.getByText('Wartet auf Händler')).toBeVisible();

	leitungSteht = true;

	await expect(zeile.getByText(BESTAETIGT)).toBeVisible({ timeout: 20_000 });
	expect(await nichtNeuGeladen(page)).toBe(true);
});
