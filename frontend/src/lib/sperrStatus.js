// DIE eine Antwort auf „darf dieser Leser ausleihen?" für die Anzeige —
// dieselbe Regel, die der Server an der Theke prüft (pruefeAusleihSperren in
// internal/service/ausleih_sperren.go): die zwei Sperren am Leser, und ein Kollege
// wird nie gesperrt.
//
// Bis zum 31.08.2026 las jede Stelle ihre eigene Spalte: Die Schülerliste nur
// ist_gesperrt (ein manuell Gesperrter stand als „Alles ok" da), das Profil
// rechnete nach dem Umschalten eine ERFUNDENE Formel (manuell || offene Schäden)
// und überschrieb den Serverwert — der Befund „entsperrt, wieder aktiv, nach Reload
// wieder gesperrt". Drei Definitionen, nur zufällig einig.
//
// ist_gesperrt ist die Sperre, die das Programm setzt (Ehemalige, Papierkorb,
// Anonymisierung); is_manually_blocked die von Hand. Seit dem 24.09.2026 hebt der
// Knopf in der Akte beide auf (api/student_lock.go) und richtet sich deshalb nach
// diesem Prädikat. Einen Kollegen sperrt niemand mehr (entschieden am 16.09.2026,
// bestätigt am 24.09.2026) — eine alte Sperre an seinem Konto zeigt die Anzeige nicht
// an, weil die Theke sie nicht prüft.
import { istKollegium } from './leserArt.js';

/** @param {any} s Leser-Objekt mit art / ist_gesperrt / is_manually_blocked */
export function ausleiheGesperrt(s) {
	if (istKollegium(s)) return false;
	return !!(s?.ist_gesperrt || s?.is_manually_blocked);
}
