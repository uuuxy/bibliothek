/**
 * @typedef {Object} MenuItem
 * @property {string} id
 * @property {string} label
 * @property {string} icon
 * @property {string} [permission]
 * @property {string[]} [roles]
 */

/**
 * Prüft ein einzelnes Recht. Die EINE Stelle, an der die Regel steht: Admin darf
 * alles, sonst entscheidet die Rechteliste aus role_permissions ('*' = alles).
 *
 * Bewusst geteilt mit canSeeItem — eine Ansicht, die ihre Rechteprüfung selbst
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
 * Determines whether a menu item should be visible for the given user.
 *
 * @param {MenuItem} item
 * @param {any} currentUser
 * @returns {boolean}
 */
export function canSeeItem(item, currentUser) {
	if (!currentUser) return false;

	const r = (currentUser.rolle || '').toLowerCase();

	// Admin hat implizit alle Rechte — und zwar VOR allen Sonderregeln.
	//
	// Diese Zeile stand bis zum 09.08.2026 unter der Portal-Ausnahme darunter. Damit galt
	// die Ausnahme auch für Admins: Sie durften die Reservierungs-Endpunkte aufrufen (das
	// Backend hat den Admin-Vorrang eingebaut), sahen den Menüpunkt aber nicht — und
	// fanden die Klassensatz-Reservierung deshalb schlicht nicht. Ein Widerspruch
	// zwischen Oberfläche und Rechtelage, kein gewollter Schutz.
	if (r === 'admin') return true;

	// Das Portal ist für das Kollegium bestimmt (Rolle hieß bis Migration 069 'lehrer').
	if (item.id === 'kollegium_portal') {
		return r === 'kollegium';
	}

	// Eine roles-Liste schränkt EIN, sie ist keine Alternative zur Rechteprüfung.
	//
	// Diese Prüfung stand bis zum 10.08.2026 im Zweig darunter und galt deshalb nur für
	// Punkte OHNE permission. „Einstellungen" hat beides — roles: ['admin'] und
	// permission: 'manage_users' —, also sprang die Funktion direkt zu hatRecht() und die
	// Rollenliste war toter Code. Jede Rolle, der ein Admin manage_users erteilt, sah
	// damit die Systemeinstellungen; auf dem Schulserver traf das die Rolle 'kollegium'.
	if (item.roles && !item.roles.includes(r)) return false;

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

// Unteransichten ohne eigenen Menüpunkt erben die Regel ihrer Elternansicht.
//
// Der Router nahm sie bis zum 10.08.2026 ganz von der Prüfung aus, weil ein Sprung in die
// Buchakte sonst als Rauswurf geendet hätte. Nur galt die Ausnahme auch für die URL-Zeile:
// /medienkatalog/buch/{id} und /statistiken/renner öffneten sich JEDEM angemeldeten
// Benutzer. Daten flossen dabei keine — die Endpunkte dahinter hängen an view_books bzw.
// view_students und antworten mit 403 —, aber eine Lehrkraft landete auf einer leeren,
// defekt wirkenden Seite statt in ihrem Portal.
/** @type {Record<string, string>} */
const unteransichtBrauchtTab = {
	book_detail: 'media_catalog',
	stats_detail: 'stats'
};

/**
 * Die Tab-IDs, die dieser Benutzer laut Navigation öffnen darf.
 *
 * @param {any} currentUser
 * @returns {Set<string>}
 */
export function erlaubteTabs(currentUser) {
	return new Set(
		menuGroups
			.flatMap((g) => g.items)
			.filter((i) => canSeeItem(i, currentUser))
			.map((i) => i.id)
	);
}

/**
 * Darf dieser Bildschirm NICHT geöffnet werden? Beantwortet dieselbe Frage wie canSeeItem,
 * nur für einen Tab statt für einen Menüpunkt — bewusst hier und nicht im Router, damit es
 * nicht wieder zwei Definitionen davon gibt, was eine Rolle erreichen darf.
 *
 * @param {string} tab
 * @param {Set<string>} erlaubt Ergebnis von erlaubteTabs
 * @returns {boolean}
 */
export function tabIstGesperrt(tab, erlaubt) {
	const elternTab = unteransichtBrauchtTab[tab];
	if (elternTab) return !erlaubt.has(elternTab);

	const menuIDs = new Set(menuGroups.flatMap((g) => g.items).map((i) => i.id));
	return menuIDs.has(tab) && !erlaubt.has(tab);
}

export const menuGroups = [
	{
		name: 'Kiosk',
		items: [
			{ id: 'kiosk', label: 'Ausleihe', icon: 'kiosk' },
			{ id: 'mahnwesen', label: 'Mahnwesen', icon: 'bell', permission: 'manage_users' }
		]
	},
	{
		name: 'Bibliothek',
		items: [
			{ id: 'media_catalog', label: 'Medienkatalog', icon: 'catalog', permission: 'view_books' },
			{ id: 'signaturen', label: 'Signaturen', icon: 'book', permission: 'view_books' },
			{ id: 'druck-center', label: 'Druck-Center', icon: 'printer', permission: 'view_students' }
		]
	},
	{
		name: 'Verwaltung',
		items: [
			{ id: 'students_dir', label: 'Schülerdatei', icon: 'users', permission: 'view_students' },
			{
				// view_books, nicht manage_users: Bis zum 08.08.2026 gab es dieselbe Übersicht
				// ein zweites Mal als Reiter im Medienkatalog, und der stand jedem mit
				// view_books offen. Dieser Punkt hat den Reiter abgelöst — mit manage_users
				// (faktisch Administrator, siehe 940d7d8) hätte die Zusammenlegung den
				// Bibliotheks-Helfern den Blick auf die Klassensätze genommen. Die
				// Verwaltungsaktionen darin hängen an edit_books, geprüft in KlassenUebersicht.
				id: 'schulklassen',
				label: 'Schulklassen',
				icon: 'identification',
				permission: 'view_books'
			},
			{ id: 'graduates', label: 'Abgänger', icon: 'academic-cap', permission: 'view_graduates' },
			{ id: 'orders', label: 'Bestellungen', icon: 'shopping-bag', permission: 'view_orders' },
			{ id: 'inventory', label: 'Inventur', icon: 'clipboard', permission: 'inventory_scan' }
		]
	},
	{
		name: 'System',
		items: [
			{ id: 'stats', label: 'Statistiken', icon: 'chart-bar', permission: 'view_stats' },
			{ id: 'system-logs', label: 'System-Logs', icon: 'shield', permission: 'audit_logs' },
			{
				// Aus „Verwaltung" hierher verschoben (09.08.2026, auf Peters Ansage): Die
				// Massenverlaengerung wird "fast nie" gebraucht, weil der LUSD-Import
				// massgebend ist — er setzt Klasse und Abgaengerstatus, und daran haengen
				// Fristen und Ausleihlimits ohnehin. In „Verwaltung" stand das Werkzeug
				// zwischen den taeglichen Punkten (Schuelerdatei, Bestellungen, Inventur)
				// und war dort praesenter als sein Nutzen. Das Recht bleibt unveraendert
				// manage_inventory — die Gruppe entscheidet nur ueber die Einordnung im
				// Menue, nicht darueber, wer den Punkt sieht (siehe canSeeItem oben).
				id: 'lmf_actions',
				label: 'LMF-Aktionen',
				icon: 'clock',
				permission: 'manage_inventory'
			},
			{
				// Die Selbstprüfung sagt, was eingerichtet, aber nicht in Betrieb ist.
				//
				// Dasselbe Recht wie die Einstellungen, aus zwei Gründen: Wer die Lücken sieht,
				// soll sie auch schließen können — eine Meldung ohne Handhabe ist nur Verdruss.
				// Und die Befunde beschreiben die Angriffsfläche der Anlage (Beispiel-Geheimnisse,
				// mock-Anmeldung, fehlende Auslagerung), nicht die Bibliothek.
				//
				// Der Eintrag hier ist nicht nur Bequemlichkeit: tabIstGesperrt() unten kennt nur
				// Bildschirme, die im Menü stehen. Ohne diese Zeile wäre /betriebsbereitschaft für
				// JEDEN Angemeldeten per URL zu öffnen — Daten flössen keine (der Endpunkt hängt
				// an manage_users), aber ein Helfer landete auf einer leeren, defekt wirkenden
				// Seite. Genau diese Lücke stand schon einmal bei book_detail und stats_detail
				// offen, siehe unteransichtBrauchtTab.
				id: 'betriebsbereitschaft',
				label: 'Betriebsbereitschaft',
				icon: 'shield',
				permission: 'manage_users',
				roles: ['admin']
			},
			{
				// Aus den Einstellungen herausgezogen (Betreiber-Entscheidung 16.08.2026):
				// Die Rechte-Verwaltung ist der Ort, auf den die Drift-Warnung der
				// Betriebsbereitschaft zeigt — als Tab in einer sechsteiligen
				// Einstellungsseite war sie dafür zu tief vergraben.
				id: 'berechtigungen',
				label: 'Berechtigungen',
				icon: 'key',
				permission: 'manage_users',
				roles: ['admin']
			},
			{
				id: 'settings',
				label: 'Einstellungen',
				icon: 'cog',
				permission: 'manage_users',
				roles: ['admin']
			}
		]
	},
	{
		name: 'Kollegium',
		items: [{ id: 'kollegium_portal', label: 'Mein Portal', icon: 'book', roles: ['kollegium'] }]
	}
];
