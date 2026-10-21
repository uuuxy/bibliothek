// Die Reiter des Druck-Centers mit den Rechten der Routen dahinter; ein Reiter braucht jedes
// seiner Rechte. Der Menüpunkt ist ein Sammelpunkt über ihnen (menue.js): Hinge er an einem
// einzigen Recht, erreichte wer nur Bücher pflegt seine Etiketten nicht.

/** @type {{ id: string, label: string, rechte: string[] }[]} */
export const DRUCK_CENTER_REITER = [
	// Gedruckt wird über POST /api/print/labels (view_books), danach vermerkt über
	// POST /api/exemplare/etiketten-gedruckt (edit_books). Ohne den Vermerk bliebe jedes
	// gedruckte Exemplar unter „Fehlende Etiketten" stehen.
	{ id: 'labels', label: 'Buch-Etiketten', rechte: ['view_books', 'edit_books'] },
	// GET /api/exemplare/etiketten-offen, POST /api/exemplare/etiketten-gedruckt
	{ id: 'nachdruck', label: 'Fehlende Etiketten', rechte: ['edit_books'] },
	// GET /api/ausweis-layout
	{ id: 'ids', label: 'Schülerausweise', rechte: ['view_students'] },
	// öffnet die Leserdatei mit markierter Klasse
	{ id: 'klassen', label: 'Klassenweise drucken', rechte: ['view_students'] }
];

/**
 * Die Rechte, mit denen der Menüpunkt „Druck-Center" sichtbar ist, eines genügt: Jedes öffnet
 * allein einen Reiter, und jeder Reiter braucht eines davon. So ist der Punkt genau dann
 * sichtbar, wenn ein Reiter aufgeht (menue.test.js).
 */
export const DRUCK_RECHTE = ['edit_books', 'view_students'];

/**
 * Die Rechte des Reiters mit dieser Kennung. Eine Übergabe an den Etikettendruck braucht die
 * von „labels", nicht nur den Menüpunkt: Ohne den Reiter landete sie auf einem anderen.
 * @param {string} id
 * @returns {string[]}
 */
export function rechteDesReiters(id) {
	const reiter = DRUCK_CENTER_REITER.find((r) => r.id === id);
	if (!reiter) throw new Error(`Den Reiter „${id}“ gibt es im Druck-Center nicht`);
	return reiter.rechte;
}
