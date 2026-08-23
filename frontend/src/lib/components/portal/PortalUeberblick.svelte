<script>
	/**
	 * @component PortalUeberblick
	 * Was auf der Startfläche des Kollegiums-Portals steht, solange nichts gesucht wurde.
	 *
	 * Vorher stand dort ein 340 px hohes Poster: ein Buchsymbol, „Suche nach einem Buch"
	 * und darunter „Titel, Autor oder ISBN eingeben" — wörtlich der Platzhalter, der
	 * einen Zentimeter darüber schon im Suchfeld steht. Es hat die Seite gefüllt, ohne
	 * etwas zu sagen (Peters Befund, 23.08.2026).
	 *
	 * Jetzt steht dort, was gerade läuft: die Warteschlange der Klassensätze und die
	 * eigenen Anliegen. Beides ist ohnehin die Antwort auf die Frage, mit der eine
	 * Lehrkraft das Portal öffnet — „ist mein Satz durch?".
	 *
	 * Die Anliegen kommen als Prop, nicht aus einem eigenen Abruf: Das Portal hält sie
	 * ohnehin für den Zähler am Reiter, und ein zweiter GET auf dieselbe Liste hätte
	 * zwei Wahrheiten über denselben Zustand erzeugt.
	 *
	 * @prop {{ titel_id: string, titel: string, klasse: string, anzahl: number, erstellt_am: string }[]} reservierungen
	 * @prop {{ id: string, art: string, titel_text: string, klasse: string, erstellt_am: string, erledigt_am?: string }[]} anliegen
	 * @prop {() => void} onanliegen - Wechselt in den Anliegen-Reiter.
	 */
	import Button from '../ui/Button.svelte';

	/** @type {{ reservierungen: { titel_id: string, titel: string, klasse: string, anzahl: number, erstellt_am: string }[], anliegen: { id: string, art: string, titel_text: string, klasse: string, erstellt_am: string, erledigt_am?: string }[], onanliegen: () => void }} */
	let { reservierungen, anliegen, onanliegen } = $props();

	const eigene = $derived(anliegen);
	const offene = $derived(eigene.filter((a) => !a.erledigt_am));
</script>

<div class="flex w-full max-w-3xl flex-col gap-10 pt-4">
	<section class="flex flex-col gap-3">
		<h2 class="text-base font-medium text-on-surface">Offene Klassensätze</h2>
		{#if reservierungen.length === 0}
			<p class="text-sm text-on-surface-variant">
				Zurzeit wartet kein Klassensatz. Suche oben einen Titel, um einen zu reservieren.
			</p>
		{:else}
			<ul class="divide-y divide-outline-variant">
				{#each reservierungen as r, _i (_i)}
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
	</section>

	<section class="flex flex-col gap-3 border-t border-outline-variant pt-8">
		<div class="flex items-center justify-between gap-4">
			<h2 class="text-base font-medium text-on-surface">Deine Anliegen</h2>
			<Button variant="ghost" size="sm" onclick={onanliegen}>Alle ansehen</Button>
		</div>
		{#if eigene.length === 0}
			<p class="text-sm text-on-surface-variant">
				Noch nichts gemeldet. Buchwünsche und Meldungen stehen im Reiter nebenan.
			</p>
		{:else}
			<ul class="divide-y divide-outline-variant">
				{#each eigene.slice(0, 3) as a (a.id)}
					<li class="flex items-start justify-between gap-4 py-2.5">
						<span class="min-w-0 flex-1 truncate text-sm text-on-surface">
							{a.art === 'wunsch' ? 'Wunsch' : 'Meldung'}: {a.titel_text}
							{#if a.klasse}<span class="text-on-surface-variant">· {a.klasse}</span>{/if}
						</span>
						<span
							class="shrink-0 rounded-full px-2 py-0.5 text-label-small font-medium {a.erledigt_am
								? 'bg-secondary-container text-on-secondary-container'
								: 'border border-outline-variant text-on-surface-variant'}"
						>
							{a.erledigt_am ? 'Erledigt' : 'Offen'}
						</span>
					</li>
				{/each}
			</ul>
			{#if eigene.length > 3}
				<p class="text-xs text-on-surface-variant">
					{eigene.length - 3} weitere · {offene.length} davon offen
				</p>
			{/if}
		{/if}
	</section>
</div>
