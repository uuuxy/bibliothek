<script>
	import { Check } from '@lucide/svelte';

	/**
	 * @component FilterChips — die Filter-Chips aus Material 3, einfach wählbar.
	 *
	 * Wofür: Wörter, nach denen ein Inhalt gefiltert wird („Fantasy", „Krimi"). Ein Chip
	 * ist an oder aus, ein zweiter Klick nimmt ihn zurück, und alle dürfen aus sein — dann
	 * gilt kein Filter. M3, Chips: „Filter chips use tags or descriptive words to filter
	 * content. They can be a good alternative to toggle buttons or checkboxes."
	 * Nicht zu verwechseln mit den Nachbarn:
	 *   - Segmente (ui/Segmente) schalten zwischen Ansichten, von denen immer eine gilt.
	 *   - ChipFeld (ui/ChipFeld) nimmt Wörter entgegen, die jemand eintippt (Input-Chips).
	 *
	 * Einfachauswahl: Ein zweiter Chip löst den ersten ab. Material Components nennt beide
	 * Formen („app:singleSelection … to toggle single-select and multi-select behaviors");
	 * bei mehreren wäre offen, ob „Fantasy" und „Freundschaft" beides oder eins von beiden
	 * heißt. Eingegrenzt wird mit dem Suchtext daneben.
	 *
	 * Maße aus den M3-Token (material-web, _md-comp-filter-chip.scss): 32 px hoch, Ecke
	 * „small" (8 px = rounded-md in diesem Haus), Beschriftung label-large. Ungewählt:
	 * Umriss `outline`, Text `on-surface-variant`. Gewählt: Fläche `secondary-container`
	 * ohne Umriss, Text `on-secondary-container`, vorn ein 18-px-Häkchen — der Rand bleibt
	 * durchsichtig stehen, damit die Höhe beim Umschalten gleich bleibt. Die Chips stehen in
	 * einer benannten Gruppe (material-web: „Chips should always appear in a set … add an
	 * aria-label"). Rückmeldung beim Zeigen gibt der State-Layer, den jeder Knopf im Haus
	 * trägt (styles/komponenten.css).
	 *
	 * @prop {{ wert: string, text: string }[]} optionen
	 * @prop {string | null} wert - der gewählte Wert; null heißt kein Filter.
	 * @prop {(wert: string | null) => void} onwahl - bekommt null, wenn der gewählte Chip zurückgenommen wird.
	 * @prop {string} etikett - Name der Gruppe für Screenreader.
	 */
	/** @type {{ optionen: { wert: string, text: string }[], wert: string | null, onwahl: (wert: string | null) => void, etikett: string }} */
	let { optionen, wert, onwahl, etikett } = $props();
</script>

<div role="group" aria-label={etikett} class="flex flex-wrap gap-2">
	{#each optionen as o (o.wert)}
		{@const gewaehlt = o.wert === wert}
		<button
			type="button"
			aria-pressed={gewaehlt}
			onclick={() => onwahl(gewaehlt ? null : o.wert)}
			class="flex h-8 cursor-pointer items-center gap-2 rounded-md border text-sm font-medium transition-colors {gewaehlt
				? 'border-transparent bg-secondary-container pr-4 pl-2 text-on-secondary-container'
				: 'border-outline px-4 text-on-surface-variant'}"
		>
			{#if gewaehlt}
				<Check class="h-4.5 w-4.5 shrink-0" aria-hidden="true" />
			{/if}
			{o.text}
		</button>
	{/each}
</div>
