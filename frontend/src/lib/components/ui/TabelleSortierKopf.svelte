<!-- @component TabelleSortierKopf — eine sortierbare Kopfzelle für ui/Tabelle.

     Ein eigenes Bauteil, weil die Frage bei jeder Liste wiederkommt und jede eigene Fassung
     ein eigenes Rezept wäre.

     Die Zelle selbst trägt nur Layout (Breite, Ausrichtung); das Aussehen kommt aus dem
     <style> von Tabelle.svelte, die Ratsche frontend-hygiene-tabellen prüft das. Bedient
     wird über einen echten <button> in der Zelle: Ein klickbares <th> ohne Knopf erreicht
     die Tastatur nicht.

     `aria-sort` sitzt an der Zelle, nicht am Knopf — so verlangt es ARIA, und nur so
     liest ein Screenreader „aufsteigend sortiert" beim Betreten der Spalte.

     Ohne `onsortiere` ist das Bauteil ein gewöhnlicher Kopf. Dieselbe Liste steht auch
     dort, wo Sortieren nichts bewirkt (Kiosk, Abgänger, Schülersuche der Vormerkung) —
     ein Pfeil, der nichts tut, wäre dort schlimmer als keiner. -->
<script>
	import Icon from './Icon.svelte';

	/**
	 * @type {{
	 *   spalte: string,
	 *   text: string,
	 *   sortierung?: { spalte: string, absteigend: boolean },
	 *   onsortiere?: (spalte: string) => void,
	 *   class?: string
	 * }}
	 */
	let {
		spalte,
		text,
		sortierung = { spalte: '', absteigend: false },
		onsortiere,
		class: klasse = ''
	} = $props();

	const aktiv = $derived(sortierung.spalte === spalte);
	const ariaSortierung = $derived(
		!aktiv ? 'none' : sortierung.absteigend ? 'descending' : 'ascending'
	);
	// Was der nächste Klick TUT, steht im Titel — nicht, was gerade gilt. „Nach Name
	// sortieren" an einer schon sortierten Spalte wäre eine Lüge über den Knopf.
	const titel = $derived(
		!aktiv
			? `Nach ${text} sortieren`
			: sortierung.absteigend
				? `Nach ${text} aufsteigend sortieren`
				: `Nach ${text} absteigend sortieren`
	);
</script>

{#if !onsortiere}
	<th class={klasse}>{text}</th>
{:else}
	<th class={klasse} aria-sort={ariaSortierung}>
		<button
			type="button"
			onclick={() => onsortiere(spalte)}
			data-tip={titel}
			aria-label={titel}
			class="inline-flex cursor-pointer items-center gap-1 rounded-sm"
		>
			{text}
			{#if !aktiv}
				<!-- Der Doppelpfeil sagt „hier kann man sortieren", ohne eine Richtung zu
				     behaupten. Ohne Symbol sähe die Spalte aus wie eine gewöhnliche. -->
				<Icon name="unfold_more" groesse={18} class="opacity-50" />
			{:else if sortierung.absteigend}
				<Icon name="arrow_downward" groesse={18} class="text-primary" />
			{:else}
				<Icon name="arrow_upward" groesse={18} class="text-primary" />
			{/if}
		</button>
	</th>
{/if}
