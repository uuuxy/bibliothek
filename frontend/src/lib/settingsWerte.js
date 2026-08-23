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
	// Nur die zwei Typen, die ein <input type="number"> über bind:value liefern kann: eine
	// Zahl, oder eine Zeichenkette (leer bei geleertem Feld). Bewusst KEIN String(v) über
	// alles: Ein Objekt würde dort still zu '[object Object]' und über NaN zu null — das
	// richtige Ergebnis aus dem falschen Grund. Der Typ-Zweig sagt stattdessen aus, was hier
	// überhaupt ankommen darf (SonarQube javascript:S6551, 22.08.2026).
	let n;
	if (typeof v === 'number') {
		n = v;
	} else if (typeof v === 'string') {
		const roh = v.trim();
		if (roh === '') return null;
		n = Number(roh);
	} else {
		return null;
	}
	if (!Number.isFinite(n) || n < 0) return null;
	return Math.trunc(n);
}
