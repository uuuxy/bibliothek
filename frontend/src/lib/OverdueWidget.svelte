<script>
	import { apiFetch } from './apiFetch.js';
	import { uiStore } from './stores/uiStore.svelte.js';

	/** @type {any} */
	let summary = $state(null);
	let loading = $state(true);

	async function fetchSummary() {
		try {
			const res = await apiFetch('/api/dashboard/summary');
			if (res.ok) {
				summary = await res.json();
			}
		} catch (err) {
			console.error(err);
		} finally {
			loading = false;
		}
	}

	$effect(() => {
		fetchSummary();
	});
</script>

{#if loading}
	<div class="flex justify-center h-full items-center py-10">
		<div
			class="w-6 h-6 border-2 border-t-rose-500 border-rose-500/20 rounded-full animate-spin"
		></div>
	</div>
{:else if summary}
	<!-- Flaches Alert mit Links-Akzent statt umschließender Karte -->
	<div class="border-l-4 border-rose-500 pl-5 flex flex-col h-full">
		<!-- Header: klickbar → springt aufs Mahnwesen (eigene, vollwertige Seite;
		     deshalb Navigation statt eines dritten Slide-in-Panels wie bei Renner/Ladenhüter) -->
		<button
			type="button"
			onclick={() => (uiStore.activeTab = 'mahnwesen')}
			class="w-full flex justify-between items-center pb-4 border-b border-gray-200 group cursor-pointer text-left"
			aria-label="Mahnwesen öffnen — alle überfälligen Ausleihen"
		>
			<div>
				<h3 class="text-rose-700 font-bold text-base flex items-center gap-1.5">
					Dringend: Mahnungen
					<svg
						class="w-3.5 h-3.5 text-rose-300 transition-all group-hover:text-rose-600 group-hover:translate-x-0.5 group-hover:-translate-y-0.5"
						fill="none"
						stroke="currentColor"
						viewBox="0 0 24 24"
						><path
							stroke-linecap="round"
							stroke-linejoin="round"
							stroke-width="2.5"
							d="M7 17L17 7M7 7h10v10"
						/></svg
					>
				</h3>
				<p class="text-sm text-gray-600 mt-0.5">Überfällige Ausleihen gesamt</p>
			</div>
			<div class="text-rose-600 font-extrabold text-4xl tabular-nums">
				{summary.total_overdue}
			</div>
		</button>

		<!-- Top 5 List -->
		<div class="pt-4 flex-1 pb-6">
			<h4 class="text-sm font-medium text-gray-600 mb-3">Am längsten überfällig (Härtefälle)</h4>
			{#if summary.top_overdue && summary.top_overdue.length > 0}
				<ul class="space-y-3">
					{#each summary.top_overdue as item, _i (_i)}
						<li class="flex justify-between items-start text-sm">
							<div class="min-w-0 pr-2">
								<span class="block font-bold text-slate-800 truncate"
									>{item.schueler_name}
									<span class="text-slate-400 font-semibold text-xs ml-1">({item.klasse})</span
									></span
								>
								<span class="block text-slate-500 text-xs font-medium truncate">{item.titel}</span>
							</div>
							<div class="shrink-0 text-right">
								<span class="text-rose-600 font-bold bg-rose-50 px-2 py-0.5 rounded text-xs"
									>{item.tage} Tage</span
								>
							</div>
						</li>
					{/each}
				</ul>
			{:else}
				<p class="text-sm text-slate-500 italic py-2">Keine überfälligen Bücher. Alles im Lot!</p>
			{/if}
		</div>
	</div>
{/if}
