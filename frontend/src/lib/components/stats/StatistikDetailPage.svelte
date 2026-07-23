<!-- @component StatistikDetailPage — Vollwertige, deep-linkbare Detailseite für die
     Statistik-Listen (Renner / Ladenhüter). Ersetzt das frühere Slide-in-Panel: eigene
     URL (/statistiken/renner|ladenhueter), funktionierender Zurück-Button, refresh-fest.
     Lädt die volle Liste selbst (limit=100); alle Filter laufen rein clientseitig. -->
<script>
	import { apiFetch } from '../../apiFetch.js';
	import { uiStore } from '../../stores/uiStore.svelte.js';

	/** @typedef {{ id?: string, titel: string, autor: string, isbn?: string, cover_url?: string, fachbereich?: string, systematik?: string, erscheinungsjahr?: number, count?: number, letzte_aus?: string }} StatRow */

	const kind = $derived(uiStore.statsDetailKind);
	const title = $derived(kind === 'renner' ? 'Beliebteste Titel (Die Renner)' : 'Ladenhüter');
	const hint = $derived(
		kind === 'renner'
			? 'Nach Anzahl der Ausleihen im gewählten Zeitraum.'
			: 'Seit über 2 Jahren nicht ausgeliehen — oder noch nie.'
	);

	/** @type {StatRow[]} */
	let items = $state([]);
	let loading = $state(true);

	// Refresh-fest: die Seite lädt ihre Daten selbst (kein Übergabe-Prop vom Dashboard).
	async function fetchListe() {
		loading = true;
		try {
			const res = await apiFetch('/api/statistiken?limit=100');
			if (!res.ok) throw new Error('Fehler beim Laden');
			const data = await res.json();
			items = (kind === 'renner' ? data.popular_titles : data.shelf_warmers) ?? [];
		} catch (err) {
			console.error('Statistik-Detail laden fehlgeschlagen:', err);
			items = [];
		} finally {
			loading = false;
		}
	}

	$effect(() => {
		// eslint-disable-next-line @typescript-eslint/no-unused-expressions
		kind; // bei Wechsel Renner↔Ladenhüter neu laden
		fetchListe();
	});

	// Lokale Filter — reine Client-Reaktivität, keine API-Calls
	let filterFach = $state('alle');
	let filterSystematik = $state('alle');
	let suchbegriff = $state('');

	const fachOptionen = $derived([...new Set(items.map((i) => i.fachbereich).filter(Boolean))].sort());
	const systematikOptionen = $derived(
		[...new Set(items.map((i) => i.systematik).filter(Boolean))].sort()
	);

	const gefiltert = $derived(
		items.filter((i) => {
			if (filterFach !== 'alle' && i.fachbereich !== filterFach) return false;
			if (filterSystematik !== 'alle' && i.systematik !== filterSystematik) return false;
			if (suchbegriff) {
				const q = suchbegriff.toLowerCase();
				if (!`${i.titel} ${i.autor}`.toLowerCase().includes(q)) return false;
			}
			return true;
		})
	);
</script>

<div class="w-full max-w-4xl mx-auto px-6 pt-6 pb-10 text-slate-800">
	<!-- Zurück zur Statistik-Übersicht -->
	<button
		onclick={() => (uiStore.activeTab = 'stats')}
		class="inline-flex items-center gap-1.5 text-sm font-semibold text-slate-500 hover:text-blue-600 transition-colors cursor-pointer mb-5"
	>
		<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"
			><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M15 19l-7-7 7-7" /></svg
		>
		Statistik
	</button>

	<header class="border-b border-slate-100 pb-5 mb-5 space-y-4">
		<div>
			<h1 class="text-2xl font-extrabold text-slate-900 tracking-tight">{title}</h1>
			<p class="text-xs text-slate-500 mt-1">{hint}</p>
		</div>

		<!-- Lokale Filterzeile -->
		<div class="flex flex-wrap items-center gap-3">
			<input
				type="search"
				bind:value={suchbegriff}
				placeholder="Titel oder Autor…"
				class="flex-1 min-w-40 px-3 py-2 rounded-lg border border-slate-200 bg-white text-sm"
			/>
			<select
				bind:value={filterFach}
				class="px-3 py-2 rounded-lg border border-slate-200 bg-white text-sm font-semibold text-slate-700"
				aria-label="Nach Fachbereich filtern"
			>
				<option value="alle">Fachbereich: alle</option>
				{#each fachOptionen as f, _i (_i)}<option value={f}>{f}</option>{/each}
			</select>
			<select
				bind:value={filterSystematik}
				class="px-3 py-2 rounded-lg border border-slate-200 bg-white text-sm font-semibold text-slate-700"
				aria-label="Nach Systematik filtern"
			>
				<option value="alle">Systematik: alle</option>
				{#each systematikOptionen as s, _i (_i)}<option value={s}>{s}</option>{/each}
			</select>
		</div>
		{#if !loading}
			<p class="text-[11px] text-slate-400 font-medium">
				{gefiltert.length} von {items.length} Einträgen
			</p>
		{/if}
	</header>

	{#if loading}
		<div class="py-20 flex justify-center">
			<div class="w-8 h-8 border-2 border-t-blue-500 border-blue-500/20 rounded-full animate-spin"></div>
		</div>
	{:else if gefiltert.length === 0}
		<div class="py-20 text-center text-slate-400 text-sm">
			{items.length === 0 ? 'Noch keine Daten vorhanden.' : 'Keine Einträge für diese Filter.'}
		</div>
	{:else}
		<ul class="divide-y divide-slate-100 border-t border-slate-100">
			<!-- Key ist die Titel-ID: Titel+ISBN taugt nicht, zwei Titel dürfen gleich
			     heissen und beide ohne ISBN sein (doppelter Key = Absturz der Ansicht). -->
			{#each gefiltert as row (row.id)}
				<li class="py-3.5 flex items-center gap-4">
					{#if row.cover_url}
						<img
							src={row.cover_url}
							alt=""
							class="w-9 aspect-3/4 object-cover rounded-sm border border-slate-100 shrink-0"
						/>
					{:else}
						<div
							class="w-9 aspect-3/4 bg-slate-50 border border-slate-100 rounded-sm flex items-center justify-center text-slate-300 text-xs shrink-0"
						>
							📖
						</div>
					{/if}
					<div class="min-w-0 flex-1">
						<p class="text-sm font-bold text-slate-800 truncate" title={row.titel}>{row.titel}</p>
						<p class="text-xs text-slate-450 truncate">
							{row.autor || '—'}
							{#if row.fachbereich}· {row.fachbereich}{/if}
							{#if row.systematik}· <span class="font-mono">{row.systematik}</span>{/if}
							{#if row.erscheinungsjahr}· {row.erscheinungsjahr}{/if}
						</p>
					</div>
					<div class="shrink-0 text-right">
						{#if kind === 'renner'}
							<span class="text-sm font-black text-slate-900 tabular-nums">{row.count}×</span>
							<span class="block text-[10px] text-slate-400 font-semibold uppercase">geliehen</span>
						{:else}
							<span class="text-xs font-bold text-amber-600">{row.letzte_aus}</span>
						{/if}
					</div>
				</li>
			{/each}
		</ul>
	{/if}
</div>
