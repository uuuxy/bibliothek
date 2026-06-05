import { apiFetch } from '../../lib/apiFetch.js';
/**
 * startseiten_api.js
 * 
 * Enthält alle API-Aufrufe und Hilfsfunktionen für die Gast-Startseite.
 * Hierzu gehören: Bücher laden, Klassen laden,
 * sowie Filterung und Gruppierung der Bücher nach Klassen.
 */

/**
 * Lädt alle Bücher aus der API.
 * @returns {Promise<any[]>} Liste der Bücher
 */
export async function buecherLaden() {
    const antwort = await apiFetch("/api/books", {
        credentials: "include",
    });
    if (!antwort.ok) {
        if (antwort.status === 401) {
            throw new Error("UNAUTHORIZED");
        }
        throw new Error("Fehler beim Laden der Bücher");
    }
    return (await antwort.json()).data ?? [];
}

/**
 * Lädt die echten Schulklassen (mit zugewiesenen Büchern) aus der API.
 * @returns {Promise<any[]>} Liste der Klassengruppen
 */
export async function echteKlassenLaden() {
    const antwort = await apiFetch("/api/class-books", {
        credentials: "include",
    });
    if (!antwort.ok) return [];
    const daten = (await antwort.json()).data ?? [];
    return daten.map((/** @type {any} */ klasse) => ({
        name: klasse.className,
        books: klasse.books,
    }));
}

/**
 * Gruppiert ein Array von Büchern in Klassengruppen (z.B. "Klasse 5 G").
 * @param {any[]} buecherArray - Alle verfügbaren Bücher
 * @returns {any[]} Sortierte Liste von Klasseobjekten
 */
export function buecherNachKlassenGruppieren(buecherArray) {
    const klassenMap = new Map();
    for (const buch of buecherArray) {
        if (!buch.gradeLevel) continue;
        const klassenName = `Klasse ${buch.gradeLevel}${buch.track ? " " + buch.track : ""}`;
        if (!klassenMap.has(klassenName)) {
            klassenMap.set(klassenName, { name: klassenName, books: [] });
        }
        klassenMap.get(klassenName).books.push(buch);
    }
    return Array.from(klassenMap.values()).sort((a, b) =>
        a.name.localeCompare(b.name),
    );
}

/**
 * Bestimmt die CSS-Klasse für die Bestandsanzeige (Farbampel).
 * @param {number} bestand - Aktueller Buchbestand
 * @returns {string} Tailwind-CSS-Klassen für die Bestandsanzeige
 */
export function bestandsFarbe(bestand) {
    if (bestand === 0)
        return "bg-red-500 shadow-[0_0_8px_rgba(239,68,68,0.5)]";
    if (bestand < 5)
        return "bg-amber-500 shadow-[0_0_8px_rgba(245,158,11,0.5)]";
    return "bg-emerald-500 shadow-[0_0_8px_rgba(16,185,129,0.5)]";
}
