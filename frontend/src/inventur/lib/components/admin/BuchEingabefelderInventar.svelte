<script>
	import Feld from '../../../../lib/components/ui/Feld.svelte';

	let { formular = $bindable() } = $props();
</script>

<div class="grid grid-cols-2 gap-4">
	<Feld
		id="buch-bestand"
		label="Aktueller Bestand"
		type="number"
		bind:value={formular.stock}
		feld="pr-14"
	>
		{#snippet nachlaufend()}Stück{/snippet}
	</Feld>
	<Feld id="buch-zaehldatum" label="Zähldatum" type="date" bind:value={formular.lastCounted} />
</div>

{#if formular.erweiterteEigenschaften}
	<!-- Signatur lebt jetzt als Pflichtfeld direkt unter Titel/Autor
         (BuchEingabefelder) und schreibt die echte DB-Spalte. -->
	<div class="grid grid-cols-2 gap-4">
		<Feld
			id="buch-standort"
			label="Standort / Regal"
			bind:value={formular.erweiterteEigenschaften.standort}
			placeholder="z. B. Krimi-Ecke oder Regal 3B"
		/>
	</div>
{/if}
