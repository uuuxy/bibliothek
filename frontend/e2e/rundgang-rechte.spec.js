// Rundgang je Recht: Wer auf der Berechtigungsseite nur ein Recht bekommt, öffnet alles, was er
// damit erreicht, ohne dass der Server ablehnt. Eine Seite, die mit einem Recht aufgeht und Daten
// hinter einem anderen abfragt, bleibt sonst unsichtbar: Ab Werk hält eine Rolle meist beide.
// Von Hand fährt RUNDGANG_ALLE_AUSSER=1 zusätzlich je Recht das Konto mit allen übrigen Rechten.
import { test, expect } from '@playwright/test';
import { uiLogin, apiPut, querySQL, seedBenutzer, seedSQL } from './helpers.js';
import { geheRundgang, legeAktenAn } from './rundgang-hilfe.js';
import { pfadJeBildschirm } from '../src/lib/routenTabelle.js';
import { bildschirmGesperrt, erlaubteBildschirme } from '../src/lib/menue.js';

const ROLLE = 'mitarbeiter';
const EMAIL = 'e2e-rundgang-recht@test.local';

/**
 * Die Pfade, die das Konto laut Navigation öffnen darf, dazu die Akte eines Titels.
 * @param {string[]} rechte
 * @param {string} buchakte
 */
function bildschirmeMit(rechte, buchakte) {
	const erlaubt = erlaubteBildschirme({ rolle: ROLLE, permissions: rechte });
	return Object.entries({ ...pfadJeBildschirm, book_detail: buchakte })
		.filter(([tab]) => !bildschirmGesperrt(tab, erlaubt))
		.map(([, pfad]) => pfad);
}

/**
 * Stellt die Rolle auf genau diese Rechte. SQL schreibt die Zeilen in einem Zug; die Tür der
 * Berechtigungsseite leert danach den Zwischenspeicher der Rechte am Server.
 * @param {import('@playwright/test').Page} admin
 * @param {string[]} rechte
 */
async function setzeRechte(admin, rechte) {
	const liste = rechte.map((r) => `'${r}'`).join(',') || `''`;
	seedSQL(`UPDATE role_permissions SET allowed = permission IN (${liste})
		WHERE UPPER(role) = '${ROLLE.toUpperCase()}';`);
	const antwort = await apiPut(admin, '/api/admin/permissions', {
		role: ROLLE,
		permission: rechte[0],
		allowed: true
	});
	expect(antwort.ok(), `Rechte der Rolle setzen: ${antwort.status()}`).toBe(true);
}

test('Ein Konto mit einem einzigen Recht öffnet, was es erreicht, ohne Ablehnung', async ({
	browser,
	page
}) => {
	test.setTimeout(process.env.RUNDGANG_ALLE_AUSSER ? 3_600_000 : 900_000);
	seedBenutzer(EMAIL, ROLLE);
	await uiLogin(page);
	// Die Seite verlassen, die Anmeldung behalten: Die offene Anwendung sperrt sich nach einer Weile
	// ohne Bedienung selbst, und die Tür der Rechte antwortet dann 423.
	await page.goto('about:blank');

	const zeilen = querySQL(
		`SELECT permission, allowed FROM role_permissions WHERE UPPER(role) = '${ROLLE.toUpperCase()}' ORDER BY permission;`
	)
		.split('\n')
		.map((z) => z.split('|').map((teil) => teil.trim()));
	const alle = zeilen.map(([recht]) => recht);
	const abWerk = zeilen.filter(([, erlaubt]) => erlaubt === 't').map(([recht]) => recht);
	// Grün heißt nur dann etwas, wenn die Liste auch gelesen wurde: 24 Rechte stehen in der Matrix.
	expect(alle.length).toBeGreaterThanOrEqual(24);

	/** @type {{ name: string, rechte: string[] }[]} */
	const faelle = alle.map((recht) => ({ name: `nur ${recht}`, rechte: [recht] }));
	if (process.env.RUNDGANG_ALLE_AUSSER) {
		faelle.push(
			...alle.map((recht) => ({
				name: `alle außer ${recht}`,
				rechte: alle.filter((r) => r !== recht)
			}))
		);
	}

	/** @type {string[]} */
	const funde = [];
	let besuchtGesamt = 0;
	const akten = legeAktenAn();
	try {
		for (const { name, rechte } of faelle) {
			await setzeRechte(page, rechte);
			const kontext = await browser.newContext();
			const seite = await kontext.newPage();
			// Scannen verlangt perform_actions; die Theke selbst geht auch ohne das Recht auf.
			const { stoerungen, besucht, gescannt } = await geheRundgang(
				seite,
				EMAIL,
				bildschirmeMit(rechte, akten.buchakte),
				rechte.includes('perform_actions') ? akten.ausweis : undefined
			);
			await kontext.close();
			funde.push(...stoerungen.map((s) => `${name}: ${s}`));
			if (rechte.includes('perform_actions') && !gescannt) {
				funde.push(`${name}: Die Theke ging nicht auf, gescannt wurde nichts.`);
			}
			besuchtGesamt += besucht.size;
		}
	} finally {
		await setzeRechte(page, abWerk);
		akten.raeumeAuf();
		seedSQL(`DELETE FROM benutzer WHERE email = '${EMAIL}';`);
	}

	expect(funde, 'Störungen im Rundgang je Recht').toEqual([]);
	// Jeder Fall öffnet mindestens seine Startseite.
	expect(besuchtGesamt).toBeGreaterThanOrEqual(faelle.length);
});
