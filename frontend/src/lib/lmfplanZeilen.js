/** Die Reihenfolge des LMF-Plans als reine Funktionen: Jede nimmt die Zeilen und gibt
 *  neue zurück, nichts wird an Ort und Stelle verändert. LmfPlanReihenfolge ruft sie,
 *  die Tests belegen sie ohne Browser.
 *
 *  Die eine Regel, die den Planer vom Excel unterscheidet (06.09.2026, nach Google
 *  Forms und Slides: Neues kommt nie ans Ende, sondern dorthin, wo man gerade ist):
 *  `einordnen` setzt eine Klasse hinter die letzte Klasse desselben Jahrgangs und
 *  Zweigs — 06F4 hinter 06F3 —, sonst hinter die letzte des Jahrgangs, sonst ans Ende.
 *  Niemand muss eine neue Klasse durch 60 Zeilen ziehen. */

/** @typedef {import('./lmfplanDienst.js').PlanZeile} PlanZeile */

/** Jahrgang und Zweig einer Klasse: „06F4" → { jahrgang: 6, zweig: 'F' }; ohne Jahrgang
 *  („ET1") null — solche Klassen kommen ans Ende.
 *  @param {string} klasse */
export function klassenTeile(klasse) {
	const m = /^\s*0*(\d{1,2})\s*([A-Za-z]*)/.exec(klasse);
	if (!m) return null;
	return { jahrgang: Number(m[1]), zweig: m[2].toUpperCase() };
}

/** Die Zeile, hinter die eine Klasse gehört — oder -1 (ans Ende).
 *  @param {PlanZeile[]} zeilen @param {string} klasse */
export function nachbarZeile(zeilen, klasse) {
	const t = klassenTeile(klasse);
	if (!t) return -1;
	let letzteGleicherZweig = -1;
	let letzteGleicherJahrgang = -1;
	zeilen.forEach((z, i) => {
		for (const k of z.klassen) {
			const kt = klassenTeile(k);
			if (!kt || kt.jahrgang !== t.jahrgang) continue;
			letzteGleicherJahrgang = i;
			if (kt.zweig === t.zweig) letzteGleicherZweig = i;
		}
	});
	return letzteGleicherZweig >= 0 ? letzteGleicherZweig : letzteGleicherJahrgang;
}

/** Fügt eine Klasse als eigene Zeile ein: vor `vor`, wenn gegeben (Ziehen auf eine
 *  Zeile), sonst nach der Nachbar-Regel. Gibt die neuen Zeilen und die Nummer der
 *  neuen Zeile zurück — die Tabelle scrollt dorthin.
 *  @param {PlanZeile[]} zeilen @param {string} klasse @param {number} [vor]
 *  @returns {{ zeilen: PlanZeile[], index: number }} */
export function einordnen(zeilen, klasse, vor) {
	const neu = { klassen: [klasse], vermerk: '', fest: null };
	let index;
	if (vor !== undefined && vor >= 0 && vor <= zeilen.length) index = vor;
	else {
		const nachbar = nachbarZeile(zeilen, klasse);
		index = nachbar >= 0 ? nachbar + 1 : zeilen.length;
	}
	return { zeilen: [...zeilen.slice(0, index), neu, ...zeilen.slice(index)], index };
}

/** @param {PlanZeile[]} zeilen @param {number} von @param {number} nach */
export function verschiebe(zeilen, von, nach) {
	if (von === nach || nach < 0 || nach >= zeilen.length || von < 0 || von >= zeilen.length)
		return zeilen;
	const kopie = [...zeilen];
	const [z] = kopie.splice(von, 1);
	kopie.splice(nach, 0, z);
	return kopie;
}

/** Zeile i mit der davor zusammenlegen: beide Klassen in einer Stunde.
 *  @param {PlanZeile[]} zeilen @param {number} i */
