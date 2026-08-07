<!-- @component StudentBatchPrint — versteckte Druckfläche für einen Stapel Ausweise aus
     der Schülerdatei.

     Rendert über PrintPreview und damit über dieselbe CardFace-Quelle wie der
     Einzeldruck im Profil (StudentPrintCard) und die Vorschau im Ausweis-Designer.
     Drei Wege, ein optisches Ergebnis — ein eigener Renderer hier hätte über kurz oder
     lang eine zweite Kartenoptik ergeben, die niemand pflegt.

     Das Ausweis-Design wird zentral geladen (GET /api/ausweis-layout), genau wie im
     Profil. Ohne das stünden hier die Store-Defaults und der Stapel sähe anders aus als
     die Karte, die man vorher im Designer eingerichtet hat.

     Kartendrucker oder A4-Bogen entscheidet NICHT dieser Bildschirm, sondern das
     gespeicherte Design (idStore.printMode). Der Ausweis-Designer legt fest, wie
     gedruckt wird; die Schülerdatei legt fest, WER gedruckt wird. -->
<script>
	import { onMount } from 'svelte';
	import { idStore, applyDesign } from '../../designer/idDesignerStore.svelte.js';
	import { apiFetch } from '../../apiFetch.js';
	import PrintPreview from '../../designer/PrintPreview.svelte';

	/** @type {{ students: any[] }} */
	let { students } = $props();

	// Cache-Buster für die Barcode-Bilder, damit ein zweiter Druck nach einer Änderung
	// nicht die alten PNGs aus dem Browser-Cache zieht (wie im Designer).
	let timestamp = $state(Date.now());

	onMount(async () => {
		try {
			const res = await apiFetch('/api/ausweis-layout');
			if (res.ok) applyDesign(await res.json());
		} catch (e) {
			console.error('Ausweis-Design konnte nicht geladen werden:', e);
		}
		timestamp = Date.now();
	});
</script>

<PrintPreview {students} barcodeType={idStore.barcodeType} {timestamp} />
