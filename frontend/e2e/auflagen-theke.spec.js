import { test, expect } from '@playwright/test';
import { uiLogin, seedSQL, uniqueSuffix, menuepunkt } from './helpers.js';

// Gemischte Auflagen an der Theke (docs/OFFEN.md 4.18, Stufe 5): Ein Kind der 7B hat die
// 3. Auflage, ein zweites bekommt die 4. Auflage desselben Buchs. Die Ausleihe geht durch, und
// über dem Konto steht die Zeile, welche Auflage die Klasse schon hat — entschieden am
// 25.09.2026: wie die Fremdrückgabe, kein Dialog.
test('Theke: Hinweis, wenn eine Klasse gemischte Auflagen bekommt', async ({ page }) => {
	const s = uniqueSuffix();
	seedSQL(`
		WITH w AS (INSERT INTO werke DEFAULT VALUES RETURNING id),
		alt AS (INSERT INTO buecher_titel (titel, auflage, erscheinungsjahr, ist_lernmittel, werk_id)
			SELECT 'E2E-Auflagen-Theke ${s}', '3. Aufl.', 2019, true, id FROM w RETURNING id),
		neu AS (INSERT INTO buecher_titel (titel, auflage, erscheinungsjahr, ist_lernmittel, werk_id)
			SELECT 'E2E-Auflagen-Theke ${s}', '4. Aufl.', 2023, true, id FROM w RETURNING id),
		ex AS (INSERT INTO buecher_exemplare (titel_id, barcode_id)
			SELECT id, 'B-AUFA${s}' FROM alt UNION ALL SELECT id, 'B-AUFN${s}' FROM neu
			RETURNING id, barcode_id),
		ida AS (INSERT INTO schueler (vorname, nachname, klasse, barcode_id, abgaenger_jahr)
			VALUES ('Ida', 'Auflage-${s}', '7B', 'S-AUFI${s}', 2031) RETURNING id)
		INSERT INTO ausleihen (exemplar_id, schueler_id, rueckgabe_frist)
		SELECT ex.id, ida.id, CURRENT_DATE + 300 FROM ex, ida WHERE ex.barcode_id = 'B-AUFA${s}';

		INSERT INTO schueler (vorname, nachname, klasse, barcode_id, abgaenger_jahr)
		VALUES ('Ben', 'Auflage-${s}', '7B', 'S-AUFB${s}', 2031);
	`);

	try {
		await uiLogin(page);
		await menuepunkt(page, 'Ausleihe').click();
		const scan = page.getByPlaceholder(/scannen/i).first();
		await scan.fill(`S-AUFB${s}`);
		await scan.press('Enter');
		await expect(page.getByText(`Auflage-${s}`).first()).toBeVisible();

		await scan.fill(`B-AUFN${s}`);
		await scan.press('Enter');
		await expect(page.getByText(`„E2E-Auflagen-Theke ${s}" ausgeliehen an Ben`)).toBeVisible();
		await expect(page.getByRole('status').filter({ hasText: 'andere Auflage' })).toHaveText(
			`„E2E-Auflagen-Theke ${s}" ist 4. Aufl. · 2023, eine andere Auflage als in der 07B: 1 Kind hat 3. Aufl. · 2019`
		);
	} finally {
		seedSQL(`
			DELETE FROM ausleihen WHERE exemplar_id IN (
				SELECT id FROM buecher_exemplare WHERE barcode_id IN ('B-AUFA${s}', 'B-AUFN${s}'));
			DELETE FROM werke WHERE id IN (
				SELECT werk_id FROM buecher_titel WHERE titel = 'E2E-Auflagen-Theke ${s}');
			DELETE FROM buecher_titel WHERE titel = 'E2E-Auflagen-Theke ${s}';
			DELETE FROM leser WHERE barcode_id IN ('S-AUFI${s}', 'S-AUFB${s}');
		`);
	}
});

