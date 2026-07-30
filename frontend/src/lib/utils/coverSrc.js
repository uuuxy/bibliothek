// utils/coverSrc.js
// Bildquelle für ein Cover — die Entscheidung, die vorher an vier Stellen einzeln
// (und falsch) getroffen wurde.
//
// Der Cover-Proxy (/api/images/cover) ist NICHT für jede Cover-URL zuständig. Er lädt
// ein Bild von einem fremden Server, wandelt es in WebP und legt es unter uploads/covers
// ab. Seine Host-Allowlist (api/image_caching.go) kennt nur DNB, OpenLibrary und Google
// Books — bei allem anderen, insbesondere bei einem lokalen "/uploads/…"-Pfad, liefert er
// ein transparentes 1×1-GIF aus (bewusst 200 statt 404, gegen Konsolen-Spam).
//
// Genau das ist der Normalfall: Der Cover-Sync migriert jedes gefundene Cover auf einen
// lokalen Pfad, damit nichts mehr extern lädt (internal/service/cover_service.go). Wer
// solche Titel durch den Proxy schickt, bekommt also zuverlässig ein leeres Bild —
// sichtbar geworden als "Kein Coverbild hinterlegt" in der Großansicht, obwohl das
// Miniaturbild derselben Zeile das Cover zeigte (es lädt den Pfad direkt).

/**
 * Liefert die anzeigbare Bildquelle für ein Cover, oder '' wenn keine ermittelbar ist.
 * '' heißt für den Aufrufer: Platzhalter zeigen, kein <img> rendern.
 *
 * @param {string} [coverUrl] Wert aus buecher_titel.cover_url — lokal ("/uploads/…") oder extern ("https://…")
 * @param {string} [isbn] Cache-Schlüssel des Proxys; ohne ISBN weist er die Anfrage ab
 * @returns {string}
 */
export function coverSrc(coverUrl, isbn) {
	const url = (coverUrl ?? '').trim();
	if (!url) return '';

	// Lokal abgelegtes Cover: direkt ausliefern. Der Umweg über den Proxy wäre nicht nur
	// überflüssig (die Datei liegt schon als WebP da), er scheitert.
	if (url.startsWith('/')) return url;

	// Extern: nur über den Proxy, nie per Hotlink — und nur mit ISBN, sonst antwortet er
	// mit dem 1×1-GIF und der Aufrufer zeigt statt des Platzhalters ein leeres Kästchen.
	const key = (isbn ?? '').trim();
	if (!key) return '';
	return `/api/images/cover?isbn=${encodeURIComponent(key)}&url=${encodeURIComponent(url)}`;
}
