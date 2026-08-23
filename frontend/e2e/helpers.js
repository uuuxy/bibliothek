// Gemeinsame Helfer für die E2E-Smoke-Flows.

import { execSync } from 'node:child_process';

export const ADMIN_EMAIL = 'pflasch@philipp-reis-schule.de';
// Mock-IMAP (IMAP_HOST=mock im lokalen Stack) akzeptiert jedes Passwort.
export const ADMIN_PASSWORD = 'e2e-egal';

/**
 * Loggt über die echte Login-UI ein und wartet, bis die App steht.
 * (Die SPA macht beim Boot keinen Session-Restore — ein reines API-Login
 * ließe den Login-Screen stehen. Der Cookie-Jar wird mit page.request geteilt,
 * API-Seeding nach diesem Login ist also authentifiziert.)
 * @param {import('@playwright/test').Page} page
 * @param {string} [asEmail] anderer Benutzer (Mock-IMAP akzeptiert jedes Passwort)
 */
export async function uiLogin(page, asEmail = ADMIN_EMAIL) {
	await page.goto('/');

	const email = page.locator('#login-email');
	const password = page.locator('#login-password');

	// Erst fokussieren, dann füllen, dann verifizieren — die Svelte-Bindings
	// rendern kurz nach dem Mount; ungeduldiges fill landet sonst im falschen Feld.
	await email.click();
	await email.fill(asEmail);
	await password.click();
	await password.fill(ADMIN_PASSWORD);

	const { expect } = await import('@playwright/test');
	await expect(email).toHaveValue(asEmail);
	await expect(password).toHaveValue(ADMIN_PASSWORD);

	await page.getByRole('button', { name: 'Anmelden' }).click();
	await page.getByRole('button', { name: 'Abmelden' }).waitFor();
}

/**
 * Holt das CSRF-Token (Double-Submit-Cookie-Pattern) für schreibende API-Calls.
 * @param {import('@playwright/test').Page} page
 */
export async function csrfToken(page) {
	const res = await page.request.get('/api/csrf-token');
	const body = await res.json();
	return body.csrf_token;
}

/**
 * Schreibender API-Call mit CSRF-Header — für Test-Seeding (Schüler, Lieferanten …).
 * @param {import('@playwright/test').Page} page
 * @param {string} url
 * @param {object} data
 */
export async function apiPost(page, url, data) {
	const token = await csrfToken(page);
	return page.request.post(url, {
		data,
		headers: { 'X-CSRF-Token': token }
	});
}

/**
 * Schreibender API-Call (PATCH) mit CSRF-Header.
 * @param {import('@playwright/test').Page} page
 * @param {string} url
 * @param {object} data
 */
export async function apiPatch(page, url, data) {
	const token = await csrfToken(page);
	return page.request.patch(url, {
		data,
		headers: { 'X-CSRF-Token': token }
	});
}

/**
 * Seedet Testdaten direkt in die lokale Stack-DB (docker-compose.local.yml).
 * Nur für Entitäten ohne einfachen API-Weg (z. B. Buch-Exemplare).
 * @param {string} sql
 */
export function seedSQL(sql) {
	const container = process.env.E2E_DB_CONTAINER || 'bibliothek-db-local';
	execSync(`docker exec -i ${container} psql -U postgres -d bibliothek -v ON_ERROR_STOP=1`, {
		input: sql
	});
}

/**
 * Legt einen Anmelde-Benutzer an — idempotent, für Specs, die eine bestimmte Rolle brauchen.
 *
 * Warum als gemeinsamer Helfer (12.08.2026): `betriebsbereitschaft.spec.js` meldete sich
 * als `e2e-helfer@test.local` an, ohne ihn anzulegen — erzeugt wurde er von
 * `helfer-kiosk.spec.js`. Alphabetisch läuft „betriebsbereitschaft" aber ZUERST. Auf einer
 * gebrauchten Entwicklungsdatenbank fiel das nie auf, weil der Benutzer von früheren
 * Läufen herumlag; in CI ist die Datenbank frisch, dort scheiterte die Anmeldung, und der
 * Test lief 30 s in den Timeout. Die e2e-Stufe war deshalb dauerhaft rot.
 *
 * Eine Spec, die einen Zustand braucht, stellt ihn selbst her.
 *
 * @param {string} email  Anmeldeadresse (Mock-IMAP nimmt jedes Passwort)
 * @param {string} rolle  benutzer_rolle: admin | kollegium | mitarbeiter | helfer
 */
