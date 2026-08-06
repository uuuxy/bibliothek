import { test, expect } from '@playwright/test';
import { uiLogin } from './helpers.js';

// Kommt die Live-Verbindung nach einem Fehler von selbst zurück?
//
// Der Anlass ist eine Meldung aus dem Betrieb: `net::ERR_NETWORK_CHANGED` auf /events —
// harmlos, weil der Browser Netzfehler selbst nachholt. Beim Nachsehen zeigte sich der
// Fall, den er NICHT nachholt: Antwortet der Server mit einem Fehler (/events verlangt
// eine gültige Sitzung, nach 12 Stunden ist sie abgelaufen), schliesst ein EventSource
// endgültig. Zwei der drei Verbindungen hatten dafür keine Vorsorge und blieben bis zum
// nächsten F5 tot — der Bildschirm arbeitete weiter und aktualisierte sich nicht mehr,
// wenn an einem anderen Arbeitsplatz gebucht wurde.
//
// Warum als E2E und nicht nur als Unit-Test: Der Unit-Test ersetzt EventSource durch eine
// Attrappe und kann damit gar nicht zeigen, dass ein ECHTER Browser den Weg noch einmal
// geht. Genau das ist hier die Frage.

test('Live-Verbindung: nach einem Fehler wird neu verbunden — und dann Ruhe', async ({ page }) => {
	let versuche = 0;

	// Die erste Verbindung bekommt eine FEHLERANTWORT (401), danach lassen wir durch.
	//
	// Bewusst kein route.abort(): Ein abgebrochener Request ist ein Netzfehler, und den
	// holt der Browser von sich aus nach — das Gate wäre auch ohne unseren Wiederaufbau
	// grün geblieben (nachgemessen: genau so passiert). Erst die Fehlerantwort erzeugt den
	// Fall, um den es geht: Darauf schliesst ein EventSource ENDGÜLTIG. Was danach noch
	// eine Verbindung öffnet, kann nur unser eigener Wiederaufbau sein.
	//
	// 401 ist zugleich die Antwort aus dem Betrieb: /events verlangt eine gültige Sitzung.
	await page.route('**/events', async (route) => {
		versuche++;
		if (versuche === 1) {
			await route.fulfill({ status: 401, contentType: 'application/json', body: '{}' });
			return;
		}
		await route.continue();
	});

	await uiLogin(page);

	// 1. Es wird überhaupt ein zweites Mal versucht. Ohne eigenen Wiederaufbau bliebe es
	//    bei genau einem Versuch — und die Ansicht wäre für den Rest des Tages blind.
	await expect
		.poll(() => versuche, {
			timeout: 15000,
			message: 'nach dem Fehler kam kein zweiter Verbindungsversuch'
		})
		.toBeGreaterThanOrEqual(2);

	// 2. Und dann ist Ruhe: Steht die Leitung, darf nicht weiter verbunden werden. Ein
	//    Wiederaufbau, der durchläuft, wäre ein Dauerfeuer gegen den Server — mal zehn
	//    Arbeitsplätze, den ganzen Schultag.
	const nachDemAufbau = versuche;
	await page.waitForTimeout(8000);
	expect(versuche, 'die Verbindung wird laufend neu aufgebaut, obwohl sie steht').toBe(
		nachDemAufbau
	);

	// 3. Der Arbeitsplatz ist bedienbar — kein Offline-Overlay über dem Bildschirm.
	await expect(page.getByRole('button', { name: 'Abmelden' })).toBeVisible();
});
