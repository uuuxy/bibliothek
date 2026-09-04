<!-- @component KlassenSuchfeld — Klassensuche mit Vorschlagsliste.

     Bewusst KEIN Select: Hier wird getippt und gefiltert, nicht aus einer festen
     Liste gewählt — bei über hundert Klassen ist Tippen der kürzere Weg. Der
     150-ms-Verzug beim Verlassen ist Absicht: Ohne ihn schließt die Liste, bevor
     der Klick auf einen Eintrag ankommt.

     Es ist die EINE Suche der Klassensatz-Seite und trägt deshalb seit dem 04.09.2026
     die 48-px-Suchpille — dieselbe Gestalt wie Medienkatalog, Portal und Theke (Peter:
     „eine Leiste … es soll gleich aussehen"). Bis dahin war es das 36-px-Suchfeld im
     Werkzeugbalken, weil darüber noch die globale Suchleiste stand. Der Pfeil sitzt als
     nachlaufendes Symbol darin, die Vorschlagsliste hängt an der Hülle. -->
<script>
	import { ChevronDown } from '@lucide/svelte';
	import Suchpille from '../../../lib/components/ui/Suchpille.svelte';
	/**
	 * @type {{
	 *   klasseSearchQuery: string,
	 *   isKlasseDropdownOpen: boolean,
	 *   filteredKlassenList: string[],
	 *   onSelectKlasse?: (klasse: string) => void,
	 *   class?: string
	 * }}
	 */
	let {
		klasseSearchQuery = $bindable(''),
		isKlasseDropdownOpen = $bindable(false),
		filteredKlassenList = [],
		onSelectKlasse,
		class: className = ''
	} = $props();
</script>

<div class={className}>
	<div class="relative w-full">
		<Suchpille
			id="klassensaetze-suchfeld"
			bind:wert={klasseSearchQuery}
			etikett="Klasse suchen"
			platzhalter="Klasse suchen (z.B. 5f1) …"
			onfocus={() => (isKlasseDropdownOpen = true)}
			onblur={() => setTimeout(() => (isKlasseDropdownOpen = false), 150)}
		>
			{#snippet nachlaufend()}
				<ChevronDown
					class="pointer-events-none h-4 w-4 text-on-surface-variant transition-transform duration-200 {isKlasseDropdownOpen
						? 'rotate-180'
						: ''}"
					aria-hidden="true"
				/>
			{/snippet}
		</Suchpille>
		{#if isKlasseDropdownOpen && filteredKlassenList.length > 0}
			<ul
				class="absolute z-10 w-full mt-1.5 bg-surface-container rounded-sm shadow-xl max-h-60 overflow-y-auto py-1"
			>
				{#each filteredKlassenList as klasse (klasse)}
					<li>
						<button
							type="button"
							class="w-full text-left px-5 py-2.5 text-slate-700 hover:bg-blue-50 hover:text-blue-700 transition-colors duration-150 cursor-pointer text-sm font-medium"
							onclick={() => onSelectKlasse?.(klasse)}
						>
							Klasse {klasse}
						</button>
					</li>
				{/each}
			</ul>
		{:else if isKlasseDropdownOpen && filteredKlassenList.length === 0}
			<div
				class="absolute z-10 w-full mt-1.5 bg-surface-container rounded-sm shadow-xl py-4 px-5 text-slate-400 text-center text-sm"
			>
				Keine Klasse gefunden.
			</div>
		{/if}
	</div>
</div>
