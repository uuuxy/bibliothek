<script>
	import { apiFetch } from './apiFetch.js';
	import Ladekreis from './components/ui/Ladekreis.svelte';
	import { uiStore } from './stores/uiStore.svelte.js';
	import { ChevronRight, CircleCheck } from '@lucide/svelte';

	/** aktuellVerliehen: laufende Ausleihen — Basis für die neutrale Überfälligkeitsquote.
	 * @type {{ aktuellVerliehen?: number }} */
	let { aktuellVerliehen = 0 } = $props();

	/** @type {any} */
	let summary = $state(null);
	let loading = $state(true);
	/** Abruf gescheitert — eine leere Kachel sähe aus wie „nichts überfällig". */
	let fehler = $state(false);

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
			if (res.ok) {
				summary = await res.json();
				fehler = false;
			} else {
				// Sweep „verschluckte Fehlantwort" (06.09.2026): Ohne summary rendert unten
				// GAR NICHTS — eine leere Kachel in der Bento-Reihe, die wie „nichts
				// überfällig" aussieht. Eine Zahl, die fehlt, muss sich als fehlend zeigen.
				fehler = true;
			}
		} catch (err) {
			fehler = true;
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
		<Ladekreis size="md" />
	</div>
{:else if fehler}
	<p class="py-8 text-center text-sm font-semibold text-error" role="alert">
		Überfällige Ausleihen konnten nicht geladen werden.
	</p>
{:else if summary}
	<!-- NEUTRAL (kein Rot-Alarm): Überfälligkeit als reine Statistik. Zahl und Quote in den Textrollen,
	     Verteilung in Grau (nur „>60 Tage" in der Warnrolle). Die operative
	     Bearbeitung liegt im Mahnwesen — hier nur ein zurückhaltender Textlink dorthin.
	     Layout: vertikal gestapelt, weil das Widget in der schmalen 1/3-Spalte der
	     Bento-Reihe sitzt — nebeneinander bräche es dort um. -->
	<div class="h-full flex flex-col">
		<h3 class="text-base font-medium text-on-surface-variant">Überfällige Ausleihen</h3>
		<div class="flex items-baseline gap-2 mt-2">
			<span class="text-4xl font-light text-on-surface tabular-nums leading-none"
				>{summary.total_overdue}</span
			>
			{#if quote !== null && hatMahnungen}
				<span class="text-xs text-on-surface-variant">≈ {quote} % der laufenden Ausleihen</span>
			{/if}
		</div>

		{#if hatMahnungen}
			<div class="flex items-baseline justify-between gap-2 mt-5 mb-2.5">
				<h4 class="text-xs font-medium text-on-surface-variant">Verteilung nach Dauer</h4>
				<span class="shrink-0 text-xs text-on-surface-variant"
					>längste: {summary.max_tage_overdue} Tage</span
				>
			</div>
			<!-- 2 Spalten fix (nicht viewport-abhängig): die Card ist immer schmal. -->
			<div class="grid grid-cols-2 gap-x-4 gap-y-2.5 min-h-0 overflow-y-auto">
				{#each buckets as bucket, _i (_i)}
					{@const alt = bucket.label === 'über 60 Tage' && bucket.count > 0}
					<div>
						<div class="flex items-baseline justify-between gap-2 mb-1.5">
							<span class="text-xs font-medium text-on-surface-variant truncate"
								>{bucket.label}</span
							>
							<span class="text-sm font-semibold tabular-nums text-on-surface">{bucket.count}</span>
						</div>
						<!-- Grau; nur die längst-überfällige Gruppe steht in der Warnrolle. -->
						<div class="h-1.5 w-full rounded-full bg-surface-container-highest overflow-hidden">
							<div
								class="h-full rounded-full {alt ? 'bg-warning' : 'bg-on-surface-variant'}"
								style="width: {(bucket.count / maxBucket) * 100}%"
							></div>
						</div>
					</div>
				{/each}
			</div>
		{:else}
			<div class="flex-1 flex items-center gap-2 text-on-surface-variant text-sm">
				<CircleCheck class="w-5 h-5 shrink-0 text-success" aria-hidden="true" />
				Keine überfälligen Ausleihen.
			</div>
		{/if}

		<button
			type="button"
			onclick={() => (uiStore.activeTab = 'mahnwesen')}
			class="mt-auto pt-3 border-t border-outline-variant inline-flex items-center justify-between gap-1 text-sm font-semibold text-primary cursor-pointer"
			aria-label="Zum Mahnwesen — überfällige Ausleihen bearbeiten"
		>
			Im Mahnwesen bearbeiten
			<ChevronRight class="w-3.5 h-3.5" aria-hidden="true" />
		</button>
	</div>
{/if}
