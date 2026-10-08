<script>
	import { onMount } from 'svelte';
	import { idStore } from './designer/idDesignerStore.svelte.js';
	import { ladeAusweisDesign } from './designer/ausweisDesignLaden.js';
	import CardFace from './designer/CardFace.svelte';
	import { kartenStil } from './designer/kartenFarben.js';

	/** @type {{ profile: any, timestamp: number }} */
	let { profile, timestamp } = $props();

	// Rückseite nur mitdrucken, wenn sie Inhalt hat (sonst leere zweite Kartenseite).
	const hasBack = $derived(idStore.back.elements.some((/** @type {any} */ e) => e.show));

	// Der Einzeldruck der Akte zeigt dasselbe Design wie der Stapeldruck: Beide zeichnen über
	// CardFace aus dem idStore, und beide laden ihn über denselben Lader.
	onMount(() => {
		ladeAusweisDesign();
	});
</script>

<!--
  Einzelkarten-Druckbereich (Profil → „Ausweis drucken").
  Auf dem Bildschirm ausgeblendet (display:none), per @media print sichtbar, wenn
  printCard() body[data-print-mode="card-single"] setzt. Außerhalb des .no-print-
  Wrappers gerendert, damit es die Druckunterdrückung überlebt.
-->
<div class="single-card-print-section" style="display:none" aria-hidden="true">
	<div class="print-card-box single-card-front" style={kartenStil(idStore.front.theme)}>
		<CardFace side="front" student={profile} barcodeType={idStore.barcodeType} {timestamp} />
	</div>
	{#if hasBack}
		<!-- Rückseite: student={null} — exakt wie im Batch-Druck (statischer Inhalt). -->
		<div class="print-card-box single-card-back" style={kartenStil(idStore.back.theme)}>
			<CardFace side="back" student={null} barcodeType={idStore.barcodeType} {timestamp} />
		</div>
	{/if}
</div>
