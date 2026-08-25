<!-- @component KlassenSuchfeld — Klassensuche mit Vorschlagsliste.

     Bewusst KEIN Select: Hier wird getippt und gefiltert, nicht aus einer festen
     Liste gewählt — bei über hundert Klassen ist Tippen der kürzere Weg. Der
     150-ms-Verzug beim Verlassen ist Absicht: Ohne ihn schließt die Liste, bevor
     der Klick auf einen Eintrag ankommt.

     Stand in der Werkzeugleiste der Schulklassen-Seite, seit der Klassen-Reiter im
     Medienkatalog aufgelöst wurde. Das Feld selbst ist seit dem 25.08.2026 das
     Suchfeld-Bauteil (36 px, Werkzeugleiste); der Pfeil sitzt als nachlaufendes
     Symbol darin und die Vorschlagsliste hängt an der Hülle. -->
<script>
	import { ChevronDown } from '@lucide/svelte';
	import Suchfeld from '../../../lib/components/ui/Suchfeld.svelte';
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
		<Suchfeld
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
		</Suchfeld>
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
