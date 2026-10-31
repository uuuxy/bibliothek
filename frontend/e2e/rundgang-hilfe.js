// Der Gang über Bildschirme und Reiter für die Rundgänge je Rolle und je Recht.
import { uiLogin, querySQL, scanneWieScanner, seedSQL, uniqueSuffix } from './helpers.js';
import { beobachteDatenanfragen, warteAufInhalt } from './messhilfe.js';

const THEKE = '/kiosk';

/**
 * @param {import('@playwright/test').Page} page
 * @param {() => string} ort wo der Rundgang gerade steht
 * @returns {string[]} wächst mit jeder Störung
 */
export function sammleStoerungen(page, ort) {
	/** @type {string[]} */
	const stoerungen = [];
	page.on('pageerror', (fehler) => stoerungen.push(`${ort()}: Absturz: ${fehler.message}`));
	page.on('console', (meldung) => {
		// Eine abgelehnte Anfrage steht unten mit ihrer Adresse; der Browser meldet sie hier ohne.
		if (meldung.type() === 'error' && !meldung.text().startsWith('Failed to load resource')) {
			stoerungen.push(`${ort()}: Konsole: ${meldung.text()}`);
		}
	});
	page.on('response', (antwort) => {
		const { pathname } = new URL(antwort.url());
		// 401 gehört zur Anmeldung: Davor fragt die Seite, ob schon eine Sitzung besteht.
		if (pathname.startsWith('/api/') && antwort.status() >= 400 && antwort.status() !== 401) {
			stoerungen.push(
				`${ort()}: ${antwort.status()} auf ${antwort.request().method()} ${pathname}`
			);
		}
	});
	return stoerungen;
}

/**
 * Ein Leser und ein Titel für den Rundgang: Die Akte eines Titels trägt ihre Kennung im Pfad, die
 * eines Lesers öffnet an der Theke ein Scan.
 * @returns {{ ausweis: string, buchakte: string, raeumeAuf: () => void }}
 */
export function legeAktenAn() {
	const s = uniqueSuffix().toUpperCase();
	const ausweis = `S-RG${s}`;
	const titel = `E2E-Rundgang ${s}`;
	seedSQL(`INSERT INTO leser (barcode_id, vorname, nachname, klasse, art, abgaenger_jahr)
		VALUES ('${ausweis}', 'Rita', 'Rundgang${s}', '07A', 'schueler', 2031);`);
	const titelId = querySQL(`WITH t AS (
			INSERT INTO buecher_titel (titel, ist_lernmittel) VALUES ('${titel}', true) RETURNING id
		), e AS (
			INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar)
			SELECT id, 'B-RG${s}', true FROM t
		)
		SELECT id FROM t;`);
	return {
		ausweis,
		buchakte: `/medienkatalog/buch/${titelId}`,
		raeumeAuf: () =>
			seedSQL(`DELETE FROM buecher_titel WHERE titel = '${titel}';
				DELETE FROM leser WHERE barcode_id = '${ausweis}';`)
	};
}

/**
 * Meldet das Konto an, öffnet die Bildschirme und klickt auf jedem jeden Reiter. Mit einem Ausweis
 * scannt es ihn an der Theke, wenn sie aufgeht.
 * @param {import('@playwright/test').Page} page
 * @param {string} email
 * @param {string[]} bildschirme Pfade
 * @param {string} [ausweis]
 * @returns {Promise<{ stoerungen: string[], besucht: Set<string>, geklickt: number }>}
 */
export async function geheRundgang(page, email, bildschirme, ausweis) {
	let ort = 'Anmeldung';
	const stoerungen = sammleStoerungen(page, () => ort);
	const anfragen = beobachteDatenanfragen(page);
	const steht = () => warteAufInhalt(anfragen, () => page.locator('body *').count());
	const reiter = page.getByRole('tab', { disabled: false });
	let geklickt = 0;
	/** @param {string} wo */
	async function klickeReiter(wo) {
		for (let i = 0; i < (await reiter.count()); i++) {
			ort = `${wo} › ${await reiter.nth(i).innerText()}`;
			await reiter.nth(i).click();
			await steht();
			geklickt++;
		}
	}
	await uiLogin(page, email);
	await steht();

	/** @type {Set<string>} */
	const besucht = new Set();
	for (const pfad of bildschirme) {
		ort = pfad;
		await page.goto(pfad);
		await page.getByRole('button', { name: 'Abmelden' }).waitFor();
		await steht();
		// Was dem Konto verschlossen ist, führt auf seine Startseite; die zählt nur einmal.
		const angekommen = new URL(page.url()).pathname;
		if (besucht.has(angekommen)) continue;
		besucht.add(angekommen);
		await klickeReiter(angekommen);
	}

	if (ausweis && besucht.has(THEKE)) {
		ort = `${THEKE} › Scan`;
		await page.goto(THEKE);
		await page
			.getByPlaceholder(/scannen/i)
			.first()
			.waitFor();
		const antwort = page.waitForResponse((a) => new URL(a.url()).pathname === '/api/action');
		await scanneWieScanner(page, ausweis);
		await antwort;
		await steht();
		await klickeReiter(ort);
	}
	return { stoerungen, besucht, geklickt };
}
