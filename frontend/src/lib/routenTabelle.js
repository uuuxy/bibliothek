/**
 * @file routenTabelle.js
 * Die Zuordnung Bildschirm → Adresse. EINE Quelle, und zwar nur diese.
 *
 * Bewusst nur einmal definiert: Vorher lag dieselbe Tabelle doppelt in Router.svelte — im
 * Routing-Effekt und im popstate-Handler. Ein neu ergänzter Bildschirm (das
 * Kollegiums-Portal) wurde in einer der beiden Kopien vergessen, seine Adresse also nie
 * gesetzt und nach einem Neuladen nicht wiederhergestellt: Die Lehrkraft flog aus ihrem
 * Portal.
 *
 * Seit dem 17.09.2026 steht sie in einer eigenen Datei statt im Router. Das ist keine
 * Kosmetik, sondern die Antwort auf die Datei-Ratsche (frontend-hygiene-dateigroesse):
 * Router.svelte ist eine geduldete Datei über 200 Zeilen, und eine geduldete Datei darf
 * nicht weiter wachsen. Eine Tabelle ist Daten und gehört ohnehin nicht in eine
 * Komponente — jeder neue Bildschirm hätte den Router sonst um eine weitere Zeile gedehnt.
 *
 * `media_catalog` liegt auf /medienkatalog und NICHT auf dem Pfad des öffentlichen OPAC.
 * Solange beide denselben beanspruchten, landete ein angemeldeter Benutzer nach F5 im
 * öffentlichen Katalog — und die UI-Gates (control-hoehen, icon-trefferflaechen) vermaßen
 * still den OPAC statt des internen Katalogs (Audit-Befund vom 01.08.2026).
 *
 * Parametrisierte Sonderrouten (Buchakte, Statistik-Detail) stehen NICHT hier: Sie tragen
 * einen Wert im Pfad und werden im Router eigens behandelt.
 *
 * @type {Record<string, string>}
 */
export const tabToPath = {
	settings: '/einstellungen',
	inventory: '/inventur',
	students_dir: '/schuelerdatei',
	schulklassen: '/schulklassen',
	orders: '/bestellungen',
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
