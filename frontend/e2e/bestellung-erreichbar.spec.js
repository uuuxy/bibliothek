// Gate gegen die abgeschnittene Bestellung: Summe und „Bestellung auslösen" stehen im Fenster, wie
// lang die Bestellung und der Bestellbedarf darunter auch sind.
//
// Die Liste wächst mit der Seite; die Zusammenfassung daneben klebt und reicht bis zum unteren
// Rand (actions/resthoehe.js). Gemessen wird, ohne den Knopf ins Bild zu holen: scrollIntoView
// fände ihn auch dann, wenn er unter dem Rand läge, indem es die Liste wegschiebt.
import { test, expect } from '@playwright/test';
import { uiLogin, seedBestellbedarf, seedSQL, querySQL, uniqueSuffix } from './helpers.js';

// Der Test bringt seinen Bedarf selbst mit, statt ihn vorauszusetzen: Ist die Bestellbedarfs-Warnung
// ausgeschaltet, liefert /api/bestellungen eine leere Liste, und das Gate hinge an der Datenlage.
/** @type {(() => void) | undefined} */
let aufraeumen;
test.beforeEach(() => {
	// Mehr Titel, als in das höchste Fenster passen: Erst dann scrollt die Seite.
	aufraeumen = seedBestellbedarf(30);
});
test.afterEach(() => {
	aufraeumen?.();
	aufraeumen = undefined;
});

/** Schul-PCs sind kurz. Die erste Größe ist der gemeldete Fall. */
const FENSTER = [
	[1366, 700],
	[1280, 800],
	[1920, 1080]
];

/**
 * Lage von Zusammenfassung, Kästchen und Knopf im Fenster; `seite` ist der Bereich, der die Seite
 * scrollt (der nächste scrollende Vorfahr der Zusammenfassung, wie in resthoehe.js).
 * @param {'oben' | 'unten'} [ziel] scrollt die Seite vorher an den Anfang oder ans Ende
 */
function lage(ziel) {
	const spalte = document.getElementById('zusammenfassung');
	if (!spalte) throw new Error('Die Zusammenfassung (#zusammenfassung) gibt es nicht mehr');
	let seite = spalte.parentElement;
	while (seite && !/(auto|scroll)/.test(getComputedStyle(seite).overflowY)) {
		seite = seite.parentElement;
	}
	if (!seite) throw new Error('Kein Bereich scrollt die Seite');
	if (ziel) seite.scrollTop = ziel === 'oben' ? 0 : seite.scrollHeight;
	const knopf = [...spalte.querySelectorAll('button')].find((b) =>
		/Bestellung(?:en)? auslösen/.test(b.textContent || '')
	);
	const kaestchen = [...spalte.querySelectorAll('label')].find((l) =>
		/Barcodes mitschicken/.test(l.textContent || '')
	);
	if (!knopf || !kaestchen) throw new Error('Knopf oder Kästchen fehlt in der Zusammenfassung');
	const k = knopf.getBoundingClientRect();
	return {
		spalteUnten: Math.round(spalte.getBoundingClientRect().bottom),
		knopfOben: Math.round(k.top),
		knopfUnten: Math.round(k.bottom),
		kaestchenUnten: Math.round(kaestchen.getBoundingClientRect().bottom),
		seiteScrollt: seite.scrollHeight > seite.clientHeight + 1,
		fenster: window.innerHeight
	};
}

