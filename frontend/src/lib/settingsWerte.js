// Zahlenfelder der Einstellungen, bei denen 0 ein ECHTER Wert ist (= aus).
//
// Ein geleertes <input type="number"> liefert über bind:value null, und `Number(null) || 0`
// machte daraus still die 0 — wer die 90 der Lesehistorie-Befristung „löschen" wollte, um
// sie neu zu tippen, und zwischendurch speicherte, hatte die Befristung abgeschaltet, ohne
// es zu merken (Prüfung 22.08.2026, A4). Leer heißt deshalb: nicht mitschicken — das
// Backend lässt den gespeicherten Wert stehen (Zeigerfeld, nil = unverändert). Nur eine
// ausdrücklich getippte 0 ist „aus".

/**
 * @param {unknown} v Eingabewert aus dem Feld (Zahl, String, null, undefined)
 * @returns {number | null} ganze Zahl ≥ 0, oder null = unverändert lassen
 */
export function zahlOderUnveraendert(v) {
	if (v === null || v === undefined || v === '') return null;
	const n = typeof v === 'number' ? v : Number(String(v).trim());
	if (!Number.isFinite(n) || n < 0) return null;
	return Math.trunc(n);
}
