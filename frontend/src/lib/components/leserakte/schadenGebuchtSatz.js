// Ganze Sätze statt zusammengesetzter Teile: Die Satzzeichen prüft frontend-hygiene-satzzeichen.
const SAETZE = {
	geraetVerloren:
		'Verlust oder Schaden gebucht, das Gerät steht in der Geräteverwaltung als verloren. Die Forderung steht unter „Gebühren & Schäden".',
	geraetDefekt:
		'Verlust oder Schaden gebucht, das Gerät steht in der Geräteverwaltung als defekt. Die Forderung steht unter „Gebühren & Schäden".',
	geraetVerlorenKollegium:
		'Verlust oder Schaden gebucht, das Gerät steht in der Geräteverwaltung als verloren. Eine Forderung entsteht bei einem Kollegen nicht.',
	geraetDefektKollegium:
		'Verlust oder Schaden gebucht, das Gerät steht in der Geräteverwaltung als defekt. Eine Forderung entsteht bei einem Kollegen nicht.',
	buchKollegium:
		'Verlust oder Schaden gebucht, das Exemplar ist ausgesondert. Eine Forderung entsteht bei einem Kollegen nicht.',
	buch: 'Verlust oder Schaden gebucht. Die Forderung steht unter „Gebühren & Schäden"; der Bescheid ist ein eigener Schritt.'
};

/**
 * Der Satz nach „Verlust oder Schaden melden“: was gebucht ist und wo die Forderung steht. Ein
 * Gerät bleibt in der Geräteverwaltung, beschädigt als defekt, verloren als verloren; einen
 * Bescheid gibt es dafür nicht (internal/repository/schaden_geraet.go).
 * @param {{ ohneForderung?: boolean, ist_geraet?: boolean }} ausleihe
 * @param {string} art 'nicht_zurueckgegeben' oder 'beschaedigt'
 */
export function schadenGebuchtSatz(ausleihe, art) {
	if (ausleihe.ist_geraet) {
		const defekt = art === 'beschaedigt';
		if (ausleihe.ohneForderung)
			return defekt ? SAETZE.geraetDefektKollegium : SAETZE.geraetVerlorenKollegium;
		return defekt ? SAETZE.geraetDefekt : SAETZE.geraetVerloren;
	}
	return ausleihe.ohneForderung ? SAETZE.buchKollegium : SAETZE.buch;
}
