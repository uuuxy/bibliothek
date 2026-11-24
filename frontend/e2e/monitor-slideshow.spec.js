import { test, expect } from '@playwright/test';

// Der Flur-Monitor: ein Bildschirm ohne Tastatur, der wochenlang läuft — ohne Anmeldung.
//
// Bis zum 30.08.2026 lud die Seite ihre Daten genau EINMAL beim Start. Zwei Folgen, beide
// still: Nach einem Stromausfall bootet der Monitor-PC schneller als der Server, der
// erste Abruf scheiterte, und „Lade Daten …" stand für immer. Und wer durchkam, zeigte
// für den Rest der Woche die Daten vom Einschalttag — „Beliebt diese Woche" von letzter
// Woche. Der frühere Smoke-Test („Seite lädt, kein Not Found") konnte davon nichts sehen.
//
// Alles mit gestellter Uhr (page.clock): Die Fristen sind Minuten, kein Test wartet sie ab.
// Der Endpunkt ist abgefangen — der Test hängt nicht am Datenbestand, und er kann die
// Antwort des Servers gezielt scheitern lassen.

/** Drei Folien, jede trägt die Stand-Marke im Titel — sichtbar, egal welche Folie gerade läuft. */
function folien(stand) {
	const titel = (zusatz) => ({
		id: zusatz,
		titel: `${stand} ${zusatz}`,
		autor: '',
		cover_url: '',
		isbn: ''
	});
	return {
		buch_des_monats: titel('Monatsbuch'),
		neu_eingetroffen: [titel('Neuzugang')],
		beliebt: [titel('Renner')]
	};
}

test('Monitor: ohne Login, holt sich nach gescheitertem Start die Daten und lädt alle fünf Minuten nach', async ({
	page
}) => {
	// Feste Startzeit, damit kein Lauf zufällig über den nächtlichen Neustart springt.
	await page.clock.install({ time: new Date('2026-09-01T10:00:00') });

	let abrufe = 0;
	await page.route('**/api/monitor/slides', async (route) => {
		abrufe++;
		if (abrufe === 1) {
			// Der Server ist noch nicht da (Boot-Wettlauf nach Stromausfall).
			await route.fulfill({ status: 503, contentType: 'application/json', body: '{}' });
			return;
		}
		await route.fulfill({
			status: 200,
			contentType: 'application/json',
			body: JSON.stringify(folien(abrufe === 2 ? 'Erster Stand' : 'Zweiter Stand'))
		});
	});

	await page.goto('/monitor');
	await expect(page).toHaveURL(/\/monitor/); // keine Umleitung zum Login
	await expect.poll(() => abrufe).toBe(1);

	// 1. Nach dem Fehlstart kommt von selbst ein zweiter Versuch — und dann Inhalt.
	await page.clock.fastForward('00:31');
	await expect(page.getByText('Erster Stand').first()).toBeVisible();
	expect(abrufe, 'Neuversuch nach dem Fehlstart').toBe(2);

	// 2. Fünf Minuten später wird nachgeladen; der neue Stand erscheint mit dem nächsten
	//    Folienwechsel — nicht unter den Augen des Betrachters.
	const dritterAbruf = page.waitForResponse('**/api/monitor/slides');
	await page.clock.fastForward('05:00');
	await dritterAbruf;
	expect(abrufe, 'Nachladen im Fünf-Minuten-Takt').toBe(3);
	await page.clock.fastForward('00:15');
	await expect(page.getByText('Zweiter Stand').first()).toBeVisible();
});

