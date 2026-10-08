<!-- @component StudentBatchPrint — versteckte Druckfläche für einen Stapel Ausweise aus
     der Schülerdatei.

     Zeichnet über PrintPreview und damit über dieselbe CardFace-Quelle wie der Einzeldruck
     der Akte (StudentPrintCard) und die Vorschau im Ausweis-Designer: Ein eigener Renderer
     ergäbe eine zweite Karte, die niemand pflegt. Ob Karten oder Etiketten gedruckt werden,
     entscheidet das gespeicherte Design (idStore.printMode); die Schülerdatei legt fest,
     wer gedruckt wird. -->
<script>
	import { onMount } from 'svelte';
	import { idStore } from '../../designer/idDesignerStore.svelte.js';
	import { ladeAusweisDesign } from '../../designer/ausweisDesignLaden.js';
	import PrintPreview from '../../designer/PrintPreview.svelte';

	/** @type {{ students: any[] }} */
	let { students } = $props();

	// Cache-Buster für die Barcode-Bilder, damit ein zweiter Druck nach einer Änderung
	// nicht die alten PNGs aus dem Browser-Cache zieht (wie im Designer).
	let timestamp = $state(Date.now());

	onMount(async () => {
		await ladeAusweisDesign();
		timestamp = Date.now();
	});
</script>

<PrintPreview {students} barcodeType={idStore.barcodeType} {timestamp} />
