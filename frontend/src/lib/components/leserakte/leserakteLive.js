import { abonniere } from '../../liveEvents.js';

/**
 * Lädt die offene Leserakte neu, sobald ein anderer Platz für denselben Leser ausleiht oder
 * zurücknimmt. Sonst stünde ein dort zurückgegebenes Buch hier weiter als überfällig. Theke und
 * Leserdatei öffnen dieselbe Akte und hören deshalb auf dieselbe Weise zu; solange die Maske
 * „Stammdaten bearbeiten" offen ist, wartet das Neuladen (useLeserakte, ladeAkte).
 * @param {() => string | undefined} offeneId Kennung des Lesers, dessen Akte offen ist
 * @param {() => void} ladeNeu
 * @returns {() => void} Abmeldung
 */
export function ladeAkteBeiBuchung(offeneId, ladeNeu) {
	return abonniere('action', (e) => {
		try {
			const id = offeneId();
			if (id && JSON.parse(e.data).student_id === id) ladeNeu();
		} catch (err) {
			console.error('Live-Meldung zur Leserakte nicht lesbar:', err);
		}
	});
}
