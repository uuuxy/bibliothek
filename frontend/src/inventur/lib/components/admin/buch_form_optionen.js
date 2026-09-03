export const klassenStufen = [0, 5, 6, 7, 8, 9, 10, 11, 12, 13];

/**
 * leeresBuchFormular: die EINE Vorlage für ein neues Buch. Sie stand bis zum 03.09.2026
 * zweimal wörtlich in routes/admin/+page.svelte (Anfangszustand und „Neues Buch"); beim
 * Nachtragen des Schulzweigs fiel auf, dass ein neues Feld an beiden Stellen gepflegt
 * werden muss — vergisst man eine, schickt genau einer der beiden Wege das Feld nie mit.
 * @returns {{ id: null, isbn: string, title: string, author: string, subject: string, gradeLevel: number, istLernmittel: boolean, track: string, stock: number, coverUrl: string, lastCounted: string, medientyp: string }}
 */
export function leeresBuchFormular() {
	return {
		id: null,
		isbn: '',
		title: '',
		author: '',
		subject: '',
		gradeLevel: 5,
		istLernmittel: false,
		track: '',
		stock: 0,
		coverUrl: '',
		lastCounted: '',
		medientyp: 'Buch'
	};
}
