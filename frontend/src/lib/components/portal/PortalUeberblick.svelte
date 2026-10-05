<script>
	/**
	 * @component PortalUeberblick
	 * Was unter der Suche des Kollegiums-Portals steht, solange nichts gesucht wird: die
	 * eigenen Reservierungen — die Antwort auf die Frage, mit der eine Lehrkraft das Portal
	 * öffnet („ist mein Satz durch?"). Darunter folgen die eigenen Meldungen
	 * (PortalMeldungen).
	 *
	 * Gezeigt werden nur die eigenen Reservierungen, offene und bereitgestellte. Die Antwort
	 * der Bibliothek („Bibliothek: …") steht hier dauerhaft und trägt auch, wenn die
	 * Bereit-Mail scheitert. Die Warteschlange aller steht als Abzeichen an den Suchtreffern.
	 *
	 * @typedef {{ id: string, titel: string, klasse: string, anzahl: number,
	 *   erledigt: boolean, erledigt_notiz?: string, erledigt_am?: string,
	 *   erstellt_am: string }} MeineReservierung
	 */

	/** @type {{ reservierungen: MeineReservierung[] }} */
	let { reservierungen } = $props();

	const offene = $derived(reservierungen.filter((r) => !r.erledigt));
	const bereitgestellte = $derived(reservierungen.filter((r) => r.erledigt));
</script>

<section class="flex w-full max-w-3xl flex-col gap-3 pt-4">
	{#if reservierungen.length === 0}
		<p class="text-sm text-on-surface-variant">
			Zurzeit wartet keine Reservierung — suche oben einen Titel, um einen Klassensatz zu
			reservieren.
		</p>
	{:else}
		{#if offene.length > 0}
			<h2 class="text-base font-medium text-on-surface">Deine Reservierungen</h2>
			<ul class="divide-y divide-outline-variant">
				{#each offene as r (r.id)}
					<li class="flex items-center justify-between gap-4 py-2.5 text-sm">
						<span class="min-w-0 flex-1 truncate text-on-surface">
							{r.titel || 'Titel nicht mehr im Katalog'}
							<span class="text-on-surface-variant">· Klasse {r.klasse}</span>
						</span>
						<span class="shrink-0 text-on-surface-variant"
							>{r.anzahl} Stück · seit {r.erstellt_am}</span
						>
					</li>
				{/each}
			</ul>
			<p class="text-xs text-on-surface-variant">
				Reservieren sperrt nichts — wer denselben Titel will, stellt sich an.
			</p>
		{/if}
		{#if bereitgestellte.length > 0}
			<h2 class="text-base font-medium text-on-surface">Bereitgestellt</h2>
			<ul class="divide-y divide-outline-variant">
				{#each bereitgestellte as r (r.id)}
					<li class="flex items-start justify-between gap-4 py-2.5 text-sm">
						<div class="min-w-0 flex-1">
							<p class="truncate text-on-surface">
								{r.titel || 'Titel nicht mehr im Katalog'}
								<span class="text-on-surface-variant">· Klasse {r.klasse}</span>
							</p>
							{#if r.erledigt_notiz}
								<p class="mt-0.5 text-xs italic text-on-surface-variant">
									Bibliothek: „{r.erledigt_notiz}"
								</p>
							{/if}
						</div>
						<span
							class="shrink-0 inline-flex items-center rounded-full bg-secondary-container px-2 py-0.5 text-label-small font-semibold text-on-secondary-container"
						>
							bereit{r.erledigt_am ? ` · ${r.erledigt_am}` : ''}
						</span>
					</li>
				{/each}
			</ul>
		{/if}
	{/if}
</section>
