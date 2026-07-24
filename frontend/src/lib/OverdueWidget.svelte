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
	<div class="flex justify-center items-center py-8">
		<div class="w-6 h-6 border-2 border-t-rose-500 border-rose-500/20 rounded-full animate-spin"></div>
	</div>
{:else if summary}
	<!-- Horizontales Aktionsband für die Top-Position der Statistik: die dringendste Zahl
	     der Seite zuerst und auf einen Blick. Links Zahl + CTA (klickbar → Mahnwesen),
	     rechts die anonyme Dauer-Verteilung. Farbwelt folgt dem Zustand (Rot nur bei
	     offenen Mahnungen, sonst ruhiges Grün). -->
	<div
		class="rounded-xl border border-slate-200 border-l-4 bg-white px-5 py-4 {hatMahnungen
			? 'border-l-rose-500'
			: 'border-l-emerald-400'}"
	>
		<div class="flex flex-col lg:flex-row lg:items-center gap-5 lg:gap-8">
			<!-- Links: Zahl + Titel + CTA -->
			<button
				type="button"
				onclick={() => (uiStore.activeTab = 'mahnwesen')}
				class="group flex items-center gap-4 text-left shrink-0 lg:w-80 rounded-lg -m-1 p-1 transition-colors {hatMahnungen
					? 'hover:bg-rose-50/50'
					: 'hover:bg-emerald-50/50'}"
				aria-label="Zum Mahnwesen — alle überfälligen Ausleihen"
			>
				<div
					class="font-extrabold text-5xl tabular-nums leading-none shrink-0 {hatMahnungen
						? 'text-rose-600'
						: 'text-emerald-600'}"
				>
					{summary.total_overdue}
				</div>
				<div class="min-w-0">
					<h3
						class="font-bold text-base leading-tight {hatMahnungen
							? 'text-rose-700'
							: 'text-emerald-700'}"
					>
						{hatMahnungen ? 'Dringend: Mahnungen' : 'Mahnungen'}
					</h3>
					<p class="text-sm text-gray-600">Überfällige Ausleihen gesamt</p>
					<span
						class="mt-1 inline-flex items-center gap-1 text-[11px] font-bold px-2.5 py-1 rounded-full transition-colors {hatMahnungen
							? 'text-rose-600 bg-rose-50 group-hover:bg-rose-100'
							: 'text-emerald-600 bg-emerald-50 group-hover:bg-emerald-100'}"
					>
						Zum Mahnwesen
						<svg
							class="w-3 h-3 transition-transform group-hover:translate-x-0.5"
							fill="none"
							stroke="currentColor"
							viewBox="0 0 24 24"
							aria-hidden="true"
							><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M9 5l7 7-7 7" /></svg
						>
					</span>
				</div>
			</button>

			<!-- Vertikaler Trenner (nur breit) -->
			<div class="hidden lg:block w-px self-stretch bg-slate-100"></div>

			<!-- Rechts: anonyme Verteilung nach Überfälligkeitsdauer.
			     BEWUSST ohne Schülernamen/Titel (Datenminimierung, Art. 5 DSGVO). -->
			<div class="flex-1 min-w-0">
				{#if hatMahnungen}
					<div class="flex items-baseline justify-between mb-2.5">
						<h4 class="text-xs font-semibold uppercase tracking-wider text-slate-400 font-sans">
							Verteilung nach Überfälligkeit
						</h4>
						<span class="text-xs font-semibold text-slate-400">
							längste: <span class="text-rose-600 font-bold">{summary.max_tage_overdue} Tage</span>
						</span>
					</div>
					<div class="grid grid-cols-2 sm:grid-cols-4 gap-x-5 gap-y-3">
						{#each buckets as bucket, _i (_i)}
							{@const kritisch = bucket.label === 'über 60 Tage' && bucket.count > 0}
							<div>
								<div class="flex items-baseline justify-between gap-2 mb-1">
									<span
										class="text-xs font-semibold truncate {kritisch
											? 'text-rose-700'
											: 'text-slate-600'}">{bucket.label}</span
									>
									<span
										class="text-sm font-bold tabular-nums {kritisch
											? 'text-rose-600'
											: 'text-slate-800'}">{bucket.count}</span
									>
								</div>
								<!-- Proportionaler Balken, am grössten Bucket skaliert -->
								<div class="h-1.5 w-full rounded-full bg-slate-100 overflow-hidden">
									<div
										class="h-full rounded-full {kritisch ? 'bg-rose-500' : 'bg-slate-400'}"
										style="width: {(bucket.count / maxBucket) * 100}%"
									></div>
								</div>
							</div>
						{/each}
					</div>
				{:else}
					<div class="flex items-center gap-2 text-emerald-700 font-medium text-sm">
						<svg
							class="w-5 h-5 shrink-0"
							fill="none"
							stroke="currentColor"
							viewBox="0 0 24 24"
							aria-hidden="true"
							><path
								stroke-linecap="round"
								stroke-linejoin="round"
								stroke-width="2"
								d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"
							/></svg
						>
						Keine überfälligen Bücher. Alles im Lot!
					</div>
				{/if}
			</div>
		</div>
	</div>
{/if}
