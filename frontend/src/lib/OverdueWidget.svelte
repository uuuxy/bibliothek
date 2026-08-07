<script>
	import { apiFetch } from './apiFetch.js';
	import { uiStore } from './stores/uiStore.svelte.js';
	import { ChevronRight, CircleCheck } from '@lucide/svelte';

	/** aktuellVerliehen: laufende Ausleihen — Basis für die neutrale Überfälligkeitsquote.
	 * @type {{ aktuellVerliehen?: number }} */
	let { aktuellVerliehen = 0 } = $props();

	/** @type {any} */
	let summary = $state(null);
	let loading = $state(true);

	const hatMahnungen = $derived((summary?.total_overdue ?? 0) > 0);

	// Neutrale Überfälligkeitsquote = überfällig ÷ laufende Ausleihen. Rein informativ,
	// keine Ampel — die Statistik-Seite ist ein Analyse-Kontext, kein Alarm.
	const quote = $derived(
		aktuellVerliehen > 0
			? Math.round(((summary?.total_overdue ?? 0) / aktuellVerliehen) * 100)
			: null
	);

	// Anonyme Dauer-Verteilung statt Klarnamen (Datenminimierung, Art. 5 DSGVO).
	const buckets = $derived(summary?.overdue_buckets ?? []);
	const maxBucket = $derived(Math.max(1, ...buckets.map((b) => b.count)));

	async function fetchSummary() {
		try {
			const res = await apiFetch('/api/dashboard/summary');
			if (res.ok) summary = await res.json();
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
	<div class="flex-1 flex justify-center items-center py-8">
		<div
			class="w-6 h-6 border-2 border-t-slate-400 border-slate-200 rounded-full animate-spin"
		></div>
	</div>
{:else if summary}
	<!-- NEUTRAL (kein Rot-Alarm): Überfälligkeit als reine Statistik. Zahl + Quote in Slate,
	     Verteilung in Grau (nur „>60 Tage" ein dezenter Amber-Akzent). Die operative
	     Bearbeitung liegt im Mahnwesen — hier nur ein zurückhaltender Textlink dorthin.
	     Layout: vertikal gestapelt, weil das Widget in der schmalen 1/3-Spalte der
	     Bento-Reihe sitzt — nebeneinander bräche es dort um. -->
	<div class="h-full flex flex-col">
		<h3 class="text-xs font-medium text-slate-500">Überfällige Ausleihen</h3>
		<div class="flex items-baseline gap-2 mt-2">
			<span class="text-4xl font-light text-slate-900 tabular-nums leading-none"
				>{summary.total_overdue}</span
			>
			{#if quote !== null && hatMahnungen}
				<span class="text-xs text-slate-400">≈ {quote}% der laufenden Ausleihen</span>
			{/if}
		</div>

		{#if hatMahnungen}
			<div class="flex items-baseline justify-between gap-2 mt-5 mb-2.5">
				<h4 class="text-label-small font-medium text-slate-400">Verteilung nach Dauer</h4>
				<span class="text-label-small text-slate-400 shrink-0"
					>längste: {summary.max_tage_overdue} Tage</span
				>
			</div>
			<!-- 2 Spalten fix (nicht viewport-abhängig): die Card ist immer schmal. -->
			<div class="grid grid-cols-2 gap-x-4 gap-y-2.5 min-h-0 overflow-y-auto">
				{#each buckets as bucket, _i (_i)}
					{@const alt = bucket.label === 'über 60 Tage' && bucket.count > 0}
					<div>
						<div class="flex items-baseline justify-between gap-2 mb-1.5">
							<span class="text-xs font-medium text-slate-500 truncate">{bucket.label}</span>
							<span class="text-sm font-semibold tabular-nums text-slate-700">{bucket.count}</span>
						</div>
						<!-- Grau; nur die längst-überfällige Gruppe bekommt einen dezenten Amber-Akzent. -->
						<div class="h-1.5 w-full rounded-full bg-slate-100 overflow-hidden">
							<div
								class="h-full rounded-full {alt ? 'bg-amber-500' : 'bg-slate-400'}"
								style="width: {(bucket.count / maxBucket) * 100}%"
							></div>
						</div>
					</div>
				{/each}
			</div>
		{:else}
			<div class="flex-1 flex items-center gap-2 text-slate-500 text-sm">
				<CircleCheck class="w-5 h-5 shrink-0 text-emerald-500" aria-hidden="true" />
				Keine überfälligen Ausleihen.
			</div>
		{/if}

		<button
			type="button"
			onclick={() => (uiStore.activeTab = 'mahnwesen')}
			class="mt-auto pt-3 border-t border-slate-100 inline-flex items-center justify-between gap-1 text-xs font-semibold text-blue-600 hover:text-blue-800 transition-colors cursor-pointer"
			aria-label="Zum Mahnwesen — überfällige Ausleihen bearbeiten"
		>
			Im Mahnwesen bearbeiten
			<ChevronRight class="w-3.5 h-3.5" aria-hidden="true" />
		</button>
	</div>
{/if}