export function zusammenlegen(zeilen, i) {
	if (i <= 0 || i >= zeilen.length) return zeilen;
	const oben = zeilen[i - 1];
	const unten = zeilen[i];
	const vermerk = [oben.vermerk, unten.vermerk].filter(Boolean).join(' · ');
	return [
		...zeilen.slice(0, i - 1),
		{ klassen: [...oben.klassen, ...unten.klassen], vermerk, fest: oben.fest ?? null },
		...zeilen.slice(i + 1)
	];
}

/** Eine Zeile mit mehreren Klassen wieder in einzelne Stunden trennen.
 *  @param {PlanZeile[]} zeilen @param {number} i */
export function trennen(zeilen, i) {
	const z = zeilen[i];
	if (!z || z.klassen.length < 2) return zeilen;
	const einzeln = z.klassen.map((k, n) => ({
		klassen: [k],
		vermerk: n === 0 ? z.vermerk : '',
		fest: n === 0 ? (z.fest ?? null) : null
	}));
	return [...zeilen.slice(0, i), ...einzeln, ...zeilen.slice(i + 1)];
}

/** Eine Zeile ohne Klasse vor i einfügen (i = Länge: anhängen).
 *  @param {PlanZeile[]} zeilen @param {number} i */
export function einfuegen(zeilen, i) {
	return [
		...zeilen.slice(0, i),
		{ klassen: [], vermerk: 'Bücher setzen', fest: null },
		...zeilen.slice(i)
	];
}

/** @param {PlanZeile[]} zeilen @param {number} i */
export function entfernen(zeilen, i) {
	return zeilen.filter((_, n) => n !== i);
}

/** Klasse k aus Zeile i nehmen; bleibt weder Klasse noch Vermerk, fällt die Zeile weg.
 *  @param {PlanZeile[]} zeilen @param {number} i @param {string} k */
export function klasseRaus(zeilen, i, k) {
	const rest = zeilen[i].klassen.filter((x) => x !== k);
	if (rest.length === 0 && !zeilen[i].vermerk.trim()) return entfernen(zeilen, i);
	return zeilen.map((z, n) => (n === i ? { ...z, klassen: rest } : z));
}

/** Festlegen: Die Zeile nimmt ihren Vorschau-Platz als Vorgabe mit, damit „festlegen"
 *  zunächst nichts verschiebt. Lösen: sie fließt wieder mit.
 *  @param {PlanZeile[]} zeilen @param {number} i @param {{ datum: string, stunde: number } | undefined} platz */
export function festWechseln(zeilen, i, platz) {
	// Ohne Platz wird NICHT festgelegt (Rasterdurchgang 06.09.2026): „fest ohne Datum"
	// kannte nur das Frontend — der Server nimmt genau zwei Zustände (fließt, oder fester
	// Platz mit Datum) und antwortet sonst mit 400 „fester Termin braucht ein Datum". Das
	// Fenster ohne Plätze ist echt: nach jedem Laden, bis die erste Vorschau da ist.
	if (!zeilen[i]?.fest && !platz?.datum) return zeilen;
	return zeilen.map((z, n) => {
		if (n !== i) return z;
		if (z.fest) return { ...z, fest: null };
		return { ...z, fest: { datum: platz?.datum ?? '', stunde: platz?.stunde ?? 1 } };
	});
}

/** Klasse `alt` in Zeile i gegen `neu` tauschen — die Zelle überschreiben wie im Excel
 *  (06.09.2026). Steht `neu` schon in der Zeile oder `alt` nicht, bleibt alles.
 *  @param {PlanZeile[]} zeilen @param {number} i @param {string} alt @param {string} neu */
export function klasseTauschen(zeilen, i, alt, neu) {
	const z = zeilen[i];
	if (!z || !z.klassen.includes(alt) || z.klassen.includes(neu)) return zeilen;
	return zeilen.map((x, n) =>
		n === i ? { ...x, klassen: x.klassen.map((k) => (k === alt ? neu : k)) } : x
	);
}
