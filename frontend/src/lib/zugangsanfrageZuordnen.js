import { apiFetch, extractApiError } from './apiFetch.js';

/**
 * Ordnet eine Zugangsanfrage dem Eintrag zu, der schon in der Leserdatei steht: Das Konto
 * zieht dorthin um, die Leserzeile des Antrags geht darin auf.
 *
 * Steht als eigene Funktion neben der Komponente, weil die Komponente sonst über die
 * 200-Zeilen-Regel liefe (docs/ARCHITECTURE.md) — und weil die RICHTUNG damit an einer
 * Stelle steht, die man lesen kann, ohne durch Markup zu blättern.
 *
 * Die Richtung ist der Kern: `zielID` ist der vorhandene Eintrag. Er BLEIBT, mit Ausweis,
 * Büchern und richtig geschriebenem Namen. `quelleID` ist die Leserzeile, die der Wächter
 * trg_benutzer_hat_leserzeile an das Anfrage-Konto gehängt hat; sie geht auf. Verkehrt
 * herum verschwände der vorhandene Eintrag, und der aus der E-Mail-Adresse geratene Name
 * bliebe übrig — still, und nur über den Rückweg-Eintrag im Protokoll zu reparieren.
 *
 * @param {string} zielID Leserzeile, die bleibt (der vorhandene Eintrag)
 * @param {string} quelleID Leserzeile, die aufgeht (die des Antrags)
 * @returns {Promise<string>} leer bei Erfolg, sonst die Meldung für die Oberfläche
 */
export async function ordneZu(zielID, quelleID) {
	try {
		const res = await apiFetch(`/api/schueler/${zielID}/zusammenfuehren`, {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ quelle_id: quelleID })
		});
		if (!res.ok) return (await extractApiError(res)) || 'Zuordnen fehlgeschlagen';
		return '';
	} catch (e) {
		return e instanceof Error ? e.message : 'Zuordnen fehlgeschlagen';
	}
}
