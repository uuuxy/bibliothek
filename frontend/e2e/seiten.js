// Die Seiten der Navigation (src/lib/routenTabelle.js) für die messenden Gates.
//
// Ohne die Bestandsbücher: Ihre Liste ist am Entwicklungsrechner 67.000 Zeilen lang, und schon
// das erste Messen dauert dort länger als das Warten auf Ruhe. Symbolknöpfe und Textfelder
// stehen dort nur im Rahmen der Anwendung und im Zeitraum (zwei Datumsfelder fester Breite).
export const SEITEN = /** @type {const} */ ([
	['Ausleihe', '/kiosk'],
	['Medienkatalog', '/medienkatalog'],
	['Signaturen', '/signaturen'],
	['Druck-Center', '/druck-center'],
	['Klassensätze', '/schulklassen'],
	['Leserdatei', '/schuelerdatei'],
	['Mahnwesen', '/mahnwesen'],
	['Abgänger', '/abgaenger'],
	['Bestellungen', '/bestellungen'],
	['Inventur', '/inventur'],
	['Statistiken', '/statistiken'],
	['Bestellberichte', '/bestellberichte'],
	['System-Logs', '/system-logs'],
	['Schuljahreswechsel', '/schuljahr'],
	['Benutzer & Rechte', '/berechtigungen'],
	['Einstellungen', '/einstellungen'],
	['Mein Portal', '/kollegium-portal']
]);
