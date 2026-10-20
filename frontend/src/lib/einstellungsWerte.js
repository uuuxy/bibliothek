// Zahlenfelder der Einstellungen.
//
// Ein geleertes <input type="number"> liefert über bind:value null oder ''. Es gilt eine
// Regel: Was im Feld steht, wird gespeichert, und ein leeres Zahlenfeld ist unvollständig
// und wird gemeldet, statt geraten. Mit `Number(null) || 0` würde daraus still eine 0: Wer
// die 90 der Lesehistorie-Befristung löscht, um sie neu zu tippen, und zwischendurch
// speichert, hätte die Befristung abgeschaltet, ohne es zu merken.

/**
 * Liest ein Zahlenfeld aus. `null` heißt LEER oder unbrauchbar.
 * @param {unknown} v Eingabewert aus dem Feld (Zahl, String, null, undefined)
 * @returns {number | null} ganze Zahl ≥ 0, oder null
 */
export function zahlOderLeer(v) {
	// Nur die zwei Typen, die ein <input type="number"> über bind:value liefern kann: eine
	// Zahl, oder eine Zeichenkette (leer bei geleertem Feld). Kein String(v) über alles: Ein
	// Objekt würde dort still zu '[object Object]' und über NaN zu null, das richtige
	// Ergebnis aus dem falschen Grund. Der Typ-Zweig sagt stattdessen aus, was hier
	// überhaupt ankommen darf (SonarQube javascript:S6551).
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

/**
 * Sammelt die Zahlenfelder einer Kategorie für den Patch und meldet, welche unbrauchbar
 * sind — mit ihrer Beschriftung, damit die Meldung auf das Feld zeigt und nicht auf
 * einen API-Schlüssel.
 *
 * `min` ist nicht nur Kosmetik: Das Backend ersetzt einen Wert unterhalb der Grenze
 * durch die Vorgabe (0 in „Tage / Buch" wird zu 21). Ohne diese Prüfung tippte man 0,
 * bekämte eine Erfolgsmeldung und fände danach 21 im Feld — die Regel „was im Feld
 * steht, wird gespeichert" gälte an genau dieser Stelle nicht. Bei den Feldern, in
 * denen die 0 „aus" BEDEUTET (Lesehistorie, Sperrbildschirm, Tage bis Sperre), steht
 * `min: 0`, und dann ist sie ein Wert wie jeder andere.
 *
 * @param {{ schluessel: string, label: string, wert: unknown, min?: number }[]} felder
 * @returns {{ werte: Record<string, number>, fehlend: string[] }}
 */
export function sammleZahlen(felder) {
	/** @type {Record<string, number>} */
	const werte = {};
	/** @type {string[]} */
	const fehlend = [];
	for (const f of felder) {
		const n = zahlOderLeer(f.wert);
		const min = f.min ?? 0;
		if (n === null) fehlend.push(f.label);
		else if (n < min) fehlend.push(`${f.label} (mindestens ${min})`);
		else werte[f.schluessel] = n;
	}
	return { werte, fehlend };
}
