// Signale zwischen den Fenstern eines Browsers für die Sperre nach Inaktivität.
//
// Die Anmeldung gehört dem ganzen Browser (ein Cookie), und der Server sperrt sie als Ganzes.
// Ohne diese Signale sperrte ein Fenster, das niemand benutzt, die Arbeit im Fenster daneben.
// Entschieden wird hier nichts: Ob gesperrt ist, sagt der Server. Ein Signal heißt nur „sieh nach".

const SPERRE = 'bibliothek.sperre';
const AKTIVITAET = 'bibliothek.aktivitaet';

/** @param {string} schluessel @param {string} wert */
function sende(schluessel, wert) {
	try {
		localStorage.setItem(schluessel, wert);
	} catch {
		/* Ohne Speicher (privates Fenster) bleibt jedes Fenster für sich; der Server sperrt trotzdem. */
	}
}

/** @param {number} jetzt */
export function meldeAktivitaet(jetzt) {
	sende(AKTIVITAET, String(jetzt));
}

/**
 * Sagt den anderen Fenstern, dass die Anmeldung am Server gesperrt oder aufgeschlossen wurde.
 * Der Zeitpunkt macht jedes Signal zu einem neuen Wert — nur dann feuert `storage`.
 * @param {boolean} gesperrt
 */
export function meldeSperre(gesperrt) {
	sende(SPERRE, `${gesperrt ? 'gesperrt' : 'offen'}:${Date.now()}`);
}

/**
 * Meldet, was in den anderen Fenstern dieses Browsers geschieht, und liefert die Abmeldung.
 * Das eigene Fenster hört sich dabei nicht selbst: `storage` feuert nur in den anderen.
 * @param {{ beiSperre: () => void, beiEntsperrt: () => void, beiAktivitaet: () => void }} zuhoerer
 * @returns {() => void}
 */
export function beobachteAndereFenster(zuhoerer) {
	/** @param {StorageEvent} e */
	const handler = (e) => {
		if (e.key === SPERRE) {
			if (e.newValue?.startsWith('gesperrt:')) zuhoerer.beiSperre();
			else zuhoerer.beiEntsperrt();
		} else if (e.key === AKTIVITAET) {
			zuhoerer.beiAktivitaet();
		}
	};
	window.addEventListener('storage', handler);
	return () => window.removeEventListener('storage', handler);
}
