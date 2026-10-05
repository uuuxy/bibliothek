<!-- @component PortalMeldungen — die eigenen Meldungen der Lehrkraft mit ihrem Stand. Steht
     unter der Suche des Portals, solange nichts gesucht wird; „Problem melden" steht darüber
     in der Zeile unter dem Suchfeld (PortalProblemZeile). -->
<script>
	import LadeFehler from '../ui/LadeFehler.svelte';

	/** @typedef {{ id: string, art: string, titel_text: string, klasse: string, kommentar?: string, erstellt_am: string, erledigt_am?: string, erledigt_notiz?: string }} Anliegen */

	// Die Liste gehört dem Portal (eigeneAnliegen): Nach dem Absenden einer Meldung liest es
	// sie neu.
	/** @type {{ anliegen: Anliegen[], onaktualisiert: () => void | Promise<void>, ladefehler?: boolean }} */
	let { anliegen, onaktualisiert, ladefehler = false } = $props();
	const eigene = $derived(anliegen);
</script>

<section class="flex w-full max-w-3xl flex-col gap-3">
	<!-- Ein gescheiterter erster Abruf sagt nichts über die Meldungen — dann steht hier der
	     Ausfall und nicht die leere Liste. „Nichts da" ließe eine abgeschickte Meldung als
	     verloren erscheinen, und der nächste Schritt wäre, sie noch einmal zu schicken. -->
	{#if ladefehler}
		<LadeFehler
			onerneut={onaktualisiert}
			titel="Deine Meldungen konnten nicht geladen werden"
			text="Bitte später noch einmal versuchen. Schon abgeschickte Meldungen sind nicht verloren — sie sind bei der Bibliothek."
		/>
	{:else if eigene.length > 0}
		<h2 class="text-base font-medium text-on-surface">Deine Meldungen</h2>
		<ul class="divide-y divide-outline-variant">
			{#each eigene as a (a.id)}
				<li class="flex items-start justify-between gap-4 py-2.5">
					<div class="min-w-0 flex-1">
						<p class="truncate text-sm text-on-surface">
							<span class="font-semibold">{a.art === 'wunsch' ? 'Wunsch' : 'Meldung'}:</span>
							{a.titel_text}
							{#if a.klasse}<span class="text-on-surface-variant">· {a.klasse}</span>{/if}
						</p>
						{#if a.erledigt_am && a.erledigt_notiz}
							<p class="mt-0.5 text-xs text-on-surface-variant italic">
								Bibliothek: „{a.erledigt_notiz}"
							</p>
						{/if}
					</div>
					<span
						class="inline-flex shrink-0 items-center rounded-full px-2 py-0.5 text-label-small font-semibold {a.erledigt_am
							? 'bg-secondary-container text-on-secondary-container'
							: 'border border-outline-variant bg-surface text-on-surface-variant'}"
					>
						{a.erledigt_am ? 'Erledigt' : 'Offen'}
					</span>
				</li>
			{/each}
		</ul>
	{/if}
</section>
