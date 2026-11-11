// Das voraussichtliche Abgangsjahr aus der Klasse, für den Knopf „Neu berechnen“ in der Akte.
// Es ist der Zwilling von repository.AbgaengerJahrAm, der bei Anlage, Klassenwechsel und
// LUSD-Import rechnet; beide Seiten prüft abgangsjahr.faelle.json. Rechnete der Browser anders,
// schriebe „Neu berechnen“ ein anderes Jahr in die Akte als der nächste Klassenwechsel.

/**
 * H endet mit 9, R mit 10, alles andere mit 13; „E…“ ist die Einführungsphase, Jahrgang 11.
 * Ohne Jahrgang von 1 bis 13 gilt das Jahr in fünf Jahren. Ab dem 1. August zählt das
 * folgende Schuljahr.
 * @param {string} klasse
 * @param {Date} jetzt
 * @returns {number}
 */
export function abgangsjahrAm(klasse, jetzt) {
	const kl = (klasse || '').toLowerCase().trim();
	let jahrgang = 11;
	let abschluss = 13;
	if (!kl.startsWith('e')) {
		const ziffern = kl.match(/^\d+/)?.[0] ?? '';
		jahrgang = Number(ziffern);
		if (ziffern === '' || jahrgang < 1 || jahrgang > 13) return jetzt.getFullYear() + 5;
		const zweig = kl.slice(ziffern.length).trim();
		abschluss = zweig.startsWith('h') ? 9 : zweig.startsWith('r') ? 10 : 13;
	}
	const basis = jetzt.getMonth() >= 7 ? jetzt.getFullYear() + 1 : jetzt.getFullYear();
	return basis + Math.max(0, abschluss - jahrgang);
}
