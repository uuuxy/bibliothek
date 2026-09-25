/**
 * Wie eine Auflage in Listen heißt (docs/OFFEN.md 4.18). Eine Stelle für den Abschnitt der
 * Titelmaske, den Zuordnen-Dialog und den Bestellbedarf: Alle zeigen Titel derselben Reihe,
 * die sich oft nur in Auflage und Jahr unterscheiden — „Lambacher Schweizer 7" zweimal hilft
 * niemandem.
 *
 * Das Feld „Auflage" trägt das Jahr manchmal schon („4. Aufl. 2023"); dann steht es nicht
 * ein zweites Mal daneben.
 *
 * Die Beschriftung steht auch mitten im Satz (Aufschlüsselung, Hinweis der Theke). Deshalb
 * beginnt jede Form, die nicht aus dem Feld kommt, mit einem Substantiv — M3 schreibt
 * Satzschreibung vor („only the first letter of the first word in a sentence or phrase is
 * capitalized"), und ein großes „Ohne" mitten im Satz bräche sie.
 * @param {{ auflage?: string, erscheinungsjahr?: number }} a
 */
export function auflagenBeschriftung(a) {
	const auflage = (a.auflage ?? '').trim();
	const jahr = a.erscheinungsjahr || 0;
	if (auflage && jahr && !auflage.includes(String(jahr))) return `${auflage} · ${jahr}`;
	if (auflage) return auflage;
	if (jahr) return `Ausgabe ${jahr}`;
	return 'Auflage ohne Angabe';
}

/**
 * Die Aufschlüsselung einer Zeile im Bestellbedarf (4.18, Stufe 3): Die Zeile ist das Buch,
 * ihre Zahlen sind die Summe, und hier steht, welche Auflagen sie tragen — in der Reihenfolge
 * des Servers, die neueste zuerst, je mit ihrem Bestand.
 * @param {{ auflage?: string, erscheinungsjahr?: number, gesamt_bestand: number }[]} auflagen
 */
export function auflagenAufschluesselung(auflagen) {
	const teile = auflagen.map((a) => `${auflagenBeschriftung(a)} (${a.gesamt_bestand})`);
	return `Bestand aus ${auflagen.length} Auflagen: ${teile.join(', ')}`;
}

/**
 * Die Hinweiszeile der Theke bei gemischten Auflagen (4.18, Stufe 5): Das eben ausgeliehene
 * Schulbuch ist eine andere Auflage als die, die Kinder derselben Klasse schon haben. Klasse
 * und Zahlen, keine Namen — genug, um das Buch zurückzulegen und die andere Auflage zu holen.
 * @param {{ klasse: string, auflage?: string, erscheinungsjahr?: number,
 *   andere: { auflage?: string, erscheinungsjahr?: number, kinder: number }[] }} h
 */
export function auflagenHinweisText(h) {
	const andere = h.andere
		.map(
			(a) =>
				`${a.kinder} ${a.kinder === 1 ? 'Kind hat' : 'Kinder haben'} ${auflagenBeschriftung(a)}`
		)
		.join(', ');
	return `Andere Auflage in der ${h.klasse}: ${andere} — dieses Exemplar ist ${auflagenBeschriftung(h)}.`;
}
