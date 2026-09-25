/**
 * Wie eine Auflage in Listen heißt (docs/OFFEN.md 4.18). Eine Stelle für den Abschnitt der
 * Titelmaske, den Zuordnen-Dialog und den Bestellbedarf: Alle zeigen Titel derselben Reihe,
 * die sich oft nur in Auflage und Jahr unterscheiden — „Lambacher Schweizer 7" zweimal hilft
 * niemandem.
 *
 * Das Feld „Auflage" trägt das Jahr manchmal schon („4. Aufl. 2023"); dann steht es nicht
 * ein zweites Mal daneben.
 * @param {{ auflage?: string, erscheinungsjahr?: number }} a
 */
export function auflagenBeschriftung(a) {
	const auflage = (a.auflage ?? '').trim();
	const jahr = a.erscheinungsjahr || 0;
	if (auflage && jahr && !auflage.includes(String(jahr))) return `${auflage} · ${jahr}`;
	if (auflage) return auflage;
	if (jahr) return `Ausgabe ${jahr}`;
	return 'Ohne Angabe zur Auflage';
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
