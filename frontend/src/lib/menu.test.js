import { describe, it, expect } from 'vitest';
import { existsSync, readFileSync } from 'node:fs';
import { dirname, resolve } from 'node:path';
import { canSeeItem, erlaubteTabs, menuGroups, tabIstGesperrt } from './menu.js';

/**
 * Liest die geseedeten Rechte einer Rolle aus db/seed.go.
 *
 * Warum aus der Datei statt als Literal hier: Der Kollegiums-Test unten stand bis zum
 * 10.08.2026 auf `permissions: []` — einem Konto, das es nicht gibt. Er war grün, während
 * ein echtes Kollegiums-Konto 10 von 15 Menüpunkten sah (Schülerdatei, Mahnwesen,
 * System-Logs, Einstellungen). Ein Rechte-Literal im Test misst nur sich selbst; die
 * Vorgabe steht in seed.go, also muss sie von dort kommen.
 *
 * @param {string} rolle GROSS geschrieben, wie in role_permissions
 * @returns {string[]}
 */
function geseedeteRechte(rolle) {
	// Aufwärts suchen statt relativ zu import.meta.url: Vitest transformiert die Datei,
	// import.meta.url trägt dort ein http-Schema und readFileSync lehnt es ab.
	let verzeichnis = process.cwd();
	while (!existsSync(resolve(verzeichnis, 'db/seed.go'))) {
		const eltern = dirname(verzeichnis);
		if (eltern === verzeichnis) throw new Error('db/seed.go nicht gefunden');
		verzeichnis = eltern;
	}

	const seed = readFileSync(resolve(verzeichnis, 'db/seed.go'), 'utf8');
	const zeilen = [...seed.matchAll(/\{"([A-Z]+)",\s*"(\w+)",\s*(true|false)\}/g)];

	// Ohne diesen Riegel prüft der Test nichts mehr, sobald sich die Schreibweise in
	// seed.go ändert: Die leere Liste ergäbe „sieht nur sein Portal" — grün aus dem
	// falschen Grund, genau der Fehler, den dieser Test ablösen soll.
	const dieserRolle = zeilen.filter(([, r]) => r === rolle);
	if (dieserRolle.length === 0) {
		throw new Error(
			`Keine Vorgaben für Rolle ${rolle} in db/seed.go gefunden — das Muster passt nicht mehr.`
		);
	}

	return dieserRolle.filter(([, , , erlaubt]) => erlaubt === 'true').map(([, , recht]) => recht);
}

/**
 * Die Sichtbarkeitsregeln der Navigation — hier stand ein Widerspruch zwischen
 * Oberfläche und Rechtelage.
 *
 * Das Backend hat den Admin-Vorrang fest eingebaut („Admins dürfen immer",
 * api/permission_middleware.go). Im Menü stand die Portal-Ausnahme aber ÜBER dem
 * Admin-Vorrang. Ergebnis: Ein Admin durfte die Klassensatz-Reservierung aufrufen, sah
 * den Menüpunkt aber nicht — und fand die Funktion deshalb schlicht nicht. Genau so ist
 * es passiert (Peter, 09.08.2026: „Klassensatz reservieren wo und wie?").
 *
 * Deshalb prüft der erste Test nicht eine Regel, sondern die Zusage: Der Admin sieht
 * ALLES, ausnahmslos.
 */
const admin = { rolle: 'admin' };
const kollegium = { rolle: 'kollegium', permissions: geseedeteRechte('KOLLEGIUM') };
const helfer = { rolle: 'helfer', permissions: geseedeteRechte('HELFER') };

/** Alle Menüpunkte flach, unabhängig von der Gruppierung. */
const allePunkte = menuGroups.flatMap((g) => g.items);

