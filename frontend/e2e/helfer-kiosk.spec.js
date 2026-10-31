import { test, expect } from '@playwright/test';
import {
	uiLogin,
	apiPost,
	seedBenutzer,
	querySQL,
	raeumtNachJedemFallAuf,
	scanneWieScanner,
	uniqueSuffix
} from './helpers.js';
import { sammleStoerungen } from './rundgang-hilfe.js';

// Die Rolle "helfer" war lange unerreichbar: Router und Rechtesystem kannten sie,
// das benutzer_rolle-ENUM aber nicht — sie liess sich niemand zuweisen (behoben mit
// Migration 042). Damit hatte der Kiosk-Modus für Hilfskräfte nie einen Test gesehen.
//
// Diese Spec sichert das ab, was die Rolle ausmacht: Sie ist vergebbar, sie führt in
// den Kiosk, sie bucht dort, und sie kommt an nichts anderes heran. Ab Werk hält sie
// perform_actions und view_books (db/rechte_vorgabe.go); die Grenze zu den Akten zieht
// view_students (docs/datenschutz/nachweis.md, Abschnitt 3).

const HELFER_EMAIL = 'e2e-helfer@test.local';

function seedHelfer() {
	seedBenutzer(HELFER_EMAIL, 'helfer');
}

const angelegt = raeumtNachJedemFallAuf(test);

/** Ein Schüler und ein freies Lernmittel: Bei dem hält keine Regel der Bücherei die Ausleihe an. */
function seedLeserUndBuch() {
	const s = uniqueSuffix().toUpperCase();
	const ausweis = `S-HK${s}`;
	const buch = `B-HK${s}`;
	const titel = `E2E-Helfer ${s}`;
	const nachname = `Theke${s}`;
	angelegt.titel.push(titel);
	angelegt.leser.push(
		querySQL(`INSERT INTO leser (barcode_id, vorname, nachname, klasse, art, abgaenger_jahr)
			VALUES ('${ausweis}', 'Hanna', '${nachname}', '07A', 'schueler', 2031) RETURNING id;`)
			.split('\n')[0]
			.trim()
	);
	const titelId = querySQL(`WITH t AS (
			INSERT INTO buecher_titel (titel, ist_lernmittel) VALUES ('${titel}', true) RETURNING id
		), e AS (
			INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar)
			SELECT id, '${buch}', true FROM t
		)
		SELECT id FROM t;`);
	return { ausweis, buch, nachname, titelId };
}

/** Bei wem das Exemplar offen verliehen ist (Ausweisnummer). @param {string} buch */
function verliehenAn(buch) {
	return querySQL(`SELECT coalesce(string_agg(l.barcode_id, ','), '')
		FROM ausleihen a JOIN buecher_exemplare e ON e.id = a.exemplar_id
		JOIN leser l ON l.id = a.schueler_id
		WHERE e.barcode_id = '${buch}' AND a.rueckgabe_am IS NULL;`);
}

test('Helfer: Rolle ist vergebbar und landet im Kiosk', async ({ page }) => {
	// Das INSERT selbst ist der Test für Migration 042: Ohne den ENUM-Wert schlägt
	// bereits das Seeding fehl.
	seedHelfer();

	await uiLogin(page, HELFER_EMAIL);

	// Router.svelte zwingt Helfer auf den Kiosk — unabhängig davon, wo sie landen.
	await expect(page).toHaveURL(/\/(kiosk)?$/);

	// Die Omnibox ist die Kiosk-Oberfläche.
	await expect(page.getByRole('button', { name: 'Abmelden' })).toBeVisible();
});

test('Helfer: kommt weder per UI noch per API an fremde Bereiche', async ({ page }) => {
	seedHelfer();
	await uiLogin(page, HELFER_EMAIL);

	// Server blockt die Bereiche, die dem Helfer nicht gehören.
	const abgaenger = await page.request.get('/api/abgaenger');
	expect(abgaenger.status(), 'Abgänger-API für Helfer').toBe(403);

	const backup = await page.request.get('/api/admin/system/backup-status');
	expect(backup.status(), 'Backup-Status für Helfer').toBe(403);

	const createUser = await apiPost(page, '/api/benutzer', {
		vorname: 'Boese',
		nachname: 'Absicht',
		email: 'helfer-eskalation@test.local',
		rolle: 'admin'
	});
	expect(createUser.status(), 'Benutzer anlegen als Helfer').toBe(403);

	// Direkter URL-Aufruf umgeht die Router-Weiche nicht: keine Daten, kein Absturz.
	await page.goto('/schuelerdatei');
	await expect(page.getByText('Barcode-ID')).toHaveCount(0);
});

// Der Scan eines Ausweises zeigt, was die Antwort der Theke trägt: Name, Klasse, Nummer, Sperre.
// Die Akte steht hinter view_students; danach zu fragen, zeigte bei jedem Scan eine rote Meldung,
// obwohl die Buchung lief.
test('Helfer: Ein Scan zeigt Name und Klasse ohne Ablehnung, und das Buch ist gebucht', async ({
	page
}) => {
	const { ausweis, buch, nachname } = seedLeserUndBuch();
	seedHelfer();
	const stoerungen = sammleStoerungen(page, () => 'Theke');
	await uiLogin(page, HELFER_EMAIL);
	await expect(page.getByPlaceholder(/scannen/i).first()).toBeVisible();

	await scanneWieScanner(page, ausweis);
	await expect(page.getByRole('heading', { name: `Hanna ${nachname}` })).toBeVisible();
	await expect(page.getByText('Klasse 07A')).toBeVisible();
	await expect(page.getByText(ausweis, { exact: true })).toBeVisible();
	await expect(page.getByText('Akte nicht geladen')).toHaveCount(0);
	// Was die Theke nicht weiß, behauptet sie nicht: „keine Forderungen" wäre ungeprüft.
	await expect(page.getByText('Offene Forderungen')).toHaveCount(0);
	await expect(page.getByText(/^Abgang/)).toHaveCount(0);
	await expect(page.getByRole('tab')).toHaveCount(0);

	await scanneWieScanner(page, buch);
	await expect.poll(() => verliehenAn(buch)).toBe(ausweis);

	await scanneWieScanner(page, buch);
	await expect.poll(() => verliehenAn(buch)).toBe('');

	await page.getByRole('button', { name: 'Leser schließen' }).click();
	await expect(page.getByRole('heading', { name: `Hanna ${nachname}` })).toHaveCount(0);
	expect(stoerungen).toEqual([]);
});

// Ausleiher und Historie nennen Personen, die Warteliste hängt an einem eigenen Recht. Der Helfer
// beantwortet „habt ihr Band 3 noch da?" mit den Exemplaren; nach dem Rest fragt die Akte nicht.
test('Helfer: Die Buchakte zeigt die Exemplare und fragt nicht nach Personen', async ({ page }) => {
	const { buch, titelId } = seedLeserUndBuch();
	seedHelfer();
	const stoerungen = sammleStoerungen(page, () => 'Buchakte');
	await uiLogin(page, HELFER_EMAIL);

	await page.goto(`/medienkatalog/buch/${titelId}`);
	await expect(page.getByText(buch)).toBeVisible();
	await expect(page.getByText(/Nicht geladen/)).toHaveCount(0);
	await expect(page.getByText('Ausleiher', { exact: true })).toHaveCount(0);
	await expect(page.getByRole('tab')).toHaveCount(0);
	expect(stoerungen).toEqual([]);
});
