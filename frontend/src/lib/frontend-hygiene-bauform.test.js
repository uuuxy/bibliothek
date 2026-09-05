import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { srcRoot, sammleQuelldateien, relPfad, vergleicheMitBestand } from './hygiene-quellen.js';

// Material 3 kennt keine Fläche, die Rahmen UND Erhebung zugleich trägt: 6 seiner 84
// Bauteile haben einen Rahmen, 36 eine Erhebung, die Schnittmenge ist leer. Die Regel
// steht ausführlich im Kopf von `e2e/m3-bauform.spec.js` — dieser Test ersetzt sie
// nicht, er deckt ihre BLINDE HÄLFTE ab.
//
// Anlass (04.09.2026): Die Coverkachel im Statistik-Dashboard trug `rounded shadow-xs
// border border-slate-100`. Das e2e-Gate läuft alle Routen ab und misst, was gerendert
// IST — die Liste „Top-Bücher" hat im e2e-Bestand aber keine Zeile mit Cover, also
// stand dort nie ein Bild, also sah das Gate den Verstoß nie. Gefunden hat ihn ein
// fremder Agent beim Lesen des Quelltexts.
//
// ARBEITSTEILUNG, nicht Ersatz:
//   * Das e2e-Gate MISST im Browser. Nur dort ist zu unterscheiden, ob ein Schatten
//     überhaupt malt (Tailwind rendert ungesetzte Schattenvariablen als Deckkraft 0),
//     ob ein `ring-*` ein Fokusring statt einer Erhebung ist, und ob eine Erhebung nur
//     im :hover gilt (das erlaubt M3 ausdrücklich). Diese drei Fallen kann ein Blick in
//     den Quelltext nicht auflösen.
//   * Dieser Test LIEST. Er sieht dafür auch, was gerade keine Daten hat, was hinter
//     einem `{#if}` liegt und was in einem geschlossenen Dialog steckt.
// Ein neuer Verstoß muss durch beide Maschen fallen, um unbemerkt zu bleiben.
//
// Erkannt wird eine Klassenliste, in der eine breitengebende Rahmen-Utility und eine
// Schatten-Utility OHNE Zustandspräfix nebeneinander stehen. `hover:shadow-xl`,
// `focus-within:ring-2`, `shadow-none` und reine Farb-Utilities (`border-slate-100`
// ohne `border`) zählen nicht — das schließt das führende Zeichen aus.
const RAHMEN = /(^|[\s'"`{(])border(-[trblxy])?(-[1-8])?(?=[\s'"`});]|$)/;
const SCHATTEN = /(^|[\s'"`{(])shadow(-(xs|sm|md|lg|xl|2xl))?(?=[\s'"`});]|$)/;

// Eingefroren am 05.09.2026, am selben Tag um neun Einträge gekürzt: Die neun
// selbstgebauten Dialoge haben ihren Rahmen abgegeben und behalten die Erhebung — die
// Rolle entscheidet, welcher der beiden Teile weicht, und ein Dialog ist in M3 eine
// erhobene Fläche (level3, kein outline-Token). Was bleibt, ist bewusst so:
//
//   * `OmniboxBlockAlert`/`OmniboxVormerkungAlert` tragen `border-4 border-rose-500`
//     als Alarmsignal an der Theke — dort IST der Rahmen die Aussage. Sie behalten
//     BEIDES: Ein Alarm, der einen Schüler an der Ausleihe stoppt, wird nicht leiser
//     gemacht, um eine Gestaltungsregel zu erfüllen. Bewusste Ausnahme, keine Schuld.
//   * `LabelPreview` bildet Papier nach; der Rahmen ist die Etikettenkante.
//   * Die Cover-Bilder (`BookTableZeile`, `BorrowedBooksList`, `OrderSearch`,
//     `WareneingangTable`, `CoverPeek`) sind Bilder, keine M3-Bauteile.
//
// Wer einen Eintrag abräumt, trägt ihn hier aus. Wer einen hinzufügen will, hat einen
// Verstoß gebaut — die Regel ist nicht ausgelegt, sondern in Googles Token-Spezifikation
// nachgezählt.
const BESTAND = [
	'src/inventur/lib/components/admin/BookTableZeile.svelte',
	'src/inventur/lib/components/admin/ClassAssignmentBookGrid.svelte',
	'src/inventur/lib/components/admin/ClassAssignmentSelector.svelte',
	'src/lib/BestellWorkspace.svelte',
	'src/lib/BookVormerkungenTab.svelte',
	'src/lib/BorrowedBooksList.svelte',
	'src/lib/CameraScanner.svelte',
	'src/lib/LitteraImportWidget.svelte',
	'src/lib/OmniboxTeacherCard.svelte',
	'src/lib/PermissionManager.svelte',
	'src/lib/StudentProfileActions.svelte',
	'src/lib/UserManagement.svelte',
	'src/lib/components/BookExemplarCard.svelte',
	'src/lib/components/OfflineIndicator.svelte',
	'src/lib/components/OmniboxBlockAlert.svelte',
	'src/lib/components/OmniboxVormerkungAlert.svelte',
	'src/lib/components/auth/Login.svelte',
	'src/lib/components/bestellungen/BestellHistorieTabelle.svelte',
	'src/lib/components/bestellungen/OrderSearch.svelte',
	'src/lib/components/bestellungen/WareneingangTable.svelte',
	'src/lib/components/labels/LabelPreview.svelte',
	'src/lib/components/students/AuswahlAktionsleiste.svelte',
	'src/lib/components/ui/CoverPeek.svelte',
	'src/lib/designer/CanvasArea.svelte',
	'src/lib/designer/PropertiesPanel.svelte'
];

/** Alle class-Attributwerte einer Svelte-Datei — Zeichenkette wie Ausdruck. */
function klassenlisten(quelle) {
	// OHNE KOMMENTARE. Ein Detektor, der die Begründung für die Sache hält, meldet ewig
	// „alles gut" (Bugklasse „Lügende Ratsche", docs/sweeps.md).
	const code = quelle.replace(/<!--[\s\S]*?-->/g, '').replace(/\/\*[\s\S]*?\*\//g, '');
	return [...code.matchAll(/\bclass=("([^"]*)"|'([^']*)'|\{([^}]*)\})/g)].map(
		(m) => m[2] ?? m[3] ?? m[4] ?? ''
	);
}

describe('M3-Bauform im Quelltext', () => {
	it('keine neue Klassenliste trägt Rahmen und Erhebung zugleich', () => {
		/** @type {Record<string, string>} */
		const belege = {};
		const betroffen = [];

		for (const datei of sammleQuelldateien(srcRoot)) {
			if (!datei.endsWith('.svelte')) continue;
			const treffer = klassenlisten(readFileSync(datei, 'utf8')).find(
				(wert) => RAHMEN.test(wert) && SCHATTEN.test(wert)
			);
			if (treffer === undefined) continue;
			const pfad = relPfad(datei);
			betroffen.push(pfad);
			belege[pfad] = treffer.replace(/\s+/g, ' ').slice(0, 120);
		}
		betroffen.sort();

		const { neu, inzwischenSauber } = vergleicheMitBestand(betroffen, BESTAND);

		expect(
			neu,
			'Neue Fläche mit Rahmen UND Schatten. Material 3 kennt diese Bauform bei keinem ' +
				'seiner 84 Bauteile: Entweder der Rahmen geht (dann ist es eine erhobene Fläche — ' +
				'Dialog, Menü, Snackbar) oder der Schatten (dann eine umrandete — outlined-*, ' +
				'data-table).\n' +
				neu.map((f) => `  ${f}\n      ${belege[f]}`).join('\n')
		).toEqual([]);

		expect(
			inzwischenSauber,
			'Der Bestand ist überholt — bitte aus BESTAND austragen. Eine Liste, die ' +
				'Erledigtes weiterführt, verliert ihre Aussage.'
		).toEqual([]);
	});
});
