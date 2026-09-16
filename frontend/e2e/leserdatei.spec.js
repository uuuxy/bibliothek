import { test, expect } from '@playwright/test';
import { uiLogin, apiPost, seedSQL, uniqueSuffix, gehZu } from './helpers.js';

// Der Nachweis für den Schritt „Theke und Leserdatei" (docs/OFFEN.md 5.16), im Browser
// und gegen den frisch gebauten Stack.
//
// Die Frage, um die es geht: Ein Kollege lässt sich an der Theke über seinen
// Ausweis laden — aber sieht man auch, welche Bücher er hat? Und findet man ihn, wenn er
// die Karte nicht dabei hat? Beides war bis zum 16.09.2026 nein.

test('Leserdatei: ein Kollege steht in der Liste, hat eine Akte und ist über den Namen zu finden', async ({
	page
}) => {
	const s = uniqueSuffix();
	const nachname = `Kollegin${s}`;

	// Über die Tür anlegen, die die Leserdatei selbst benutzt: Art zuerst, keine Klasse,
	// kein Geburtsdatum.
	//
	// Die Schul-E-Mail ist seit dem 16.09.2026 PFLICHT (api/student_create.go): Ohne sie
	// entsteht kein Konto, und ohne Konto stünde die Person zweimal in der Leserdatei,
	// sobald sie sich selbst anmeldet. Die Domain muss zu SELBSTANMELDUNG_DOMAIN passen —
	// im lokalen Stack `test.local` (docker-compose.local.yml).
	await uiLogin(page);
	const angelegt = await apiPost(page, '/api/schueler', {
		art: 'lehrkraft',
		vorname: 'Katrin',
		nachname,
		email: `katrin.${nachname.toLowerCase()}@test.local`
	});
	expect(angelegt.ok(), `Anlegen fehlgeschlagen: ${angelegt.status()}`).toBeTruthy();
	const { id, barcode_id: ausweis } = await angelegt.json();
	expect(ausweis, 'ohne Ausweisnummer ließe sich kein Ausweis drucken').toBeTruthy();

	// ── 1. Die Liste ────────────────────────────────────────────────────────────
	await page.getByTitle('Leserdatei').click();
	const suche = page.getByLabel('Leser suchen');
	await suche.click();
	await suche.fill(nachname);

	const zeile = page.getByRole('row', { name: new RegExp(nachname) });
	await expect(zeile).toBeVisible();
	// Die Spalte Art ist der Unterschied zur Schülerdatei: Ohne sie stünde die Kollegin
	// hier wie eine Schülerin, bei der die Klasse fehlt.
	await expect(zeile.getByText('Lehrkraft')).toBeVisible();

	// ── 2. Die Akte ─────────────────────────────────────────────────────────────
	await zeile.getByRole('button', { name: new RegExp(`Profil von Katrin ${nachname}`) }).click();
	await expect(page.getByRole('heading', { name: `Katrin ${nachname}` })).toBeVisible();

	// Der Reiter heisst seit dem 16.09.2026 fuer JEDEN „Stammdaten & Adresse" — eine Akte,
	// ein Name. Vorher stand ueber derselben Sache beim Kollegium „Persoenliche Daten".
	await page.getByRole('button', { name: 'Stammdaten & Adresse' }).click();
	const akte = page.locator('main');
	await expect(akte.getByText(ausweis).first()).toBeVisible();
	// Dieselben Angaben wie bei einer Schuelerin — die Akte hat nur noch EINE Form
	// (Absprache: „warum eine andere maske als bei schuelern?"). Die Postanschrift gehoert
	// ausdruecklich dazu: An ihr haengen Mahnung und Bescheid.
	await expect(akte.getByText('Postanschrift')).toBeVisible();
	await expect(akte.getByText('Art', { exact: true })).toBeVisible();
	// Was am KONTO haengt, steht weiterhin nicht hier.
	await expect(akte.getByText(/Benutzer & Rechte/)).toBeVisible();

	// ── 3. Die Theke: über den NAMEN, nicht über den Ausweis ────────────────────
	// Ein Buch auf ihren Namen, damit die Akte an der Theke etwas zu zeigen hat.
	seedSQL(`
		INSERT INTO buecher_titel (titel, autor, ist_lernmittel)
		VALUES ('Leserdateibuch ${s}', 'Testautor', false);
		INSERT INTO buecher_exemplare (barcode_id, titel_id)
		SELECT 'B-LD-${s}', id FROM buecher_titel WHERE titel = 'Leserdateibuch ${s}';
		INSERT INTO ausleihen (exemplar_id, schueler_id, rueckgabe_frist)
		SELECT e.id, '${id}', CURRENT_TIMESTAMP + INTERVAL '365 days'
		FROM buecher_exemplare e WHERE e.barcode_id = 'B-LD-${s}';
	`);

	await gehZu(page, '/kiosk');
	const scanfeld = page.getByPlaceholder(/scannen/i).first();
	await scanfeld.click();
	await scanfeld.fill(nachname);

	// Die Trefferliste nennt die Gruppe „Leser" und die Art am Treffer.
	const treffer = page.getByRole('option', { name: new RegExp(`Lehrkraft: Katrin ${nachname}`) });
	await expect(treffer).toBeVisible();
	await treffer.click();

	// BEWEIS: Nicht nur eine schmale Karte — ihre AKTE mit dem Buch, das sie hat.
	await expect(page.getByText(`Leserdateibuch ${s}`).first()).toBeVisible();
	// Und der Hinweis sagt, was an ihr anders ist.
	await expect(page.getByText(/Frist ein Jahr/)).toBeVisible();
});