// Rotiert der Monitor überhaupt von selbst? Der frühere Smoke-Test konnte auch das
// nicht sehen — und im Produktvideo stand 9 s lang dieselbe Folie, weil der erste
// Wechsel erst nach 15 s kommt. Hier mit gestellter Uhr, am echten Bundle: Wechsel,
// Überspringen einer leeren Folie (Ferien), Sprung über die Punkte.
test('Monitor: Folien wechseln alle 15 Sekunden von selbst, leere werden übersprungen, die Punkte springen', async ({
	page
}) => {
	await page.clock.install({ time: new Date('2026-09-01T10:00:00') });
	await page.route('**/api/monitor/slides', (route) =>
		route.fulfill({
			status: 200,
			contentType: 'application/json',
			body: JSON.stringify({ ...folien('Stand'), beliebt: [] })
		})
	);

	await page.goto('/monitor');
	const folie = page.getByTestId('monitor-folie');
	await expect(folie).toHaveText('Buch des Monats');
	// Ohne Cover nennt auch der Platzhalter den Titel; geprüft wird die Überschrift der Folie.
	await expect(page.getByRole('heading', { name: 'Stand Monatsbuch' })).toBeVisible();

	await page.clock.fastForward('00:15');
	await expect(folie).toHaveText('Neu eingetroffen');
	await expect(page.getByText('Stand Neuzugang').last()).toBeVisible();

	// „Beliebt" ist leer (Ferien) — der Takt überspringt sie und fängt vorne an.
	await page.clock.fastForward('00:15');
	await expect(folie).toHaveText('Buch des Monats');

	// Von Hand ist auch die leere Folie erreichbar — wer klickt, will sie sehen.
	await page.getByRole('button', { name: 'Beliebt diese Woche anzeigen' }).click();
	await expect(folie).toHaveText('Beliebt diese Woche');
	await expect(page.getByText('Keine Daten verfügbar')).toBeVisible();
});

// Am Tag der Bücherausgabe steht die Folie „Heute" vorn, mit eigenem Punkt, und wächst mit dem
// Bildschirm: gezeichnet im Raster des Fernsehers, auf Full HD doppelt so groß. Mit dem ersten
// Stand ohne Plantag verschwinden Folie und Punkt wieder.
test('Monitor: Am Tag des LMF-Plans steht die Folie „Heute" vorn, in Fernsehgröße', async ({
	page
}) => {
	await page.clock.install({ time: new Date('2026-09-01T10:00:00') });
	await page.setViewportSize({ width: 1920, height: 1080 });
	let plantag = true;
	await page.route('**/api/monitor/slides', (route) =>
		route.fulfill({
			status: 200,
			contentType: 'application/json',
			body: JSON.stringify({
				...folien('Stand'),
				lmf_heute: plantag
					? {
							art: 'ausgabe',
							datum: '2026-09-01',
							termine: [
								{ stunde: 1, klassen: ['05F'], vermerk: '' },
								{ stunde: 2, klassen: [], vermerk: 'Nachzügler' }
							]
						}
					: null
			})
		})
	);
	await page.goto('/monitor');
	const folie = page.getByTestId('monitor-folie');
	await expect(folie).toHaveText('Heute Bücherausgabe');
	const titel = page.getByRole('heading', { name: 'Heute Bücherausgabe' });
	await expect(titel).toBeVisible();
	await expect(page.getByRole('listitem')).toHaveText(['1 05F', '2 Nachzügler']);
	// Zwei Zeilen der Rolle display-medium (52 px Zeilenhöhe), auf Full HD verdoppelt.
	const hoehe = (await titel.boundingBox())?.height ?? 0;
	expect(hoehe, 'Titel der Folie in Fernsehgröße').toBeGreaterThan(200);
	expect(hoehe).toBeLessThan(216);
	await expect(page.getByRole('button', { name: /anzeigen$/ })).toHaveCount(4);

	// fastForward löst einen fälligen Takt nur einmal aus: ein Schritt je Folie.
	for (const naechste of [
		'Buch des Monats',
		'Neu eingetroffen',
		'Beliebt diese Woche',
		'Heute Bücherausgabe'
	]) {
		await page.clock.fastForward('00:15');
		await expect(folie).toHaveText(naechste);
	}

	// Der nächste Stand kennt den Plantag nicht mehr; er gilt ab dem Folienwechsel danach.
	plantag = false;
	const abruf = page.waitForResponse('**/api/monitor/slides');
	await page.clock.fastForward('04:00');
	await abruf;
	await page.clock.fastForward('00:15');
	await expect(page.getByRole('button', { name: /anzeigen$/ })).toHaveCount(3);
	await expect(page.getByRole('button', { name: /^Heute/ })).toHaveCount(0);
	for (let i = 0; i < 3; i++) {
		await expect(folie).not.toHaveText(/^Heute/);
		await page.clock.fastForward('00:15');
	}
});
