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
