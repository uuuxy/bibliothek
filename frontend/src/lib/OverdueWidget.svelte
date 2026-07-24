<script>
	import { apiFetch } from './apiFetch.js';
	import { uiStore } from './stores/uiStore.svelte.js';

	/** @type {any} */
	let summary = $state(null);
	let loading = $state(true);

	// Rot nur bei echten Mahnungen. Bei 0 ruhiger Grün-Zustand — kein Fehlalarm im Bestzustand.
	const hatMahnungen = $derived((summary?.total_overdue ?? 0) > 0);

	// Anonyme Dauer-Verteilung statt Klarnamen: Die Statistik-Seite ist ein Analyse-
	// Kontext; namentliche Bearbeitung läuft im Mahnwesen. Die Balken werden am grössten
	// Bucket skaliert, damit die Verteilung auf einen Blick lesbar ist.
	const buckets = $derived(summary?.overdue_buckets ?? []);
	const maxBucket = $derived(Math.max(1, ...buckets.map((b) => b.count)));

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
	<!-- Flaches Alert mit Links-Akzent statt umschließender Karte.
	     Farbwelt folgt dem Zustand: Rot nur bei offenen Mahnungen, sonst ruhiges Grün. -->
	<div
		class="border-l-4 {hatMahnungen ? 'border-rose-500' : 'border-emerald-400'} pl-5 flex flex-col h-full"
	>
		<!-- Header: klickbar → springt aufs Mahnwesen (eigene, vollwertige Seite;
		     deshalb Navigation statt eines dritten Slide-in-Panels wie bei Renner/Ladenhüter) -->
		<button
			type="button"
			onclick={() => (uiStore.activeTab = 'mahnwesen')}
			class="w-full flex justify-between items-start gap-3 pb-4 border-b border-gray-200 -mx-2 px-2 rounded-md group cursor-pointer text-left transition-colors {hatMahnungen
				? 'hover:bg-rose-50/50'
				: 'hover:bg-emerald-50/50'}"
			aria-label="Zum Mahnwesen — alle überfälligen Ausleihen"
		>
			<div class="space-y-1">
				<h3 class="{hatMahnungen ? 'text-rose-700' : 'text-emerald-700'} font-bold text-base">
					{hatMahnungen ? 'Dringend: Mahnungen' : 'Mahnungen'}
				</h3>
				<p class="text-sm text-gray-600">Überfällige Ausleihen gesamt</p>
				<!-- Explizites „Zum Mahnwesen": macht klar, dass dies in einen ANDEREN Bereich
				     springt (kein Statistik-Unterblatt) — deshalb bewusst kein „zurück zur Statistik". -->
				<span
					class="inline-flex items-center gap-1 text-[11px] font-bold px-2.5 py-1 rounded-full transition-colors {hatMahnungen
						? 'text-rose-600 bg-rose-50 group-hover:bg-rose-100'
						: 'text-emerald-600 bg-emerald-50 group-hover:bg-emerald-100'}"
				>
					Zum Mahnwesen
					<svg
						class="w-3 h-3 transition-transform group-hover:translate-x-0.5"
						fill="none"
						stroke="currentColor"
						viewBox="0 0 24 24"
						><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M9 5l7 7-7 7" /></svg
					>
				</span>
			</div>
			<div
				class="{hatMahnungen
					? 'text-rose-600'
					: 'text-emerald-600'} font-extrabold text-4xl tabular-nums shrink-0"
			>
				{summary.total_overdue}
			</div>
		</button>

		<!-- Anonyme Verteilung nach Überfälligkeitsdauer.
		     BEWUSST ohne Schülernamen/Titel: Die Statistik-Seite ist ein Analyse-Kontext
		     (Datenminimierung, Art. 5 DSGVO). Namentliche Bearbeitung → „Zum Mahnwesen". -->
		<div class="pt-4 flex-1 pb-6">
			<div class="flex items-baseline justify-between mb-3">
				<h4 class="text-sm font-medium text-gray-600">Verteilung nach Überfälligkeit</h4>
				{#if hatMahnungen}
					<span class="text-xs font-semibold text-slate-400">
						längste: <span class="text-rose-600 font-bold">{summary.max_tage_overdue} Tage</span>
					</span>
				{/if}
			</div>
			{#if hatMahnungen}
				<ul class="space-y-2.5">
					{#each buckets as bucket, _i (_i)}
						{@const kritisch = bucket.label === 'über 60 Tage' && bucket.count > 0}
						<li class="text-sm">
							<div class="flex items-center justify-between mb-1">
								<span class="text-xs font-semibold {kritisch ? 'text-rose-700' : 'text-slate-600'}"
									>{bucket.label}</span
								>
								<span
									class="text-xs font-bold tabular-nums {kritisch
										? 'text-rose-600'
										: 'text-slate-700'}">{bucket.count}</span
								>
							</div>
							<!-- Proportionaler Balken, am grössten Bucket skaliert -->
							<div class="h-1.5 w-full rounded-full bg-slate-100 overflow-hidden">
								<div
									class="h-full rounded-full {kritisch ? 'bg-rose-500' : 'bg-slate-400'}"
									style="width: {(bucket.count / maxBucket) * 100}%"
								></div>
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
