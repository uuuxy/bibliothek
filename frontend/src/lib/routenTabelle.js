/**
 * @file routenTabelle.js
 * Die Zuordnung Bildschirm → Adresse, eine Quelle für Routing-Effekt und popstate: Fehlt ein
 * Bildschirm in einer von zwei Kopien, wird seine Adresse nie gesetzt und nach einem
 * Neuladen nicht wiederhergestellt. Eine Tabelle ist Daten und gehört nicht in eine
 * Komponente.
 *
 * `media_catalog` liegt auf /medienkatalog und nicht auf dem Pfad des öffentlichen OPAC:
 * Beanspruchten beide denselben, landete ein angemeldeter Benutzer nach F5 im öffentlichen
 * Katalog, und die UI-Gates (control-hoehen, icon-trefferflaechen) vermäßen still den OPAC.
 *
 * Parametrisierte Sonderrouten (Buchakte, Statistik-Detail) stehen nicht hier: Sie tragen
 * einen Wert im Pfad und werden im Router eigens behandelt.
 *
 * @type {Record<string, string>}
 */
export const pfadJeBildschirm = {
	settings: '/einstellungen',
	inventory: '/inventur',
	students_dir: '/schuelerdatei',
	schulklassen: '/schulklassen',
	orders: '/bestellungen',
	bestellberichte: '/bestellberichte',
	media_catalog: '/medienkatalog',
	bestandsbuecher: '/bestandsbuecher',
	signaturen: '/signaturen',
	graduates: '/abgaenger',
	schuljahr: '/schuljahr',
	stats: '/statistiken',
	mahnwesen: '/mahnwesen',
	kollegium_portal: '/kollegium-portal',
	'system-logs': '/system-logs',
	berechtigungen: '/berechtigungen',
	'druck-center': '/druck-center',
	kiosk: '/kiosk'
};
