// Die eine Antwort auf „darf dieser Leser ausleihen?" für die Anzeige, dieselbe Regel, die
// der Server an der Theke prüft (pruefeAusleihSperren in internal/service/ausleih_sperren.go):
// die zwei Sperren am Leser, und ein Kollege wird nie gesperrt. Liest jede Stelle ihre eigene
// Spalte, steht ein Gesperrter in der Liste als „Alles ok" da.
//
// ist_gesperrt ist die Sperre, die das Programm setzt (Ehemalige, Papierkorb,
// Anonymisierung); is_manually_blocked die von Hand. Der Knopf in der Akte hebt beide auf
// (api/leser/leser_sperre.go) und richtet sich deshalb nach diesem Prädikat. Eine alte Sperre
// am Konto eines Kollegen zeigt die Anzeige nicht an, weil die Theke sie nicht prüft.
import { istKollegium } from './leserArt.js';

/** @param {any} s Leser-Objekt mit art / ist_gesperrt / is_manually_blocked */
export function ausleiheGesperrt(s) {
	if (istKollegium(s)) return false;
	return !!(s?.ist_gesperrt || s?.is_manually_blocked);
}
