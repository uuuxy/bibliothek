import { describe, it, expect } from 'vitest';
import { ALLE_KATEGORIE_RECHTE } from './components/settings/kategorien.js';
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

	it('öffnet die Einstellungen mit jedem Recht, das eine Kategorie darin öffnet', () => {
		// Bis 24.08.2026 hing „Einstellungen" an manage_users PLUS roles: ['admin'], dann
		// kurz an manage_settings allein. Beides ließ Rechte ohne Tür zurück: Ein
		// Mitarbeiter hat ab Werk import_students, sah aber nie den LUSD-Import, weil der
		// Menüpunkt ein anderes Recht verlangte als die Kategorie dahinter.
		const settings = allePunkte.find((i) => i.id === 'settings');
		if (!settings) throw new Error('Menüpunkt settings fehlt — Test läuft ins Leere');

		for (const rolle of ['kollegium', 'mitarbeiter', 'helfer']) {
			expect(canSeeItem(settings, { rolle, permissions: ['manage_users'] }), rolle).toBe(false);
			expect(canSeeItem(settings, { rolle, permissions: ['view_students'] }), rolle).toBe(false);
			expect(canSeeItem(settings, { rolle, permissions: ['manage_settings'] }), rolle).toBe(true);
			expect(canSeeItem(settings, { rolle, permissions: ['manage_inventory'] }), rolle).toBe(true);
		}
		expect(canSeeItem(settings, admin)).toBe(true);
	});

	it('öffnet den Schuljahreswechsel mit jedem Recht eines seiner Reiter', () => {
		// LUSD-Abgleich und Versetzung wohnen seit 05.09.2026 nicht mehr in den
		// Einstellungen, sondern als Reiter der Seite „Schuljahreswechsel" neben LMF-Plan
		// und Abgängern. Wer nur import_students hat, muss die Seite trotzdem sehen —
		// sonst gäbe es wieder ein Recht ohne Tür.
		const schuljahr = allePunkte.find((i) => i.id === 'schuljahr');
		if (!schuljahr) throw new Error('Menüpunkt schuljahr fehlt — Test läuft ins Leere');
		for (const recht of [
			'edit_books',
			'view_graduates',
			'import_students',
			'manage_students_admin'
		]) {
			expect(canSeeItem(schuljahr, { rolle: 'mitarbeiter', permissions: [recht] }), recht).toBe(
				true
			);
		}
		expect(canSeeItem(schuljahr, { rolle: 'mitarbeiter', permissions: ['view_students'] })).toBe(
			false
		);
		expect(
			canSeeItem(schuljahr, { rolle: 'kollegium', permissions: ['create_reservations'] })
		).toBe(false);
	});

	it('nennt am Menüpunkt „Einstellungen" genau die Rechte der Kategorien', () => {
		// Zwei Listen, die dasselbe meinen: die Türliste am Menüpunkt und die Rechte in
		// kategorien.js. Laufen sie auseinander, gibt es entweder ein Recht ohne Tür
		// (Kategorie sichtbar, Menüpunkt nicht) oder eine Tür ins Leere.
		const settings = allePunkte.find((i) => i.id === 'settings');
		if (!settings) throw new Error('Menüpunkt settings fehlt — Test läuft ins Leere');
		expect([...(settings.permissions ?? [])].sort()).toEqual([...ALLE_KATEGORIE_RECHTE].sort());
	});
	it('öffnet das Portal für jede Rolle mit create_reservations — und nur für die', () => {
		// Bis 26.08.2026 hing „Mein Portal" an der Rolle kollegium. Eine Lehrkraft, die
		// in Bibliothek/LMF mitarbeitet und deshalb Mitarbeiter ist, fand das Portal nicht,
		// obwohl der Server sie mit create_reservations überall hineinließ (Peter: am
		// Recht aufhängen). Der Helfer hat das Recht ab Werk nicht und bleibt draußen.
		const portal = allePunkte.find((i) => i.id === 'kollegium_portal');
		// Kein expect(...).toBeTruthy(): Das verengt den Typ nicht, und ohne den Punkt
		// prüfte der Test unten stillschweigend gegen undefined.
		if (!portal) throw new Error('Menüpunkt kollegium_portal fehlt — Test läuft ins Leere');
		expect(canSeeItem(portal, helfer)).toBe(false);
		expect(
			canSeeItem(portal, { rolle: 'mitarbeiter', permissions: geseedeteRechte('MITARBEITER') })
		).toBe(true);
		expect(canSeeItem(portal, { rolle: 'mitarbeiter', permissions: ['view_books'] })).toBe(false);
		expect(canSeeItem(portal, { rolle: 'helfer', permissions: ['create_reservations'] })).toBe(
			true
		);
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
