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

	// Das Lehrer-Portal ist ausschließlich für Lehrer bestimmt.
	if (item.id === 'lehrer_portal') {
		return r === 'lehrer';
	}

	// Admin hat implizit alle Rechte.
	if (r === 'admin') return true;

	// Punkte ohne Permission-Anforderung sind allgemeine Theken-Werkzeuge (z. B. Kiosk).
	if (!item.permission) {
		if (item.roles) return item.roles.includes(r);
		// Lehrer sehen nur ihr Portal + ausdrücklich freigeschaltete Rechte, keine
		// allgemeinen Werkzeuge ohne Rechtebezug.
		return r !== 'lehrer';
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
			{ id: 'inventory', label: 'Inventur', icon: 'clipboard', permission: 'inventory_scan' },
			{ id: 'lmf_actions', label: 'LMF-Aktionen', icon: 'clock', permission: 'manage_inventory' }
		]
	},
	{
		name: 'System',
		items: [
			{ id: 'stats', label: 'Statistiken', icon: 'chart-bar', permission: 'view_stats' },
			{ id: 'system-logs', label: 'System-Logs', icon: 'shield', permission: 'audit_logs' },
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
		name: 'Lehrer',
		items: [{ id: 'lehrer_portal', label: 'Mein Portal', icon: 'book', roles: ['lehrer'] }]
	}
];
