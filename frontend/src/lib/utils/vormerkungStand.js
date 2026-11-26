import { formatDatum } from './format.js';

/**
 * Der Stand einer Vormerkung, gleich in Leser- und Buchakte: Eine abholbereite nennt das Ende der
 * Abholfrist, jede andere ('wartend' und Unbekanntes) die Wartezeit. vormerkung-status-konsistenz
 * hält die Statuswerte der Datenbank und diese Zeilen deckungsgleich.
 * @param {{ status?: string, bereitgestellt_bis?: string, erstellt_am: string }} v
 */
export function vormerkungStand(v) {
	if (v.status !== 'abholbereit') return `Wartet seit: ${formatDatum(v.erstellt_am)}`;
	return v.bereitgestellt_bis
		? `Abholbereit bis: ${formatDatum(v.bereitgestellt_bis)}`
		: 'Abholbereit';
}
