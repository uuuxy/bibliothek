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