for (const [breite, hoehe] of FENSTER) {
	test(`Summe und Knopf bleiben bei ${breite}×${hoehe} im Fenster`, async ({ page }) => {
		test.setTimeout(120_000);
		await page.setViewportSize({ width: breite, height: hoehe });
		await uiLogin(page);
		await page.goto('/bestellungen');

		// Ein hinzugefügter Titel verlässt den Bestellbedarf; der nächste Knopf steht deshalb immer an
		// erster Stelle. Auf jede Wirkung wird gewartet, und am Ende belegt der Knopf, dass wirklich
		// Positionen im Korb liegen: Sonst mäße der Test den falschen Fall.
		const knopf = page.getByRole('button', { name: /Bestellung(?:en)? auslösen/ });
		const hinzufuegen = page
			.getByRole('region', { name: 'Bestellbedarf' })
			.getByRole('button', { name: /^\d+ hinzufügen/ });
		let exemplare = 0;
		for (let i = 0; i < 6; i++) {
			const erster = hinzufuegen.first();
			await erster.waitFor({ timeout: 15_000 });
			exemplare += Number(/\d+/.exec((await erster.textContent()) || '')?.[0]);
			await erster.click();
			await expect(knopf).toHaveText(new RegExp(`${exemplare}\\s*Exemplar`), { timeout: 10_000 });
		}
		await expect(page.locator('#zusammenfassung').getByRole('listitem')).not.toHaveCount(0);

		for (const ziel of /** @type {const} */ (['oben', 'unten'])) {
			// Die Zusammenfassung misst ihre Höhe nach dem Scrollen neu; deshalb wartend prüfen.
			await expect
				.poll(async () => (await page.evaluate(lage, ziel)).spalteUnten, {
					timeout: 5000,
					intervals: [50, 100, 200, 400],
					message: `Die Zusammenfassung reicht ${ziel} unter den ${hoehe}-px-Fensterrand (actions/resthoehe.js).`
				})
				.toBeLessThanOrEqual(hoehe);
			const m = await page.evaluate(lage);
			expect(m.seiteScrollt, 'Die Seite scrollt nicht: Der Test misst den falschen Fall').toBe(
				true
			);
			expect(
				m.knopfOben >= 0 && m.knopfUnten <= hoehe,
				`„Bestellung auslösen" liegt ${ziel} bei y=${m.knopfOben} bis ${m.knopfUnten}, außerhalb des ${hoehe}-px-Fensters`
			).toBe(true);
			// Kästchen und Knopf sind beide inline; in einem gemeinsamen Block rückte der Knopf
			// neben das Kästchen (docs/sweeps.md, „Knopf in der Zeile des Nachbarn").
			expect(
				m.kaestchenUnten,
				`„Bestellung auslösen" beginnt bei y=${m.knopfOben}, über dem Ende des Kästchens (${m.kaestchenUnten})`
			).toBeLessThanOrEqual(m.knopfOben);
		}
	});
}

// Unter 1024 px steht die Zusammenfassung in voller Breite unter der Liste: Dort passen Kästchen und
// Knopf nebeneinander, und dort rückte der Knopf ohne eigenen Block in die Zeile des Kästchens.
test('„Bestellung auslösen" steht auch in voller Breite in einer eigenen Zeile', async ({
	page
}) => {
	await page.setViewportSize({ width: 900, height: 1000 });
	await uiLogin(page);
	await page.goto('/bestellungen');
	const knopf = page.getByRole('button', { name: /Bestellung(?:en)? auslösen/ });
	await page
		.getByRole('region', { name: 'Bestellbedarf' })
		.getByRole('button', { name: /^\d+ hinzufügen/ })
		.first()
		.click();
	await knopf.scrollIntoViewIfNeeded();
	const m = await page.evaluate(lage);
	expect(
		m.kaestchenUnten,
		`„Bestellung auslösen" beginnt bei y=${m.knopfOben}, über dem Ende des Kästchens (${m.kaestchenUnten})`
	).toBeLessThanOrEqual(m.knopfOben);
});