export function seedBenutzer(email, rolle) {
	seedSQL(`
		INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv)
		VALUES ('E2E', '${rolle}', '${email}', '${rolle}', true)
		ON CONFLICT DO NOTHING;
	`);
}

/**
 * Liest einen Wert aus der Test-DB (für Zustands-Asserts nach UI-Aktionen).
 * @param {string} sql  SELECT-Statement
 * @returns {string} getrimmte Ausgabe (tuples-only)
 */
export function querySQL(sql) {
	const container = process.env.E2E_DB_CONTAINER || 'bibliothek-db-local';
	return execSync(
		`docker exec -i ${container} psql -U postgres -d bibliothek -tA -v ON_ERROR_STOP=1`,
		{
			input: sql
		}
	)
		.toString()
		.trim();
}

/**
 * Stellt her, dass die Bestellbedarfs-Liste gefüllt ist — Voraussetzung für jeden
 * Test, der im Bestell-Bildschirm auf „+ zur Bestellung hinzufügen" klickt.
 *
 * Warum das nötig ist: Die Liste hing an ZWEI Bedingungen, die beide außerhalb des
 * Tests lagen. `bestellbedarf_warnung_aktiv` ist ein Schalter in den Einstellungen —
 * steht er aus, liefert /api/bestellungen eine LEERE Liste, ganz gleich wie der
 * Bestand aussieht. Genau das war am 04.08.2026 der Fall: Zwei Specs waren rot, ohne
 * dass am Code etwas fehlte, und die Oberfläche meldete dabei „Kein Titel liegt unter
 * der Bestellbedarf-Schwelle" — was in die falsche Richtung zeigt.
 *
 * Deshalb wird hier beides gesetzt: der Schalter an UND eigene Titel angelegt. Der
 * Bestand der Datenbank ist damit egal.
 *
 * Ein Titel zählt als Bedarf, wenn er LMF ist (`^lmf[ -]`, siehe pkg/lmf) und weniger
 * nicht ausgesonderte Exemplare hat als die Schwelle. Die angelegten Titel haben gar
 * keine — sie stehen damit zugleich ganz oben, weil die Liste nach Fehlbestand sortiert.
 *
 * @param {number} [anzahl] wie viele Titel sicher in der Liste stehen sollen
 * @returns {() => void} Aufräumen: Titel weg, Schalter zurück auf den alten Wert
 */
export function seedBestellbedarf(anzahl = 8) {
	const marke = `E2E-Bedarf-${uniqueSuffix()}`;
	const schalterVorher = querySQL(
		`SELECT wert FROM system_einstellungen WHERE schluessel = 'bestellbedarf_warnung_aktiv'`
	);

	seedSQL(`
		INSERT INTO system_einstellungen (schluessel, wert)
		VALUES ('bestellbedarf_warnung_aktiv', 'true')
		ON CONFLICT (schluessel) DO UPDATE SET wert = 'true';

		INSERT INTO buecher_titel (titel, autor, meldebestand)
		SELECT 'LMF-${marke} ' || g, 'E2E', 30 FROM generate_series(1, ${anzahl}) g;
	`);

	return () => {
		seedSQL(`
			DELETE FROM buecher_titel WHERE titel LIKE 'LMF-${marke} %';
			${
				schalterVorher
					? `UPDATE system_einstellungen SET wert = '${schalterVorher}'
					   WHERE schluessel = 'bestellbedarf_warnung_aktiv';`
					: `DELETE FROM system_einstellungen WHERE schluessel = 'bestellbedarf_warnung_aktiv';`
			}
		`);
	};
}

/** 8×8-PNG, damit ein Test ein ECHTES Bild bekommt. Die Größe ist der Punkt: Der
 *  Cover-Proxy liefert bei jedem Fehler ein 1×1-GIF mit Status 200 aus, ein Test gegen
 *  "Bild geladen" ginge daran vorbei. Nur naturalWidth > 1 trennt beides. */
const TESTBILD_PNG_BASE64 =
	'iVBORw0KGgoAAAANSUhEUgAAAAgAAAAICAIAAABLbSncAAAAEUlEQVR4nGM4EaCBFTEMLQkAaplQAc/OcKAAAAAASUVORK5CYII=';