// Der Bücherstapel: Fünf Kinder einer Klasse bekommen Mathe 8, das sechste Mathe 7, das sonst
// niemand hat. Die Zeile steht über dem Konto und übersteht die nächsten Scans; ein zweites Mathe 8
// bringt die zweite Zeile, zurückgenommen fällt sie weg, und mit dem nächsten Kind sind beide fort.
test('Theke: Hinweise beim Bücherstapel stehen, bis das nächste Kind kommt', async ({ page }) => {
	const s = uniqueSuffix();
	// Eine Klasse in der Form der Schule, damit die Datenbank sie in ihre Anzeigeform bringt
	// (klassen_anzeigeform: Jahrgang zweistellig, Rest groß); die Ziffern machen sie je Lauf eindeutig.
	const nr = String(parseInt(s.slice(-6), 16) % 1000000).padStart(6, '0');
	const klasse = `8q${nr}`;
	const namen = ['Ada', 'Ben', 'Cem', 'Dora', 'Emil', 'Fia', 'Gus', 'Hen'];
	const mathe8 = `E2E-Stapel-Mathe8 ${s}`;
	const mathe7 = `E2E-Stapel-Mathe7 ${s}`;
	const kinder = namen
		.map((v, i) => `('${v}', 'Stapel${i}-${s}', '${klasse}', 'S-STP${i}${s}', 2031)`)
		.join(', ');
	seedSQL(`
		WITH m8 AS (INSERT INTO buecher_titel (titel, ist_lernmittel) VALUES ('${mathe8}', true) RETURNING id),
		m7 AS (INSERT INTO buecher_titel (titel, ist_lernmittel) VALUES ('${mathe7}', true) RETURNING id)
		INSERT INTO buecher_exemplare (titel_id, barcode_id)
		SELECT m8.id, 'B-STP8-' || i || '${s}' FROM m8, generate_series(0, 6) AS i
		UNION ALL SELECT m7.id, 'B-STP7${s}' FROM m7;

		INSERT INTO schueler (vorname, nachname, klasse, barcode_id, abgaenger_jahr) VALUES ${kinder};
	`);

	try {
		await uiLogin(page);
		await menuepunkt(page, 'Ausleihe').click();
		const scan = page.getByPlaceholder(/scannen/i).first();
		/** @param {string} nummer */
		const scanne = async (nummer) => {
			await scan.fill(nummer);
			await scan.press('Enter');
		};
		/** @param {number} i */
		const ladeKind = async (i) => {
			await scanne(`S-STP${i}${s}`);
			await expect(page.getByText(`Stapel${i}-${s}`).first()).toBeVisible();
		};
		const fremdZeile = page.getByRole('status').filter({ hasText: 'sonst niemand' });
		const doppeltZeile = page.getByRole('status').filter({ hasText: 'jetzt zweimal' });

		for (let i = 0; i < 5; i++) {
			await ladeKind(i);
			await scanne(`B-STP8-${i}${s}`);
			await expect(page.getByText(`„${mathe8}" ausgeliehen an ${namen[i]}`)).toBeVisible();
		}
		await expect(fremdZeile).toHaveCount(0);

		// Fia bekommt Mathe 7: Mit ihr haben sechs von acht ihre Bücher, und keines hat Mathe 7.
		await ladeKind(5);
		await scanne(`B-STP7${s}`);
		const fremd = `„${mathe7}" hat sonst niemand in der 08Q${nr}. 6 von 8 Kindern haben ihre Schulbücher schon.`;
		await expect(fremdZeile).toHaveText(fremd);

		// Der nächste Scan nimmt die Zeile nicht weg: Fia bekommt Mathe 8 wie alle.
		await scanne(`B-STP8-5${s}`);
		await expect(page.getByText(`„${mathe8}" ausgeliehen an Fia`)).toBeVisible();
		await expect(fremdZeile).toHaveText(fremd);
		await expect(doppeltZeile).toHaveCount(0);

		// Ein zweites Mathe 8 bringt die zweite Zeile; die erste bleibt.
		await scanne(`B-STP8-6${s}`);
		await expect(doppeltZeile).toHaveText(`Fia hat „${mathe8}" jetzt zweimal`);
		await expect(fremdZeile).toHaveText(fremd);

		// Zurückgenommen (erneuter Scan): Die Zeile dieses Buchs fällt weg, die andere bleibt.
		await scanne(`B-STP8-6${s}`);
		await expect(page.getByText(`„${mathe8}" erfolgreich zurückgegeben`)).toBeVisible();
		await expect(doppeltZeile).toHaveCount(0);
		await expect(fremdZeile).toHaveText(fremd);

		// Das nächste Kind: Die Zeilen gehören zu Fia und gehen mit ihr.
		await ladeKind(6);
		await expect(fremdZeile).toHaveCount(0);
	} finally {
		seedSQL(`
			DELETE FROM ausleihen WHERE exemplar_id IN (
				SELECT e.id FROM buecher_exemplare e JOIN buecher_titel t ON t.id = e.titel_id
				WHERE t.titel IN ('${mathe8}', '${mathe7}'));
			DELETE FROM buecher_titel WHERE titel IN ('${mathe8}', '${mathe7}');
			DELETE FROM leser WHERE barcode_id IN (${namen.map((_, i) => `'S-STP${i}${s}'`).join(', ')});
		`);
	}
});
