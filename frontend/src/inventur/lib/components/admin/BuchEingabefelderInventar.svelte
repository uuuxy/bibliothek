<script>
	import Feld from '../../../../lib/components/ui/Feld.svelte';

	let { formular = $bindable() } = $props();
</script>

<div class="grid grid-cols-2 gap-4">
	<!-- Eigene Subgrid-Hülle statt `label=` am Feld: Die Einheit „Stück" liegt als
	     Überlagerung IM Feld, und dafür braucht die Feldzeile einen eigenen
	     relative-Kasten. Drei Zeilen wie beim Feld mit label, damit das Zähldatum
	     daneben auf gleicher Höhe steht. -->
	<div class="row-span-3 grid grid-rows-subgrid gap-y-1.5">
		<label for="buch-bestand" class="text-sm font-medium text-on-surface-variant"
			>Aktueller Bestand</label
		>
		<div class="relative">
			<Feld id="buch-bestand" type="number" bind:value={formular.stock} feld="pr-14" />
			<span
				class="pointer-events-none absolute inset-y-0 right-3 flex items-center text-sm text-on-surface-variant"
				>Stück</span
			>
		</div>
	</div>
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