/**
 * Legt ein echtes Bild im uploads-Verzeichnis des Backends ab und gibt dessen
 * öffentlichen Pfad zurück — so, wie der Cover-Sync ihn in buecher_titel.cover_url
 * schreibt (lokal, nicht extern).
 * @param {string} name Dateiname ohne Endung
 * @returns {string} Pfad für cover_url, z. B. "/uploads/e2e-cover-abc.png"
 */
export function seedCoverDatei(name) {
	const container = process.env.E2E_BACKEND_CONTAINER || 'bibliothek-backend-local';
	const datei = `${name}.png`;
	execSync(`docker exec -i ${container} sh -c 'base64 -d > /app/uploads/${datei}'`, {
		input: TESTBILD_PNG_BASE64
	});
	return `/uploads/${datei}`;
}

/**
 * Eindeutiger Suffix, damit Läufe auf derselben DB nicht kollidieren.
 *
 * randomUUID statt Math.random(): Playwright führt Specs parallel aus, und
 * `Math.random() * 1000` lieferte nur 1000 Werte — zwei Specs in derselben
 * Millisekunde kollidierten mit 1:1000. Das ist selten genug, um als unerklärlicher
 * Flake durchzugehen, und häufig genug, um irgendwann aufzutreten. randomUUID ist
 * hier nicht wegen Kryptografie richtig, sondern weil es das Versprechen im Namen
 * dieser Funktion tatsächlich einlöst.
 */
export function uniqueSuffix() {
	return `${Date.now().toString(36)}${crypto.randomUUID().slice(0, 8)}`;
}

/**
 * Navigiert und stellt sicher, dass die Anwendung wirklich DORT gelandet ist.
 *
 * `page.goto()` allein reicht nicht: Auf einen unbekannten Pfad antwortet diese SPA nicht
 * mit einem Fehler, sondern schiebt still auf /kiosk. Ein Gate, das eine Liste von
 * Bildschirmen abklappert, misst dann den Kiosk mehrfach und bleibt grün — es hat den
 * gemeinten Bildschirm nur nie gesehen.
 *
 * Genau das war am 10.08.2026 der Fall. Der Rollen-Umbau (15d2806) benannte den Tab von
 * `lehrer` in `kollegium` um, der Router führt seither `/kollegium-portal`. Die beiden
 * Gates control-hoehen.spec.js und icon-trefferflaechen.spec.js besuchten weiter
 * `/lehrer-portal` und maßen ab da den Kiosk statt des Portals — ohne dass irgendetwas
 * rot wurde.
 *
 * @param {import('@playwright/test').Page} page
 * @param {string} pfad
 */
export async function gehZu(page, pfad) {
	const { expect } = await import('@playwright/test');
	await page.goto(pfad);
	await expect(
		page,
		`„${pfad}" hat nicht gehalten — die Anwendung ist woandershin gesprungen. ` +
			`Unbekannte Pfade landen still auf /kiosk; vermutlich wurde eine Route umbenannt.`
	).toHaveURL(new RegExp(`${pfad.replace(/[.*+?^${}()|[\]\\]/g, String.raw`\$&`)}$`));
}

/**
 * Der Eintrag einer Einstellungs-Kategorie in der Liste links.
 *
 * Auf die Liste eingegrenzt, nicht auf die Seite: Der Speichern-Knopf der geöffneten
 * Kategorie heißt „<Name> speichern" und passt auf dieselbe Namenssuche. Wer nur
 * `getByRole('button', { name: /^Datenschutz/ })` schreibt, trifft je nachdem, welche
 * Kategorie gerade offen ist — mal den Listeneintrag, mal den Knopf.
 *
 * @param {import('@playwright/test').Page} page
 * @param {string} titel Titel der Kategorie, z. B. 'Datenverwaltung'.
 */
export function einstellungsKategorie(page, titel) {
	return page.getByRole('navigation', { name: 'Einstellungs-Kategorien' }).getByRole('button', {
		// Anker auf Titel PLUS Leerzeichen: Der zugängliche Name eines Eintrags ist
		// „<Titel> <Beitext>". Ohne das Leerzeichen trifft „Mahnwesen" auch
		// „Mahnwesen-Routing" — Playwright bricht dann mit strict mode ab, oder klickt
		// in einer künftigen Liste stillschweigend die falsche Kategorie.
		name: new RegExp(`^${titel.replace(/[.*+?^${}()|[\]\\]/g, String.raw`\$&`)} `)
	});
}
