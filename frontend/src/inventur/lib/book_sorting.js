export const subjectOrder = [
    "mathe",
    "deutsch",
    "englisch",
    "französisch",
    "spanisch",
    "latein",
    "erdkunde",
    "geographie",
    "geschichte",
    "politik",
    "arbeitslehre",
    "biologie",
    "chemie",
    "physik"
];

/**
 * Sorts two book objects first by subject (using a predefined order), then by title.
 * @param {Object} a - First book object
 * @param {Object} b - Second book object
 * @returns {number} Sorting result
 */
/**
 * @param {{subject?: string, title: string, gradeLevel?: string|number, track?: string}} a
 * @param {{subject?: string, title: string, gradeLevel?: string|number, track?: string}} b
 */
export function sortBooksBySubjectAndTitle(a, b) {
    let subjA = (a.subject || "").toLowerCase().trim();
    let subjB = (b.subject || "").toLowerCase().trim();

    // Handle variations
    if (subjA === "mathematik") subjA = "mathe";
    if (subjB === "mathematik") subjB = "mathe";

    let indexA = subjectOrder.indexOf(subjA);
    let indexB = subjectOrder.indexOf(subjB);
    if (indexA === -1) indexA = 999;
    if (indexB === -1) indexB = 999;

    if (indexA !== indexB) {
        return indexA - indexB;
    }
    return a.title.localeCompare(b.title);
}
