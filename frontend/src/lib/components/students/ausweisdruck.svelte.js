import {
	designFuerDruck,
	ladeAusweisDesign,
	meldeDesignNichtGeladen
} from '../../designer/ausweisDesignLaden.js';
import { idStore } from '../../designer/idDesignerStore.svelte.js';
import { felderProBogen } from '../../etikettformate.js';
import { oeffneEtikettenbogen } from '../../schuelerEtiketten.js';

/**
 * Der Stapeldruck der Schülerdatei: Ausweiskarten oder Klebeetiketten.
 *
 * Womit gedruckt wird, steht im zentral gespeicherten Ausweis-Design
 * (`idStore.printMode`): Der Designer legt das fest, diese Ansicht nur, wer gedruckt wird.
 * Karten entstehen über die Druck-CSS des Browsers (StudentBatchPrint zeichnet sie
 * versteckt ins Dokument), der Etikettenbogen kommt als fertiges PDF vom Server, weil die
 * Druck-CSS des Browsers Klebebögen nicht auf den Millimeter trifft.
 */
export function erzeugeAusweisdruck() {
	// Am printMode entscheidet sich, was aus dem Drucker kommt und welche Bedienung die
	// Seite zeigt. Wer ihn liest, sorgt selbst dafür, dass er geladen ist; die Druckfläche
	// (StudentBatchPrint) hängt nur im Baum, solange Karten markiert sind.
	ladeAusweisDesign().then((geladen) => {
		if (!geladen) meldeDesignNichtGeladen();
	});

	const etikettModus = $derived(idStore.printMode === 'etikett');

	// Auf welchem Feld eines angebrochenen Klebebogens der Druck anfängt. Nicht im zentral
	// gespeicherten Design: Wie viele Etiketten schon abgezogen sind, gilt für den Bogen in
	// der Hand, nicht für die Schule.
	let startPosition = $state(1);
	const maxPosition = $derived(felderProBogen(idStore.etikettFormat));

	return {
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
			if (!(await designFuerDruck())) return;
			if (etikettModus) {
				await oeffneEtikettenbogen({
					formatId: idStore.etikettFormat,
					startPosition,
					schuelerIds: markierte.map((s) => s.id)
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
	};
}
