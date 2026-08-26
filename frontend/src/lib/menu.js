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

	// Eine roles-Liste schränkt EIN, sie ist keine Alternative zur Rechteprüfung.
	//
	// Diese Prüfung stand bis zum 10.08.2026 im Zweig darunter und galt deshalb nur für
	// Punkte OHNE permission. „Einstellungen" hat beides — roles: ['admin'] und
	// permission: 'manage_users' —, also sprang die Funktion direkt zu hatRecht() und die
	// Rollenliste war toter Code. Jede Rolle, der ein Admin manage_users erteilt, sah
	// damit die Systemeinstellungen; auf dem Schulserver traf das die Rolle 'kollegium'.
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
		items: [{ id: 'kiosk', label: 'Ausleihe', icon: 'kiosk' }]
	},
	{
		name: 'Bibliothek',
		items: [
			{ id: 'media_catalog', label: 'Medienkatalog', icon: 'catalog', permission: 'view_books' },
			{ id: 'signaturen', label: 'Signaturen', icon: 'book', permission: 'view_books' },
			{ id: 'druck-center', label: 'Druck-Center', icon: 'printer', permission: 'view_students' },
			{
				// view_books, nicht manage_users: Bis zum 08.08.2026 gab es dieselbe Übersicht
				// ein zweites Mal als Reiter im Medienkatalog, und der stand jedem mit
				// view_books offen. Dieser Punkt hat den Reiter abgelöst — mit manage_users
				// (faktisch Administrator, siehe 940d7d8) hätte die Zusammenlegung den
				// Bibliotheks-Helfern den Blick auf die Klassensätze genommen. Die
				// Verwaltungsaktionen darin hängen an edit_books, geprüft in KlassenUebersicht.
				id: 'schulklassen',
				// „Klassensätze“ statt „Schulklassen“ (24.08.2026): Die Seite zeigt Bücher je
				// Klasse, keine Klassen — und „Klasse 5“ stand damit zweimal in der Oberfläche
				// (Medienkatalog → Jahrgänge). Route und id bleiben.
				label: 'Klassensätze',
				icon: 'identification',
				permission: 'view_books'
			}
		]
	},
	{
		name: 'Verwaltung',
		items: [
			{ id: 'students_dir', label: 'Schülerdatei', icon: 'users', permission: 'view_students' },
			{
				// Seit 24.08.2026 unter Verwaltung (vorher Kiosk): Mahnen ist Schülerarbeit,
				// keine Thekenarbeit.
				// view_students, nicht manage_users: Alle Mahnwesen-Routen (/api/mahnwesen*,
				// /api/print/mahnung/…) verlangen view_students. Bis zum 24.08.2026 forderte
				// der Menüpunkt manage_users — wer mahnen durfte, sah den Punkt nicht, und wer
				// ihn sah, brauchte das Recht dafür gar nicht. Dieselbe Klasse wie view_stats
				// (23.08.), geprüft in api/rechte_paritaet_test.go.
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
		name: 'System',
		items: [
			{ id: 'stats', label: 'Statistiken', icon: 'chart-bar', permission: 'view_stats' },
			{ id: 'system-logs', label: 'System-Logs', icon: 'shield', permission: 'audit_logs' },
			// „LMF-Aktionen" stand hier bis zum 24.08.2026 als eigener Punkt (zuvor in
			// „Verwaltung"). Auf Peters Ansage in die Einstellungen gewandert — die
			// Massenverlaengerung wird "fast nie" gebraucht, weil der LUSD-Import
			// massgebend ist. Damit sieht sie nur noch, wer die Einstellungen sieht.
			{
				// Aus den Einstellungen herausgezogen (Betreiber-Entscheidung 16.08.2026):
				// Die Rechte-Verwaltung ist der Ort, auf den die Drift-Warnung der
				// Betriebsbereitschaft zeigt — als Tab in einer sechsteiligen
				// Einstellungsseite war sie dafür zu tief vergraben.
				// Kein roles: ['admin'] mehr (seit 24.08.2026): Ein Rollen-Pin neben dem Recht
				// machte das Recht zum Schalter ohne Wirkung — wer manage_users bekam, sah den
				// Punkt trotzdem nicht. Die Sorge dahinter (10.08.: Kollegium sah die
				// Einstellungen) war ein zu grobes Recht; das ist seit der Aufteilung in
				// manage_users / manage_settings / manage_students_admin gelöst.
				id: 'berechtigungen',
				label: 'Benutzer & Rechte',
				icon: 'key',
				permission: 'manage_users'
			},
			{
				// Ein Sammelpunkt: Die Kategorien dahinter hängen an verschiedenen Rechten
				// (kategorien.js) — Schule/Fristen/Mail an manage_settings, LMF-Aktionen an
				// edit_books, Datenverwaltung an manage_inventory, Schuljahreswechsel an
				// import_students/manage_students_admin. Bis 24.08.2026 abends öffnete nur
				// manage_settings die Tür: Ein Mitarbeiter hatte ab Werk import_students und
				// manage_inventory, kam aber nie an LUSD-Import oder Littera-Import heran —
				// Rechte ohne Tür. Die Liste hier MUSS die Rechte aus kategorien.js spiegeln
				// (Gate: menu.test.js).
				id: 'settings',
				label: 'Einstellungen',
				icon: 'cog',
				permissions: [
					'manage_settings',
					'edit_books',
					'manage_inventory',
					'import_students',
					'manage_students_admin',
					// Lieferanten-Kategorie (25.08.2026) — Schreiben verlangt create_orders.
					'create_orders'
				]
			}
		]
	},
	{
		name: 'Kollegium',
		// Am RECHT, nicht an der Rolle (Peter, 26.08.2026): Bis dahin stand hier
		// roles: ['kollegium'] — eine Lehrkraft, die in Bibliothek/LMF mitarbeitet und
		// deshalb als Mitarbeiter angelegt ist, fand das Portal nicht, obwohl der Server
		// sie (create_reservations) überall hineinließ. Zwei Wahrheitsquellen, die nur
		// zufällig einig waren. Wer reservieren darf, sieht die Tür.
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
