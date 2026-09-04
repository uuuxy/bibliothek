<script>
	/**
	 * @component KlassensatzErledigte
	 * Die zuletzt bereitgestellten Klassensätze samt Antwort der Bibliothek — der
	 * Rückweg der Abschluss-Notiz für die Theke. Bis zum 31.08.2026 filterte die
	 * Liste Erledigte komplett aus, und die Notiz („24 von 30, Rest bei der 8a")
	 * existierte nur in der Bereit-Mail: Scheiterte die oder fehlte die Adresse
	 * (Reservierung ohne Konto), konnte niemand mehr nachschlagen, was zugesagt war.
	 *
	 * @typedef {{ id: string, titel_name: string, klasse: string, anzahl: number,
	 *   erledigt_notiz?: string, erledigt_am?: string }} Erledigte
	 */

	/** @type {{ erledigte: Erledigte[] }} */
	let { erledigte } = $props();

	// Die Liste kommt vom Server „Erledigte neueste zuerst" — hier nur die jüngsten
	// zehn: Die Theke schlägt nach, sie archiviert nicht.
	const juengste = $derived(erledigte.slice(0, 10));
</script>

{#if juengste.length > 0}
	<div>
		<h3 class="text-base font-semibold text-on-surface">Zuletzt bereitgestellt</h3>
		<ul class="divide-y divide-outline-variant">
			{#each juengste as r (r.id)}
				<li class="flex items-start justify-between gap-4 py-2 text-sm">
					<div class="min-w-0 flex-1">
						<p class="truncate text-on-surface">
							{r.titel_name}
							<span class="text-on-surface-variant">· Klasse {r.klasse} · {r.anzahl} Stück</span>
						</p>
						{#if r.erledigt_notiz}
							<p class="mt-0.5 text-xs italic text-on-surface-variant">
								Notiz: „{r.erledigt_notiz}"
							</p>
						{/if}
					</div>
					{#if r.erledigt_am}
						<span class="shrink-0 text-xs text-on-surface-variant">{r.erledigt_am}</span>
					{/if}
				</li>
			{/each}
		</ul>
	</div>
{/if}
