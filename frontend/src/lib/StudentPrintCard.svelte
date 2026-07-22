<script>
	import { onMount } from 'svelte';
	import { idStore, applyDesign } from './designer/idDesignerStore.svelte.js';
	import { apiFetch } from './apiFetch.js';
	import CardFace from './designer/CardFace.svelte';

	/** @type {{ profile: any, timestamp: number }} */
	let { profile, timestamp } = $props();

	// Rückseite nur mitdrucken, wenn sie Inhalt hat (sonst leere zweite Kartenseite).
	const hasBack = $derived(idStore.back.elements.some((/** @type {any} */ e) => e.show));

	// Zentrales Ausweis-Design laden, damit der profilseitige Einzeldruck EXAKT dasselbe
	// Layout wie der DruckCenter-Batchdruck zeigt. Beide rendern über CardFace aus
	// demselben idStore — es gibt nur noch ein optisches Ergebnis pro Ausweis, egal von
	// welchem Button/Arbeitsplatz gedruckt wird.
	onMount(async () => {
		try {
			const res = await apiFetch('/api/ausweis-layout');
			if (res.ok) applyDesign(await res.json());
		} catch (e) {
			console.error('Ausweis-Design konnte nicht geladen werden:', e);
		}
	});
</script>

<!--
  Einzelkarten-Druckbereich (Profil → „Ausweis drucken").
  Auf dem Bildschirm ausgeblendet (display:none), per @media print sichtbar, wenn
  printCard() body[data-print-mode="card-single"] setzt. Außerhalb des .no-print-
  Wrappers gerendert, damit es die Druckunterdrückung überlebt.
-->
<div class="single-card-print-section" style="display:none" aria-hidden="true">
	<div class="print-card-box single-card-front {idStore.front.theme}">
		<CardFace side="front" student={profile} barcodeType={idStore.barcodeType} {timestamp} />
	</div>
	{#if hasBack}
		<!-- Rückseite: student={null} — exakt wie im Batch-Druck (statischer Inhalt). -->
		<div class="print-card-box single-card-back {idStore.back.theme}">
			<CardFace side="back" student={null} barcodeType={idStore.barcodeType} {timestamp} />
		</div>
	{/if}
</div>
