<!-- @component KlassenSuchfeld — Klassensuche mit Vorschlagsliste.

     Kein Select: Hier wird getippt und gefiltert, nicht aus einer festen Liste gewählt — bei
     über hundert Klassen ist Tippen der kürzere Weg. Der Verzug von 150 ms beim Verlassen
     hält die Liste offen, bis der Klick auf einen Eintrag ankommt.

     Es ist die eine Suche der Klassensatz-Seite und trägt deshalb die Suchpille von 48 px wie
     Medienkatalog, Portal und Theke. Der Pfeil sitzt als nachlaufendes Symbol darin, die
     Vorschlagsliste hängt an der Hülle. -->
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
							class="w-full text-left px-5 py-2.5 text-on-surface cursor-pointer text-sm font-medium"
							onclick={() => onSelectKlasse?.(klasse)}
						>
							Klasse {klasse}
						</button>
					</li>
				{/each}
			</ul>
		{:else if isKlasseDropdownOpen && filteredKlassenList.length === 0}
			<div
				class="absolute z-10 w-full mt-1.5 bg-surface-container rounded-sm shadow-xl py-4 px-5 text-on-surface-variant text-center text-sm"
			>
				Keine Klasse gefunden.
			</div>
		{/if}
	</div>
</div>
