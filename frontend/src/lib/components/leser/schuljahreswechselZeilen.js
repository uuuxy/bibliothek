/** @typedef {{ promoted_count: number, archived_count: number, dry_run: boolean, mapping_versetzt: number, mapping_entfernt: number, mapping_konflikte?: string[], klassen_ausserhalb?: string[] }} PromoteStudentsResponse */

/**
 * Die Zeilen der Zusammenfassung von Vorschau und Ergebnis. Klassen, die mit keinem Jahrgang
 * von 1 bis 13 beginnen, versetzt der Server nicht; die Zeile der Versetzten nennt sie.
 * @param {PromoteStudentsResponse | null} r
 */
export function zeilen(r) {
	return r
		? [
				{
					key: 'promoted',
					label: 'Versetzte Schüler',
					hint:
						'Klasse wird um eine Stufe hochgezählt' +
						(r.klassen_ausserhalb?.length
							? ' — nicht versetzt, weil kein Jahrgang von 1 bis 13: ' +
								r.klassen_ausserhalb.join(', ')
							: ''),
					value: r.promoted_count,
					valueClass: 'text-success'
				},
				{
					key: 'archived',
					label: 'Neue Abgänger',
					hint: 'Abschlussklassen werden archiviert',
					value: r.archived_count,
					valueClass: 'text-error'
				},
				{
					key: 'mapping',
					label: 'Klassenlehrer-Zuordnungen',
					hint:
						'rücken mit hoch, außer bei Abschluss, vor Klasse 7 und vor der Oberstufe' +
						(r.mapping_konflikte?.length
							? ' — Namenskonflikte: ' + r.mapping_konflikte.join(', ')
							: ''),
					value: r.mapping_versetzt + r.mapping_entfernt,
					valueClass: 'text-on-surface-variant'
				}
			]
		: [];
}
