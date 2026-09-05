/**
 * Welcher Reiter der Seite „Schuljahreswechsel" hinter einem Pfad steckt — oder null,
 * wenn der Pfad nichts damit zu tun hat. /schuljahr/<reiter> ist die Adresse; die alten
 * Menüpunkt-Adressen /abgaenger und /lmf-plan (bis 05.09.2026) führen weiter auf
 * ihren Reiter, damit Lesezeichen halten.
 * @param {string} path
 * @returns {string | null}
 */
export function schuljahrReiterAusPfad(path) {
	if (path === '/abgaenger') return 'abgaenger';
	if (path === '/lmf-plan') return 'lmf-plan';
	if (path.startsWith('/schuljahr/')) return path.slice('/schuljahr/'.length) || null;
	return null;
}
