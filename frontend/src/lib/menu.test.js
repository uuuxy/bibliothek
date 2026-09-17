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
 * es passiert (Absprache vom 09.08.2026: „Klassensatz reservieren wo und wie?").
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
			expect(canSeeItem(settings, { rolle, permissions: ['import_students'] }), rolle).toBe(true);
			expect(canSeeItem(settings, { rolle, permissions: ['manage_inventory'] }), rolle).toBe(true);
		}
		expect(canSeeItem(settings, admin)).toBe(true);
	});

	it('hängt Abgänger unter Verwaltung an view_graduates und den Schuljahreswechsel unter System an edit_books', () => {
		// Absprache vom 05.09.2026 abends: Unter Verwaltung steht nur „Abgänger"; der LMF-Plan
		// heißt im System-Menü „Schuljahreswechsel"; der LUSD-Import bleibt eine
		// Einstellungs-Kategorie. Eine Sammelseite mit drei Reitern unter Verwaltung gab es
		// für einen Abend — sie hat nicht getragen.
		const gruppe = (/** @type {string} */ id) =>
			menuGroups.find((g) => g.items.some((i) => i.id === id))?.name;
		expect(gruppe('graduates')).toBe('Verwaltung');
		expect(gruppe('schuljahr')).toBe('System');
		const abgaenger = allePunkte.find((i) => i.id === 'graduates');
		const schuljahr = allePunkte.find((i) => i.id === 'schuljahr');
		if (!abgaenger || !schuljahr) throw new Error('Menüpunkt fehlt — Test läuft ins Leere');
		expect(canSeeItem(abgaenger, { rolle: 'mitarbeiter', permissions: ['view_graduates'] })).toBe(
			true
		);
		expect(canSeeItem(abgaenger, { rolle: 'mitarbeiter', permissions: ['edit_books'] })).toBe(
			false
		);
		expect(canSeeItem(schuljahr, { rolle: 'mitarbeiter', permissions: ['edit_books'] })).toBe(true);
		expect(canSeeItem(schuljahr, { rolle: 'mitarbeiter', permissions: ['view_graduates'] })).toBe(
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
		// obwohl der Server sie mit create_reservations überall hineinließ (Absprache: am
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

	it('stellt Statistiken und Bestandsbücher in die Sektion „Berichte", nicht in den System-Topf', () => {
		// Beide standen bis zum 17.09.2026 unter „System" — einer Gruppe, die beim Laden
		// zugeklappt ist (Sidebar.svelte). Wer einen Nachweis oder eine Zahl suchte, sah
		// nichts und musste erst eine Gruppe aufklappen, deren Name nichts davon nennt.
		//
		// Der Test prüft die Sektion UND das Recht: Ein späterer Umzug soll nicht
		// unbemerkt auch die Tür verschieben. Die Bestandsbücher hängen an view_books wie
		// ihre vier Routen (api/routes_books.go); die Helfer haben view_orders ab Werk
		// NICHT, ein Platz im Bestellwesen hätte sie ausgesperrt.
		const gruppe = (/** @type {string} */ id) =>
			menuGroups.find((g) => g.items.some((i) => i.id === id))?.name;
		expect(gruppe('stats')).toBe('Berichte');
		expect(gruppe('bestandsbuecher')).toBe('Berichte');
		expect(gruppe('system-logs')).toBe('System');

		const punkt = (/** @type {string} */ id) => allePunkte.find((i) => i.id === id);
		expect(punkt('stats')?.permission).toBe('view_stats');
		expect(punkt('bestandsbuecher')?.permission).toBe('view_books');

		const bestandsbuecher = punkt('bestandsbuecher');
		if (!bestandsbuecher) throw new Error('Menüpunkt fehlt — Test läuft ins Leere');
		expect(canSeeItem(bestandsbuecher, helfer), 'Helfer sieht die Bestandsbücher').toBe(true);
	});

	it('vergibt das Wort „Berichte" nur einmal — der Reiter im Bestellwesen heißt „Bestellberichte"', () => {
		// Der Grund des ganzen Umbaus: Es gab zwei Orte namens „Berichte" — die Sektion in
		// der Seitenleiste und den Reiter im Bestellwesen. Wer die Bestandsbücher suchte,
		// vermutete sie bei den Bestellberichten, obwohl dort über BESTELLUNGEN gerechnet
		// wird (Zeitraum, Lieferant, Topf) und im Zugangsbuch auch Exemplare ohne jede
		// Bestellung stehen.
		//
		// M3 zu Reitern: „As a set, all tabs are unified by a shared topic" (Tabs,
		// Guidelines). Die Leiste trägt das Etikett „Bereiche des Bestellwesens" — also
		// nennt der Reiter sein Thema.
		let verzeichnis = process.cwd();
		while (!existsSync(resolve(verzeichnis, 'src/lib/BestellWorkspace.svelte'))) {
			const eltern = dirname(verzeichnis);
			if (eltern === verzeichnis) throw new Error('BestellWorkspace.svelte nicht gefunden');
			verzeichnis = eltern;
		}
		const quelle = readFileSync(resolve(verzeichnis, 'src/lib/BestellWorkspace.svelte'), 'utf8');

		// Nicht-leer-Garantie: Ohne sie wäre der Test auch dann grün, wenn die Reiterliste
		// umgebaut wird und beide Muster ins Leere greifen.
		expect(quelle).toMatch(/id: 'berichte'/);
		expect(quelle).toMatch(/label: 'Bestellberichte'/);
		expect(quelle).not.toMatch(/label: 'Berichte'/);
		expect(menuGroups.filter((g) => g.name === 'Berichte')).toHaveLength(1);
	});

	it('lässt ohne Anmeldung nichts durch', () => {
		expect(allePunkte.every((item) => canSeeItem(item, null))).toBe(false);
		expect(allePunkte.some((item) => canSeeItem(item, null))).toBe(false);
	});
});
