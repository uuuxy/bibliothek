import { test, expect } from '@playwright/test';
import { uiLogin, apiPost, csrfToken, seedSQL, uniqueSuffix } from './helpers.js';

// Der geglückte Versand steht bewusst NICHT hier, sondern in api/mail_settings_test.go
// (Fake-SMTP-Server, der die Nachricht annimmt und sich befragen lässt): Die
// SMTP-Konfiguration des lokalen Stacks zeigt auf den Schulserver — ein E2E-Test des
// Versandwegs würde echte Mails verschicken. Was nur hier zu prüfen ist, ist der
// Weg durch Router und Berechtigungen; die Abweisungen lösen keine Verbindung aus.
//
// Vorher stand hier expect([200, 400, 500]).toContain(status) — eine Zusicherung, die
// nicht rot werden konnte: Der Aufruf schickte zudem Felder (test_recipient), die die
// API nie gelesen hat, lief also dauerhaft in die 400 und belegte nichts.

// Klein geschrieben passiert absichtlich: Go-Fehlerstrings beginnen laut Linter-Gate
// kleingeschrieben, und die Meldung wird unverändert zur Toast-Nachricht.
const EMPFAENGER_FELD = /empfänger/i;

// Diese Spec legt pro Lauf EINEN Benutzer an, und zwar unter eindeutigem Namen
// (uniqueSuffix), weil sich alle Specs eine Datenbank teilen. Weggeräumt hat ihn nie
// jemand: Am 12.08.2026 lagen 143 davon in der Entwicklungsdatenbank. Das ist nicht nur
// unordentlich — genau solche Altlasten haben heute schon einmal einen echten Fehler
// verdeckt, weil ein Testnutzer aus einem früheren Lauf noch existierte und eine Spec
// dadurch grün war, die in CI fiel (siehe betriebsbereitschaft.spec.js).
//
// Das Muster ist bewusst so eng gefasst, dass es nichts anderes treffen KANN: Präfix
// dieser Spec plus die Domain test.local. Ein zu weites Aufräumen ist hier schon einmal
// teuer gewesen — ein früherer Teardown nahm den Hauptlieferanten mit.
//
// afterAll statt afterEach, weil workers: 1 und fullyParallel: false gelten (siehe
// playwright.config.js): Es läuft nie ein zweiter Lauf daneben, dem wir die Zeile unter
// den Füßen wegziehen könnten. Der Lauf räumt damit auch die Altlasten mit auf.
test.afterAll(() => {
	seedSQL(`DELETE FROM benutzer WHERE email LIKE 'e2e-mailtest-%@test.local';`);
});

test('Test-Mail: unbrauchbarer Empfänger wird abgewiesen, ohne SMTP zu bemühen', async ({
	page
}) => {
	await uiLogin(page);

	// Leer: Das Formular lässt es nicht zu, ein API-Client schon.
	const leer = await apiPost(page, '/api/admin/settings/mail/test', { to: '' });
	expect(leer.status(), 'leerer Empfänger').toBe(400);
	expect((await leer.json()).error).toMatch(EMPFAENGER_FELD);

	// Keine Adresse: muss als Eingabefehler zurückkommen (400), nicht als
	// Serverstörung (500) — sonst kann das Formular es nicht von einem SMTP-Ausfall
	// unterscheiden, und apierrors dampft die Meldung auf "Datenbankfehler" ein.
	const kaputt = await apiPost(page, '/api/admin/settings/mail/test', { to: 'admin(at)schule.de' });
	expect(kaputt.status(), 'Adresse ohne @').toBe(400);
	expect((await kaputt.json()).error).toMatch(EMPFAENGER_FELD);

	// Kopfzeilen-Schmuggel: Ein CR/LF im Empfänger hängte sonst ein Bcc an die Mail.
	const schmuggel = await apiPost(page, '/api/admin/settings/mail/test', {
		to: 'admin@schule.de>\r\nBcc: mitleser@example.com'
	});
	expect(schmuggel.status(), 'Bcc-Schmuggel im Empfänger').toBe(400);
});

// Der Testversand verschickt Mail über den Schul-SMTP — er gehört keinem Konto in
// die Hand, das keine Benutzer verwalten darf. Die Berechtigung hängt am Router, ein
// Handler-Test in Go kann sie nicht belegen.
test('Test-Mail: ohne manage_users bleibt der Versandknopf verschlossen', async ({ page }) => {
	const mitarbeiter = `e2e-mailtest-${uniqueSuffix()}@test.local`;
	seedSQL(`
        INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv)
        VALUES ('E2E', 'Mailtest', '${mitarbeiter}', 'mitarbeiter', true)
        ON CONFLICT DO NOTHING;
    `);
	await uiLogin(page, mitarbeiter);

	const res = await apiPost(page, '/api/admin/settings/mail/test', { to: 'admin@schule.de' });
	expect(res.status(), 'Testversand als Mitarbeiter').toBe(403);
});

// Mahn-Template-Bearbeitung: der Admin muss Betreff/Text der Mahnungen ändern
// können (Roundtrip GET → PUT → GET). Der Originalzustand wird im finally
// wiederhergestellt — die Test-DB teilen sich alle Specs.
test('Mail-Templates: Mahnungs-Vorlage lässt sich ändern und speichern', async ({ page }) => {
	await uiLogin(page);
	const s = uniqueSuffix();

	const list = await page.request.get('/api/mail-templates');
	expect(list.status()).toBe(200);
	const templates = await list.json();
	const mahnung = templates.find((/** @type {any} */ t) => t.typ === 'MAHNUNG_ELTERN');
	expect(mahnung, 'Vorlage MAHNUNG_ELTERN muss existieren (Mahnwesen!)').toBeTruthy();

	const token = await csrfToken(page);
	try {
		const put = await page.request.put(`/api/mail-templates/${mahnung.id}`, {
			headers: { 'X-CSRF-Token': token },
			data: { betreff: `${mahnung.betreff} [E2E ${s}]`, text_body: mahnung.text_body }
		});
		expect(put.status()).toBe(200);

		const verify = await page.request.get('/api/mail-templates');
		const updated = (await verify.json()).find((/** @type {any} */ t) => t.id === mahnung.id);
		expect(updated.betreff).toContain(`[E2E ${s}]`);
		// Platzhalter-Variablen dürfen den Roundtrip nicht verlieren
		expect(updated.text_body).toContain('{{.Vorname}}');
	} finally {
		await page.request.put(`/api/mail-templates/${mahnung.id}`, {
			headers: { 'X-CSRF-Token': token },
			data: { betreff: mahnung.betreff, text_body: mahnung.text_body }
		});
	}
});
