<script>
	/**
	 * @component Reiter — die Reiterleiste der Anwendung (Material 3, Primary Tabs).
	 *
	 * Reiter waren bis zum 23.08.2026 vier Mal von Hand gebaut (Medienkatalog,
	 * Bestellwesen, Buch-Akte, Inventur-Startseite) und dabei auseinandergelaufen —
	 * dieselbe Geschichte wie bei den Suchfeldern, die in zehn Kopien an sieben Werten
	 * differierten. Diese Komponente ist der eine Ort; die vier Bestandsfälle stehen in
	 * `frontend-hygiene-reiter.test.js` als eingefrorener Rest und werden bei ihrem
	 * nächsten fachlichen Anfassen nachgezogen.
	 *
	 * M3 kennt Reiter für DREI BIS FÜNF gleichrangige Bereiche. Mehr Bereiche oder
	 * ungleichgewichtige gehören in eine Liste (siehe KategorieListe der Einstellungen,
	 * wo sechs Reiter mit einem überfüllten „Allgemein" genau daran scheiterten).
	 *
	 * `anzahl` ist der Zähler am Reiter — er sagt VOR dem Klick, ob dort etwas wartet.
	 *
	 * @prop {{ id: string, label: string, anzahl?: number }[]} reiter
	 * @prop {string} aktiv
	 * @prop {(id: string) => void} onwahl
	 * @prop {string} etikett - Name der Leiste für Screenreader.
	 */
	/** @type {{ reiter: { id: string, label: string, anzahl?: number }[], aktiv: string, onwahl: (id: string) => void, etikett: string }} */
	let { reiter, aktiv, onwahl, etikett } = $props();
</script>

<div role="tablist" aria-label={etikett} class="flex gap-6 border-b border-outline-variant">
	{#each reiter as r (r.id)}
		{@const gewaehlt = r.id === aktiv}
		<button
			type="button"
			role="tab"
			aria-selected={gewaehlt}
			onclick={() => onwahl(r.id)}
			class="relative cursor-pointer pb-3 text-sm font-medium transition-colors {gewaehlt
				? 'text-primary'
				: 'text-on-surface-variant hover:text-on-surface'}"
		>
			{r.label}
			{#if r.anzahl}
				<span
					class="ml-2 inline-flex min-w-5 justify-center rounded-full px-1.5 text-label-small {gewaehlt
						? 'bg-primary text-on-primary'
						: 'bg-surface-container-high text-on-surface-variant'}"
				>
					{r.anzahl}
				</span>
			{/if}
			{#if gewaehlt}
				<span class="absolute inset-x-0 bottom-0 h-0.5 rounded-full bg-primary"></span>
			{/if}
		</button>
	{/each}
</div>