// Die Trefferliste der Titelsuche steht über allem, was darunter folgt, und kein scrollender Bereich
// schneidet sie ab: In einer Spalte mit eigenem Scrollbalken lag sie zum Teil unter deren Rand.
// Die Antwort der Suche ist nachgestellt; der Stack fragte sonst die DNB.
test('Die Trefferliste der Titelsuche liegt bei leerem Warenkorb ganz im Fenster', async ({
	page
}) => {
	await page.setViewportSize({ width: 1280, height: 800 });
	const treffer = Array.from({ length: 6 }, (_, i) => ({
		id: `00000000-0000-4000-8000-00000000000${i}`,
		titel: `Trefferprobe ${i + 1}`,
		autor: 'Muster, Erika',
		isbn: `978000000000${i}`,
		source: 'local',
		current_stock: i
	}));
	await page.route('**/api/bestellungen/suche', (route) =>
		route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(treffer) })
	);
	await uiLogin(page);
	await page.goto('/bestellungen');
	const antwort = page.waitForResponse('**/api/bestellungen/suche');
	await page.getByRole('searchbox', { name: 'Titel suchen & hinzufügen' }).fill('trefferprobe');
	await antwort;

	const liste = page.getByText('Im lokalen Bestand').locator('..');
	await expect(page.getByRole('button', { name: /Trefferprobe 1/ })).toBeVisible();
	const ort = await liste.evaluate((el) => {
		const r = el.getBoundingClientRect();
		const unten = document.elementFromPoint(r.left + r.width / 2, r.bottom - 4);
		return {
			listeUnten: Math.round(r.bottom),
			listeHoehe: Math.round(r.height),
			fenster: window.innerHeight,
			ganzZuSehen: el.contains(unten)
		};
	});
	// Sechs Treffer sind höher als die Liste: Sie steht an ihrer Höchstgrenze und scrollt in sich.
	expect(ort.listeHoehe, 'Die Trefferliste steht nicht an ihrer Höchstgrenze').toBe(288);
	expect(
		ort.listeUnten,
		`Die Trefferliste endet bei ${ort.listeUnten} px, das Fenster bei ${ort.fenster} px`
	).toBeLessThanOrEqual(ort.fenster);
	expect(ort.ganzZuSehen, 'Der untere Rand der Trefferliste ist abgeschnitten oder verdeckt').toBe(
		true
	);
});

// Erschien ein Hinweis über dem Bestellwesen erst nach dem Aufbau der Zusammenfassung, rückte sie um
// seine Höhe unter den Rand: Keine beobachtete Größe änderte sich, und gemessen wurde nicht neu. In der
// ganzen Suite hing das an der Reihenfolge der Antworten (ein früherer Fall lässt einen
// Hauptlieferanten zurück, die öffentliche Adresse fehlt). Hier steht das Rennen fest: Die Antwort,
// die den Hinweis auslöst, kommt anderthalb Sekunden nach dem Aufbau.
test('Ein Hinweis, der nach dem Aufbau erscheint, schiebt die Zusammenfassung nicht unter den Rand', async ({
	page
}) => {
	const s = uniqueSuffix();
	const adresse = querySQL(
		`SELECT wert FROM system_einstellungen WHERE schluessel = 'oeffentliche_adresse'`
	);
	seedSQL(`
		DELETE FROM system_einstellungen WHERE schluessel = 'oeffentliche_adresse';
		UPDATE lieferanten SET ist_hauptlieferant = false WHERE ist_hauptlieferant;
		INSERT INTO lieferanten (name, email, kundennummer, ist_hauptlieferant)
		VALUES ('E2E-Spaeter-Hinweis ${s}', 'e2e-${s}@example.invalid', 'K-${s}', true);
	`);
	try {
		await page.route('**/api/bestellungen/konfiguration', async (route) => {
			await new Promise((fertig) => setTimeout(fertig, 1500));
			return route.fallback();
		});
		await page.setViewportSize({ width: 1280, height: 800 });
		await uiLogin(page);
		await page.goto('/bestellungen');
		await expect(page.getByText('Bestellungen gehen ohne Bestätigungs-Link raus')).toBeVisible({
			timeout: 10_000
		});
		await expect
			.poll(
				() =>
					page.evaluate(() => {
						const spalte = document.getElementById('zusammenfassung');
						if (!spalte)
							throw new Error('Die Zusammenfassung (#zusammenfassung) gibt es nicht mehr');
						return Math.round(spalte.getBoundingClientRect().bottom);
					}),
				{ timeout: 5000, message: 'Die Zusammenfassung reicht nach dem Hinweis unter den Rand' }
			)
			.toBeLessThanOrEqual(800);
	} finally {
		// Der Hauptlieferant der Anlage kommt mit dem globalTeardown zurück (e2e/global-setup.js).
		seedSQL(`
			DELETE FROM lieferanten WHERE name = 'E2E-Spaeter-Hinweis ${s}';
			${
				adresse
					? `INSERT INTO system_einstellungen (schluessel, wert) VALUES ('oeffentliche_adresse', '${adresse}')
					   ON CONFLICT (schluessel) DO UPDATE SET wert = EXCLUDED.wert;`
					: ''
			}
		`);
	}
});
