import { idStore } from '../../designer/idDesignerStore.svelte.js';
import { felderProBogen } from '../../etikettformate.js';
import { oeffneEtikettenbogen } from '../../schuelerEtiketten.js';

/**
 * Der Stapeldruck der Schülerdatei: Ausweiskarten oder Klebeetiketten.
 *
 * WOMIT gedruckt wird, steht im zentral gespeicherten Ausweis-Design
 * (`idStore.printMode`) — der Designer legt das fest, diese Ansicht nur, WER gedruckt
 * wird. Die beiden Wege sind grundverschieden: Karten entstehen über die Druck-CSS des
 * Browsers (StudentBatchPrint rendert sie versteckt ins Dokument), der Etikettenbogen
 * kommt als fertiges PDF vom Server. Millimetergenaue Klebebögen über Browser-Druck-CSS
 * zu treffen, hat noch nie zuverlässig funktioniert.
 *
 * Eigene Datei, weil StudentDirectory.svelte an der Größen-Ratsche steht und der Druck
 * das eine Stück darin ist, das nichts mit dem Führen der Liste zu tun hat.
 */
export function erzeugeAusweisdruck() {
	const etikettModus = $derived(idStore.printMode === 'etikett');

	// Angebrochener Klebebogen: Auf welchem Feld soll der Druck anfangen? Bewusst NICHT
	// im zentral gespeicherten Design, anders als das Format: Wie viele Etiketten schon
	// abgezogen sind, ist eine Eigenschaft des Bogens in der Hand — nicht der Schule.
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
			document.body.setAttribute('data-print-mode', 'card');
			document.body.setAttribute('data-print-side', 'front');
			window.print();
			document.head.removeChild(style);
			document.body.removeAttribute('data-print-mode');
			document.body.removeAttribute('data-print-side');
		}
	};
}
