// Was die Anwendung je Vorlagen-Typ (mail_vorlagen.typ) weiß: den Namen in der Liste, wofür der
// Text verwendet wird und welche Platzhalter der Renderer ersetzt. Die Platzhalter hält
// api/mail_vorlagen_platzhalter_test.go deckungsgleich mit den Go-Renderern (pdf/mahnbrief.go,
// internal/service/bestellmail_text.go); ein Platzhalter, den nur diese Liste nennt, stünde
// wörtlich im Versand.
/** @type {Record<string, { name: string, verwendung: string, platzhalter: string[] }>} */
export const vorlagenInfo = {
	MAHNUNG_ELTERN: {
		name: 'Mahnbrief an die Eltern',
		verwendung:
			'Gedruckter Eltern-Mahnbrief (Fensterkuvert) — es geht keine Mail an Eltern. Die Mahn-Mails an Klassenleitungen haben eigene, feste Texte.',
		platzhalter: ['{{.Vorname}}', '{{.Nachname}}', '{{.BuchListe}}', '{{.Frist}}']
	},
	BESTELLUNG_HAENDLER: {
		name: 'Bestellung an den Händler',
		verwendung:
			'Bestellmail an den Buchhändler. Fehlt {{.BestaetigungsLink}} im Text, hängt das System den Bestätigungs-Link automatisch als eigenen Absatz an. {{.LinkGueltigBis}} ist das Ablaufdatum des Links (Einstellung „Bestellwesen"). {{.Mittel}} ist der Topf der Bestellung („Lernmittelfreiheit" oder „Schülerbücherei") — fehlt er, ergänzt das System Betreff und Text automatisch um den Vermerk.',
		platzhalter: [
			'{{.Datum}}',
			'{{.Kundennummer}}',
			'{{.AnzahlTitel}}',
			'{{.AnzahlExemplare}}',
			'{{.BestaetigungsLink}}',
			'{{.LinkGueltigBis}}',
			'{{.Mittel}}'
		]
	}
};

/**
 * Name einer Vorlage für die Liste. Ein Typ, den diese Tabelle nicht kennt, steht mit seinem
 * Schlüssel da statt als leere Zeile.
 * @param {string} typ
 */
export const vorlagenName = (typ) => vorlagenInfo[typ]?.name ?? typ.replaceAll('_', ' ');
