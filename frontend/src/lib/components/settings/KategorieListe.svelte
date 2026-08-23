<script>
	/**
	 * @component KategorieListe
	 * Die Kategorienliste der Einstellungen — links auf breiten Bildschirmen, auf
	 * schmalen die ganze Seite (Material 3 „list-detail": erst die Liste, dann das
	 * Detail, zurück über den Pfeil).
	 *
	 * Sie ersetzt sechs Reiter, von denen einer sieben fremde Themen trug und einer
	 * genau einen Inhalt hatte. Reiter sind für drei bis fünf gleichgewichtige
	 * Bereiche gedacht; Einstellungen sind eine Aufzählung ungleicher Themen, und
	 * dafür kennt M3 die Liste mit führendem Symbol, Titel und einer Zeile Beitext.
	 *
	 * Der Beitext ist kein Zierrat: Er beantwortet die Frage, die einen überhaupt auf
	 * diese Seite geführt hat („wo stelle ich die Mahnfrist ein?"), ohne dass man
	 * sieben Bildschirme durchklickt.
	 *
	 * @prop {{ id: string, titel: string, kurz: string, icon: unknown }[]} kategorien
	 * @prop {string} aktiv
	 * @prop {(id: string) => void} onwahl
	 */
	let { kategorien, aktiv, onwahl } = $props();
</script>

<nav aria-label="Einstellungs-Kategorien" class="flex w-full shrink-0 flex-col gap-1 lg:w-80">
	{#each kategorien as k (k.id)}
		{@const gewaehlt = k.id === aktiv}
		<button
			type="button"
			aria-current={gewaehlt ? 'page' : undefined}
			onclick={() => onwahl(k.id)}
			class="flex w-full cursor-pointer items-center gap-4 rounded-2xl px-4 py-3 text-left transition-colors {gewaehlt
				? 'bg-secondary-container text-on-secondary-container'
				: 'text-on-surface-variant hover:bg-surface-container'}"
		>
			<k.icon size={20} strokeWidth={gewaehlt ? 2.25 : 2} class="shrink-0" />
			<span class="flex min-w-0 flex-col">
				<span class="truncate text-sm font-medium">{k.titel}</span>
				<span class="truncate text-xs {gewaehlt ? 'opacity-80' : 'text-on-surface-variant'}"
					>{k.kurz}</span
				>
			</span>
		</button>
	{/each}
</nav>
