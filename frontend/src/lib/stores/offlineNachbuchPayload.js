import { leserFuerVersand } from './offlineWarteschlange.js';

// Baut die Portion für die Nachbuch-Tür (POST /api/action/nachbuchen). Sie kann den
// Scan-Zeitpunkt buchen, einen Schlüssel genau einmal buchen, einen offline gescannten
// Ausweis auflösen und jede Abweichung als Meldung festhalten.
//
// Die Uhr: `gescannt_am` und `gesendet_am` müssen von derselben Uhr kommen, sonst rechnet
// der Server den Versatz falsch. Das ist hier kein Randfall: Die Uhr eines Theken-Rechners
// wird oft in dem Moment korrigiert, in dem das Netz zurückkommt, also zwischen Scan und
// Versand.
//
// Deshalb wird der Scan-Zeitpunkt neu bestimmt, wenn der Eintrag aus diesem Seitenaufruf
// stammt: `performance.now()` läuft gleichmäßig weiter und springt nicht. Aus dem Abstand
// seit dem Scan und der Wanduhr von jetzt ergibt sich ein Scan-Zeitpunkt, der zu
// `gesendet_am` passt, auch wenn die Wanduhr dazwischen gesprungen ist. Stammt der Eintrag
// aus einem früheren Seitenaufruf (Neuladen, eingespielte Sicherung), gilt der gespeicherte
// Wert; dort gibt es nichts Besseres.
/**
 * @param {import('./offlineWarteschlange.js').OfflineEintrag[]} batchItems
 * @returns {{ gesendet_am: string, eintraege: any[] }}
 */
export function baueNachbuchPayload(batchItems) {
	const jetzt = Date.now();
	const mono = typeof performance !== 'undefined' ? Math.round(performance.now()) : null;
	const ursprung =
		typeof performance !== 'undefined' ? Math.round(performance.timeOrigin ?? 0) : null;

	return {
		gesendet_am: new Date(jetzt).toISOString(),
		eintraege: batchItems.map((posten) => {
			let gescannt = posten.gescannt_am;
			if (mono !== null && posten.ursprung === ursprung && typeof posten.mono === 'number') {
				gescannt = jetzt - (mono - posten.mono);
			}
			/** @type {any} */
			const e = {
				schluessel: posten.id,
				absicht: posten.art,
				barcode: posten.barcode,
				gescannt_am: new Date(gescannt).toISOString()
			};
			// Die Person nur bei einer Ausleihe (leserFuerVersand); ein ohne Netz gescannter
			// Ausweis steht in `ausweis_barcode`, die Dauer des Besuchs in `besuch_seit_ms`.
			if (leserFuerVersand(posten)) e.leser_id = leserFuerVersand(posten);
			if (posten.ausweis_barcode) e.ausweis_barcode = posten.ausweis_barcode;
			if (posten.art === 'ausleihe' && typeof posten.besuch_seit_ms === 'number') {
				e.besuch_seit_ms = posten.besuch_seit_ms;
			}
			return e;
		})
	};
}