describe('Menü-Sichtbarkeit', () => {
	it('zeigt dem Admin ausnahmslos jeden Menüpunkt', () => {
		const unsichtbar = allePunkte.filter((item) => !canSeeItem(item, admin)).map((i) => i.id);

		expect(
			unsichtbar,
			`Diese Punkte sieht der Admin nicht. Das Backend lässt ihn überall hinein — eine\n` +
				`Ausnahme im Menü macht die Funktion damit nur unauffindbar, nicht sicherer:\n  ${unsichtbar.join('\n  ')}`
		).toEqual([]);
	});

	it('zeigt dem Kollegium nur sein Portal', () => {
		const sichtbar = allePunkte.filter((item) => canSeeItem(item, kollegium)).map((i) => i.id);
		expect(
			sichtbar,
			`Eine Lehrkraft meldet sich an, um einen Klassensatz zu reservieren — mehr ist der\n` +
				`Zweck der Rolle nicht. Diese Punkte sieht sie zusätzlich:\n  ` +
				`${sichtbar.filter((id) => id !== 'kollegium_portal').join('\n  ')}`
		).toEqual(['kollegium_portal']);
	});

	it('macht aus manage_users keinen Zugang zu den Einstellungen', () => {
		// „Einstellungen" trägt roles: ['admin'] UND permission: 'manage_users'. Die
		// Rollenliste war wirkungslos, weil canSeeItem bei gesetzter permission direkt zur
		// Rechteprüfung sprang — jede Rolle mit manage_users sah die Systemeinstellungen.
		const settings = allePunkte.find((i) => i.id === 'settings');
		if (!settings) throw new Error('Menüpunkt settings fehlt — Test läuft ins Leere');

		for (const rolle of ['kollegium', 'mitarbeiter', 'helfer']) {
			expect(canSeeItem(settings, { rolle, permissions: ['manage_users'] }), rolle).toBe(false);
		}
		expect(canSeeItem(settings, admin)).toBe(true);
	});

	it('hält das Portal von allen anderen Rollen fern', () => {
		const portal = allePunkte.find((i) => i.id === 'kollegium_portal');
		// Kein expect(...).toBeTruthy(): Das verengt den Typ nicht, und ohne den Punkt
		// prüfte der Test unten stillschweigend gegen undefined.
		if (!portal) throw new Error('Menüpunkt kollegium_portal fehlt — Test läuft ins Leere');
		expect(canSeeItem(portal, helfer)).toBe(false);
		expect(canSeeItem(portal, { rolle: 'mitarbeiter', permissions: ['*'] })).toBe(false);
	});

	it('sperrt auch die Unteransichten, die keinen Menüpunkt haben', () => {
		// Buchakte und Statistik-Detail haben keinen eigenen Menüeintrag und waren deshalb
		// von der Router-Prüfung ausgenommen — per URL-Zeile standen sie jedem angemeldeten
		// Benutzer offen. Sie erben jetzt die Regel ihrer Elternansicht.
		const erlaubtKollegium = erlaubteTabs(kollegium);
		expect(tabIstGesperrt('book_detail', erlaubtKollegium), 'book_detail').toBe(true);
		expect(tabIstGesperrt('stats_detail', erlaubtKollegium), 'stats_detail').toBe(true);

		// Und bleiben offen, wo die Elternansicht offen ist — sonst wäre jeder Sprung in
		// eine Buchakte ein Rauswurf.
		const erlaubtAdmin = erlaubteTabs(admin);
		expect(tabIstGesperrt('book_detail', erlaubtAdmin), 'book_detail/admin').toBe(false);
		expect(tabIstGesperrt('stats_detail', erlaubtAdmin), 'stats_detail/admin').toBe(false);

		// Der Helfer darf den Katalog (view_books), aber keine Statistiken.
		const erlaubtHelfer = erlaubteTabs(helfer);
		expect(tabIstGesperrt('book_detail', erlaubtHelfer), 'book_detail/helfer').toBe(false);
		expect(tabIstGesperrt('stats_detail', erlaubtHelfer), 'stats_detail/helfer').toBe(true);
	});

	it('lässt ohne Anmeldung nichts durch', () => {
		expect(allePunkte.every((item) => canSeeItem(item, null))).toBe(false);
		expect(allePunkte.some((item) => canSeeItem(item, null))).toBe(false);
	});
});
