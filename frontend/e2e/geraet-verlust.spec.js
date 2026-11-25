import { test, expect } from '@playwright/test';
import {
	uiLogin,
	apiPost,
	seedSQL,
	querySQL,
	uniqueSuffix,
	oeffneSchuelerProfil,
	raeumtNachJedemFallAuf
} from './helpers.js';

const angelegt = raeumtNachJedemFallAuf(test);
/** @type {string[]} Gerätenummern dieses Falls; das Gerät geht nach Ausleihe und Forderung. */
const geraete = [];
test.afterEach(() => {
	for (const nummer of geraete.splice(0))
		seedSQL(`DELETE FROM geraete WHERE barcode_id = '${nummer}';`);
});

// Ein Gerät, das nicht zurückkommt, ließ sich nicht aus der Ausleihe buchen. „Verlust oder
// Schaden melden" steht jetzt auch an der Gerätezeile der Leserakte: Die Ausleihe endet, ein
// verlorenes Gerät steht in der Geräteverwaltung als verloren, und die Forderung steht mit dem
// Modell unter „Gebühren & Schäden".
test('Gerät verloren: es bleibt als verloren stehen, die Forderung steht in der Akte', async ({
	page
}) => {
	await uiLogin(page);
	const suffix = uniqueSuffix();
	const created = await apiPost(page, '/api/schueler', {
		geburtsdatum: '2012-06-15',
		vorname: `Geraetprobe${suffix}`,
		nachname: 'Verlust',
		klasse: '9R',
		barcode_id: `S-${suffix}`
	});
	expect(created.ok(), `Schüler anlegen: ${created.status()}`).toBeTruthy();
	const { id: schuelerId } = await created.json();
	angelegt.leser.push(schuelerId);
	geraete.push(`G-${suffix}`);
	seedSQL(`
		WITH g AS (
			INSERT INTO geraete (modellname, barcode_id) VALUES ('E2E-Tablet-${suffix}', 'G-${suffix}') RETURNING id
		)
		INSERT INTO ausleihen (geraet_id, schueler_id, bearbeiter_id, ausgeliehen_am, rueckgabe_frist)
		SELECT g.id, '${schuelerId}', (SELECT id FROM benutzer ORDER BY erstellt_am LIMIT 1), NOW(),
		       NOW() + INTERVAL '14 days' FROM g;
	`);

	await oeffneSchuelerProfil(page, `Geraetprobe${suffix}`);
	await page.getByRole('button', { name: 'Verlust oder Schaden melden' }).click();
	const dialog = page.getByRole('dialog', { name: 'Verlust oder Schaden melden' });
	await expect(dialog).toContainText(`Gerät G-${suffix}`);
	await expect(dialog).toContainText('steht danach als verloren in der Geräteverwaltung');
	await dialog.locator('#damage-amount').fill('149');
	await dialog.getByRole('button', { name: 'Melden', exact: true }).click();
	await expect(page.getByText(/Verlust oder Schaden gebucht/)).toBeVisible();

	const stand = querySQL(`
		SELECT g.ist_ausgesondert, a.rueckgabe_am IS NOT NULL,
		       (SELECT count(*) FROM schadensfaelle f WHERE f.geraet_id = g.id AND f.betrag = 149)
		FROM geraete g JOIN ausleihen a ON a.geraet_id = g.id WHERE g.barcode_id = 'G-${suffix}'`);
	expect(stand, 'verloren | Ausleihe beendet | Forderung zu 149 €').toBe('t|t|1');
	// Die Forderung nennt das Modell, im Reiter „Gebühren & Schäden" der Akte.
	await page.getByRole('tab', { name: /Gebühren & Schäden/ }).click();
	await expect(page.getByText(`E2E-Tablet-${suffix}`).first()).toBeVisible();
});
