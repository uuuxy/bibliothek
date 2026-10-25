import { DRUCK_RECHTE } from './components/etiketten/druckCenterReiter.js';

/**
 * @typedef {Object} MenuItem
 * @property {string} id
 * @property {string} label
 * @property {string} icon
 * @property {string} [permission]
 * @property {string[]} [permissions] - EINES davon genügt (Sammelpunkt wie „Einstellungen").
 * @property {string[]} [roles]
 */

/**
 * Prüft ein einzelnes Recht. Die EINE Stelle, an der die Regel steht: Admin darf
 * alles, sonst entscheidet die Rechteliste aus role_permissions ('*' = alles).
 *
 * Bewusst geteilt mit darfSehen — eine Ansicht, die ihre Rechteprüfung selbst
 * nachbaut, weicht beim nächsten Rollen-Feinschliff still von der Navigation ab:
 * Der Menüpunkt wäre dann sichtbar und die Aktion darin gesperrt, oder umgekehrt.
 *
 * @param {any} currentUser
 * @param {string} permission
 * @returns {boolean}
 */
export function hatRecht(currentUser, permission) {
	if (!currentUser) return false;
	if ((currentUser.rolle || '').toLowerCase() === 'admin') return true;
	const perms = currentUser.permissions || [];
	return perms.includes('*') || perms.includes(permission);
}

/**
 * Ob der Menüpunkt für diesen Benutzer sichtbar ist.
 * @param {MenuItem} item
 * @param {any} currentUser
 * @returns {boolean}
 */
export function darfSehen(item, currentUser) {
	if (!currentUser) return false;

	const r = (currentUser.rolle || '').toLowerCase();

	// Admin hat implizit alle Rechte, und zwar vor allen Sonderregeln: Sonst gälte die
	// Portal-Ausnahme darunter auch für Admins, und sie fänden die Klassensatz-Reservierung
	// nicht, obwohl das Backend sie hineinlässt.
	if (r === 'admin') return true;

	// Eine roles-Liste schränkt ein, sie ist keine Alternative zur Rechteprüfung, und sie gilt
	// auch für Punkte mit permission: Sonst sähe jede Rolle, der ein Admin das Recht eines
	// Punkts erteilt, auch einen Punkt, den roles auf Admins begrenzt.
	if (item.roles && !item.roles.includes(r)) return false;

	// Sammelpunkt: sichtbar, sobald EINE der genannten Türen dahinter offen ist.
	if (item.permissions) return item.permissions.some((p) => hatRecht(currentUser, p));

	// Punkte ohne Permission-Anforderung sind allgemeine Theken-Werkzeuge (z. B. Kiosk).
	if (!item.permission) {
		// Ausdrücklich für diese Rolle gelistet — die Liste IST hier die Freigabe.
		if (item.roles) return true;
		// Das Kollegium sieht nur sein Portal + ausdrücklich freigeschaltete Rechte,
		// keine allgemeinen Werkzeuge ohne Rechtebezug.
		return r !== 'kollegium';
	}

	return hatRecht(currentUser, item.permission);
}

// Unteransichten ohne eigenen Menüpunkt erben die Regel ihrer Elternansicht. Ganz ohne
// Prüfung öffneten sich /medienkatalog/buch/{id} und /statistiken/renner über die URL-Zeile
// jedem angemeldeten Benutzer; Daten flössen keine (die Endpunkte antworten mit 403), aber
// eine Lehrkraft landete auf einer leeren, defekt wirkenden Seite statt in ihrem Portal.
/** @type {Record<string, string>} */
const unteransichtBrauchtBildschirm = {
	book_detail: 'media_catalog',
	stats_detail: 'stats'
};

/**
 * Die Tab-IDs, die dieser Benutzer laut Navigation öffnen darf.
 * @param {any} currentUser
 * @returns {Set<string>}
 */
export function erlaubteBildschirme(currentUser) {
	return new Set(
		menueGruppen
			.flatMap((g) => g.items)
			.filter((i) => darfSehen(i, currentUser))
			.map((i) => i.id)
	);
}

/**
 * Darf dieser Bildschirm NICHT geöffnet werden? Beantwortet dieselbe Frage wie darfSehen,
 * nur für einen Tab statt für einen Menüpunkt — bewusst hier und nicht im Router, damit es
 * nicht wieder zwei Definitionen davon gibt, was eine Rolle erreichen darf.
 *
 * @param {string} tab
 * @param {Set<string>} erlaubt Ergebnis von erlaubteBildschirme
 * @returns {boolean}
 */
