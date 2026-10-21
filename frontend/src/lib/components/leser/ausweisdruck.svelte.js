import {
	designFuerDruck,
	ladeAusweisDesign,
	meldeDesignNichtGeladen
} from '../../designer/ausweisDesignLaden.js';
import { idStore } from '../../designer/idDesignerStore.svelte.js';
import { felderProBogen } from '../../etikettformate.js';
import { oeffneEtikettenbogen } from '../../schuelerEtiketten.js';
import { snackbars } from '../../stores/snackbars.svelte.js';

/**
 * Die Markierten, für die es einen Ausweis gibt. Ohne Nummer trüge die Karte ein leeres
 * Strichcodefeld und das Etikett nur Name und Klasse; die Akte sperrt dann ihren Druckknopf
 * (LeserakteAktionen.svelte), der Stapeldruck lässt solche Leser aus und nennt sie.
 * @param {any[]} markierte
 */
export function mitAusweisnummer(markierte) {
	return markierte.filter((leser) => !!leser.barcode_id);
}

/** @param {any[]} ohneNummer */
function meldeOhneNummer(ohneNummer) {
	const namen = ohneNummer.slice(0, 3).map((l) => `${l.vorname} ${l.nachname}`);
	if (ohneNummer.length > 3) namen.push(`${ohneNummer.length - 3} weitere`);
	snackbars.addToast(
		`Ohne Ausweisnummer kein Ausweis für ${namen.join(', ')}. Die Nummer wird in der Akte unter „Stammdaten & Adresse“ eingetragen.`,
		'warning'
	);
}

/**
 * Der Stapeldruck der Schülerdatei: Ausweiskarten oder Klebeetiketten.
 *
 * Womit gedruckt wird, steht im zentral gespeicherten Ausweis-Design
 * (`idStore.printMode`): Der Designer legt das fest, diese Ansicht nur, wer gedruckt wird.
 * Karten entstehen über die Druck-CSS des Browsers (AusweisStapeldruck zeichnet sie
 * versteckt ins Dokument), der Etikettenbogen kommt als fertiges PDF vom Server, weil die
 * Druck-CSS des Browsers Klebebögen nicht auf den Millimeter trifft.
 */
export function erzeugeAusweisdruck() {
	// Am printMode entscheidet sich, was aus dem Drucker kommt und welche Bedienung die
	// Seite zeigt. Wer ihn liest, sorgt selbst dafür, dass er geladen ist; die Druckfläche
	// (AusweisStapeldruck) hängt nur im Baum, solange Karten markiert sind.
	ladeAusweisDesign().then((geladen) => {
		if (!geladen) meldeDesignNichtGeladen();
	});

	const etikettModus = $derived(idStore.printMode === 'etikett');

	// Auf welchem Feld eines angebrochenen Klebebogens der Druck anfängt. Nicht im zentral
	// gespeicherten Design: Wie viele Etiketten schon abgezogen sind, gilt für den Bogen in
	// der Hand, nicht für die Schule.
	let startPosition = $state(1);
	const maxPosition = $derived(felderProBogen(idStore.etikettFormat));

	return /** @satisfies {object} */ ({
		get etikettModus() {
			return etikettModus;
		},
		get maxPosition() {
			return maxPosition;
		},
		get startPosition() {
			return startPosition;
		},
		set startPosition(wert) {
			startPosition = wert;
		},

		/** @param {any[]} markierte */
		async drucke(markierte) {
			const druckbar = mitAusweisnummer(markierte);
			if (druckbar.length < markierte.length) {
				meldeOhneNummer(markierte.filter((leser) => !leser.barcode_id));
			}
			if (druckbar.length === 0) return;
			if (!(await designFuerDruck())) return;
			if (etikettModus) {
				await oeffneEtikettenbogen({
					formatId: idStore.etikettFormat,
					startPosition,
					schuelerIds: druckbar.map((s) => s.id)
				});
				return;
			}
			const style = document.createElement('style');
			style.textContent = '@media print { @page { size: 85.6mm 53.98mm; margin: 0; } }';
			document.head.appendChild(style);
			document.body.dataset.printMode = 'card';
			document.body.dataset.printSide = 'front';
			window.print();
			style.remove();
			delete document.body.dataset.printMode;
			delete document.body.dataset.printSide;
		}
	});
}
