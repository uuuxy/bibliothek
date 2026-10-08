import { expect } from '@playwright/test';

/**
 * Für die messenden Specs (Kontrast, Schriftgrößen, Feldhöhen, Trefferflächen): Eine Seite
 * wird erst gemessen, wenn ihre Datenanfragen beantwortet sind, und der Test wird rot, wenn
 * nach der Messung noch Inhalt nachkommt. Eine feste Untergrenze je Seite taugt dafür nicht,
 * weil die Datenmenge je Umgebung verschieden ist.
 */

/**
 * Führt Buch über die offenen Datenanfragen der Seite: fetch und XHR unter /api/. Der
 * Ereignisstrom bleibt immer offen und zählt nicht; deshalb tritt `networkidle` hier nie ein.
 * @param {import('@playwright/test').Page} page
 */
export function beobachteDatenanfragen(page) {
	/** @type {Set<import('@playwright/test').Request>} */
	const offen = new Set();
	const neuesDokument = (/** @type {import('@playwright/test').Request} */ anfrage) =>
		anfrage.isNavigationRequest() && anfrage.frame() === page.mainFrame();
	const erledigt = (/** @type {import('@playwright/test').Request} */ anfrage) => {
		offen.delete(anfrage);
	};
	page.on('request', (anfrage) => {
		// Was das alte Dokument noch offen hatte, endet mit ihm, ohne dass ein Ereignis es meldet.
		if (neuesDokument(anfrage)) return offen.clear();
		const art = anfrage.resourceType();
		if ((art === 'fetch' || art === 'xhr') && new URL(anfrage.url()).pathname.startsWith('/api/')) {
			offen.add(anfrage);
		}
	});
	// Beantwortet ist eine Anfrage mit ihrer Antwort; ein Ende meldet der Browser nicht zu jeder.
	page.on('response', (antwort) => {
		if (neuesDokument(antwort.request())) offen.clear();
		else erledigt(antwort.request());
	});
	page.on('requestfinished', erledigt);
	page.on('requestfailed', erledigt);
	return {
		offen: () => offen.size,
		pfade: () => [...offen].map((anfrage) => new URL(anfrage.url()).pathname)
	};
}

/**
 * Wartet, bis keine Datenanfrage mehr offen ist und `zaehle` dreimal hintereinander dasselbe
 * liefert. Die Null zählt mit: Manche Seiten haben nichts von dem, was ein Spec misst.
 * @param {ReturnType<typeof beobachteDatenanfragen>} anfragen
 * @param {() => Promise<number>} zaehle
 * @returns {Promise<number>} die Zählung, bei der gemessen wird
 */
export async function warteAufInhalt(anfragen, zaehle) {
	let vorherige = -1;
	let gleich = 0;
	try {
		await expect
			.poll(
				async () => {
					const jetzt = anfragen.offen() > 0 ? -1 : await zaehle();
					gleich = jetzt >= 0 && jetzt === vorherige ? gleich + 1 : 0;
					vorherige = jetzt;
					return gleich;
				},
				{ timeout: 20_000, intervals: [200, 300, 400, 500] }
			)
			.toBeGreaterThanOrEqual(2);
	} catch (fehler) {
		throw new Error(
			`Die Seite kommt nicht zur Ruhe. Offene Datenanfragen: ${anfragen.pfade().join(', ') || 'keine'}; ` +
				`letzte Zählung: ${vorherige}.`,
			{ cause: fehler }
		);
	}
	return vorherige;
}

/**
 * Die Gegenprobe nach einer Messung. Sie traut dem Warten davor nicht: Steht nach dem Ende
 * aller Datenanfragen etwas anderes da als bei der Messung, wurde die Seite ohne ihren Inhalt
 * bewertet.
 * @param {import('@playwright/test').Page} page
 * @param {ReturnType<typeof beobachteDatenanfragen>} anfragen
 * @param {() => Promise<number>} zaehle
 * @param {number} gemessen die Zählung aus warteAufInhalt
 * @param {string} seite
 */
export async function pruefeNichtZuFruehGemessen(page, anfragen, zaehle, gemessen, seite) {
	await expect
		.poll(
			async () => {
				if (anfragen.offen() > 0) return false;
				// Eine Antwort ist da, bevor sie gezeichnet ist: zwei Bilder abwarten.
				await page.evaluate(
					() => new Promise((fertig) => requestAnimationFrame(() => requestAnimationFrame(fertig)))
				);
				return anfragen.offen() === 0;
			},
			{ timeout: 20_000, intervals: [100, 200, 300] }
		)
		.toBe(true);
	expect(
		await zaehle(),
		`${seite}: Nach dem Ende der Datenanfragen steht etwas anderes da als bei der Messung ` +
			`(dort ${gemessen}) — die Seite wurde ohne ihren Inhalt gemessen.`
	).toBe(gemessen);
}