export function bildschirmGesperrt(tab, erlaubt) {
	const elternBildschirm = unteransichtBrauchtBildschirm[tab];
	if (elternBildschirm) return !erlaubt.has(elternBildschirm);

	const menuepunkte = new Set(menueGruppen.flatMap((g) => g.items).map((i) => i.id));
	return menuepunkte.has(tab) && !erlaubt.has(tab);
}

/**
 * Titel eines Bildschirms für die (unsichtbare) Seitenüberschrift im Router — aus dem
 * Menü, damit es nicht zwei Listen von Bildschirmnamen gibt. Unteransichten, die kein
 * Menüpunkt sind, stehen hier mit eigenem Namen.
 * @param {string} tab
 * @returns {string}
 */
export function bildschirmTitel(tab) {
	for (const g of menueGruppen) for (const i of g.items) if (i.id === tab) return i.label;
	return { book_detail: 'Buchakte', stats_detail: 'Statistik im Detail' }[tab] ?? 'Bibliothek';
}

export const menueGruppen = [
	{
		name: 'Kiosk',
		items: [{ id: 'kiosk', label: 'Ausleihe', icon: 'kiosk' }]
	},
	{
		name: 'Bibliothek',
		items: [
			{ id: 'media_catalog', label: 'Medienkatalog', icon: 'catalog', permission: 'view_books' },
			{ id: 'signaturen', label: 'Signaturen', icon: 'book', permission: 'view_books' },
			{ id: 'druck-center', label: 'Druck-Center', icon: 'printer', permissions: DRUCK_RECHTE },
			{
				// view_books, nicht manage_users: Wer den Bestand sehen darf, sieht auch die
				// Klassensätze, auch die Bibliotheks-Helfer. Die Verwaltungsaktionen darin
				// hängen an edit_books, geprüft in KlassenUebersicht.
				id: 'schulklassen',
				// „Klassensätze“ statt „Schulklassen“: Die Seite zeigt Bücher je Klasse, keine
				// Klassen. Route und id bleiben.
				label: 'Klassensätze',
				icon: 'identification',
				permission: 'view_books'
			}
		]
	},
	{
		name: 'Verwaltung',
		items: [
			// „Leserdatei", nicht „Schülerdatei": Seit Migration 125 stehen Schüler und
			// Kollegium in einer Tabelle, und die Liste zeigt beide. Das Recht bleibt
			// view_students — es ist dasselbe Verzeichnis, nur vollständig.
			{ id: 'students_dir', label: 'Leserdatei', icon: 'users', permission: 'view_students' },
			{
				// Unter Verwaltung, nicht an der Theke: Mahnen ist Schülerarbeit, keine
				// Thekenarbeit. view_students wie die Mahnliste (/api/mahnwesen): Mit einem
				// anderen Recht als seine Route sähe, wer mahnen darf, den Punkt nicht. Geprüft
				// in api/rechte_paritaet_test.go.
				id: 'mahnwesen',
				label: 'Mahnwesen',
				icon: 'bell',
				permission: 'view_students'
			},
			{ id: 'graduates', label: 'Abgänger', icon: 'academic-cap', permission: 'view_graduates' },
			{ id: 'orders', label: 'Bestellungen', icon: 'shopping-bag', permission: 'view_orders' },
			{ id: 'inventory', label: 'Inventur', icon: 'clipboard', permission: 'inventory_scan' }
		]
	},
	{
		// Eine eigene Sektion: Im Sammeltopf „System", der beim Laden zugeklappt ist
		// (Seitenleiste.svelte), sähe niemand einen Nachweis oder eine Zahl, ohne erst eine
		// Gruppe aufzuklappen, deren Name nichts davon ankündigt. M3 nennt genau dieses
		// Mittel: „Short subhead section labels can help group related destinations in the
		// navigation drawer" (Navigation drawer, Guidelines).
		//
		// Unter „Berichte" steht, was ein Blatt für jemand anderen erzeugt und dabei nichts
		// ändert. Bei seinem Vorgang bleibt, was eine Folgeaktion hat: der
		// Fehlbestandsbericht (Knöpfe „Gefunden" und Löschen) in der Inventur, die Mahnungen
		// im Mahnwesen.
		name: 'Berichte',
		items: [
			{ id: 'stats', label: 'Statistiken', icon: 'chart-bar', permission: 'view_stats' },
			{
				// Zugangs- und Abgangsbuch, ein Punkt: ein Nachweis zum Stichtag (15.3./15.9.),
				// den jemand ausdruckt und abheftet. Als Reiter im Medienkatalog stünden sie der
				// täglichen Arbeit im Weg.
				//
				// Recht: view_books wie der übrige Blick auf den Bestand — die vier Türen
				// dahinter verlangen dasselbe (api/routen_katalog.go), und ein Menüpunkt, der
				// enger ist als seine Route, schützt nichts: Die Adresse bleibt offen. Wer den
				// Nachweis wirklich auf das Sekretariat begrenzen will, hängt beide Seiten auf
				// manage_inventory um (Helfer und Kollegium haben es ab Werk nicht, db/rechte_vorgabe.go).
				id: 'bestandsbuecher',
				label: 'Bestandsbücher',
				icon: 'book-marked',
				permission: 'view_books'
			},
			{
				// Monats-, Jahresbericht und Lieferantenabrechnung. view_orders wie die Route
				// dahinter (/api/bestellhistorie/bericht): Wer
				// keine Bestellungen sehen darf, sieht hier zwei Einträge statt drei.
				id: 'bestellberichte',
				label: 'Bestellberichte',
				icon: 'file-text',
				permission: 'view_orders'
			}
		]
	},
	{
		name: 'System',
		items: [
			{ id: 'system-logs', label: 'System-Logs', icon: 'shield', permission: 'audit_logs' },
			{
				// „Schuljahreswechsel" = der LMF-Plan: Rückgabe- und Ausgabetermine je Klasse, ein-
				// bis zweimal im Jahr gebraucht, deshalb unter System und nicht dauerhaft im
				// Bibliotheks-Menü. Abgänger stehen unter Verwaltung, der LUSD-Import in den
				// Einstellungen. edit_books wie die übrige Lernmittel-Pflege; das Kollegium liest
				// den Plan im Portal.
				id: 'schuljahr',
				label: 'Schuljahreswechsel',
				icon: 'calendar',
				permission: 'edit_books'
			},
			// Die LMF-Aktionen sind kein eigener Punkt, sondern eine Kategorie der Einstellungen:
			// Die Massenverlängerung wird fast nie gebraucht, weil der LUSD-Import maßgebend ist.
			{
				// Ein eigener Punkt, nicht ein Reiter der Einstellungen: Die Rechte-Verwaltung
				// ist der Ort, auf den die Drift-Warnung der Betriebsbereitschaft zeigt. Kein
				// roles: ['admin']: Ein Rollen-Pin neben dem Recht machte das Recht zum Schalter
				// ohne Wirkung. Die Rechte sind fein genug geteilt (manage_users /
				// manage_settings / manage_students_admin).
				id: 'berechtigungen',
				label: 'Benutzer & Rechte',
				icon: 'key',
				permission: 'manage_users'
			},
			{
				// Ein Sammelpunkt: Die Kategorien dahinter hängen an verschiedenen Rechten
				// (kategorien.js) — Schule/Fristen/Mail an manage_settings, LMF-Aktionen an
				// edit_books, Datenverwaltung an manage_inventory, LUSD & Versetzung (LUSD,
				// Versetzung) an import_students/manage_students_admin. Öffnete nur manage_settings
				// die Tür, hätte ein Mitarbeiter mit import_students oder manage_inventory Rechte
				// ohne Tür. Die Liste hier spiegelt deshalb die Rechte aus kategorien.js
				// (Gate: menue.test.js).
				id: 'settings',
				label: 'Einstellungen',
				icon: 'cog',
				permissions: [
					'manage_settings',
					'edit_books',
					'manage_inventory',
					'import_students',
					'manage_students_admin',
					// Lieferanten-Kategorie; Schreiben verlangt create_orders.
					'create_orders'
				]
			}
		]
	},
	{
		name: 'Kollegium',
		// Am Recht, nicht an der Rolle: Eine Lehrkraft, die in Bibliothek oder LMF mitarbeitet
		// und deshalb als Mitarbeiter angelegt ist, soll das Portal finden, in das der Server
		// sie (create_reservations) hineinlässt. Wer reservieren darf, sieht die Tür.
		items: [
			{
				id: 'kollegium_portal',
				label: 'Mein Portal',
				icon: 'book',
				permission: 'create_reservations'
			}
		]
	}
];
