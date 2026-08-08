<!-- @component KlassenSuchfeld — Klassensuche mit Vorschlagsliste.

     Bewusst KEIN Select: Hier wird getippt und gefiltert, nicht aus einer festen
     Liste gewählt — bei über hundert Klassen ist Tippen der kürzere Weg. Der
     150-ms-Verzug beim Verlassen ist Absicht: Ohne ihn schließt die Liste, bevor
     der Klick auf einen Eintrag ankommt.

     Stand in der Werkzeugleiste der Schulklassen-Seite, seit der Klassen-Reiter im
     Medienkatalog aufgelöst wurde. Feld-Vokabular deshalb wie Select.svelte
     (rounded-xl, border-outline-variant, Fokus auf primary) — die Zeile trägt
     Suchfeld, zwei Auswahlfelder und einen Button und soll als EINE Leiste lesen.
     Die Höhe kommt aus dem base-Layer (36 px), nicht aus den Klassen hier. -->
<script>
	import { ChevronDown } from '@lucide/svelte';
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
		<input
			type="text"
			bind:value={klasseSearchQuery}
			aria-label="Klasse suchen"
			onfocus={() => (isKlasseDropdownOpen = true)}
			onblur={() => setTimeout(() => (isKlasseDropdownOpen = false), 150)}
			placeholder="Klasse suchen (z.B. 5f1) …"
			class="block w-full rounded-xl border border-outline-variant bg-surface-container-lowest pl-3 pr-10 text-sm text-on-surface transition-colors placeholder:text-on-surface-variant focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
		/>
		<div
			class="pointer-events-none absolute inset-y-0 right-0 flex items-center px-3 text-on-surface-variant"
		>
			<ChevronDown
				class="h-4 w-4 transition-transform duration-200 {isKlasseDropdownOpen ? 'rotate-180' : ''}"
				aria-hidden="true"
			/>
		</div>
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
