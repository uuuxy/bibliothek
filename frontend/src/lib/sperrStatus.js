// DIE eine Antwort auf „darf dieser Schüler ausleihen?" für die Anzeige —
// dasselbe Paar, das der Server an der Theke prüft (ist_gesperrt OR
// is_manually_blocked, z. B. internal/service/loan_return.go).
//
// Bis zum 31.08.2026 las jede Stelle ihre eigene Spalte: Die Schülerliste nur
// ist_gesperrt (ein manuell Gesperrter stand als „Alles ok" da), das Profil
// rechnete nach dem Umschalten eine ERFUNDENE Formel (manuell || offene Schäden)
// und überschrieb den Serverwert — Peters „entsperrt, wieder aktiv, nach Reload
// wieder gesperrt". Drei Definitionen, nur zufällig einig.
//
// ist_gesperrt bleibt die SYSTEM-Sperre (Papierkorb, Abgänger, Anonymisierung);
// is_manually_blocked das Handschloss. Der Sperren-Knopf fragt weiterhin nur das
// Handschloss (StudentKontoStatus erklärt, warum).

/** @param {any} s Schüler-Objekt mit ist_gesperrt / is_manually_blocked */
export function ausleiheGesperrt(s) {
	return !!(s?.ist_gesperrt || s?.is_manually_blocked);
}
