<!-- @component StatistikDetailPanel — Drill-Down-Sidepanel für die Statistik-Listen
     (Renner / Ladenhüter). Flaches Edge-to-Edge-Panel von rechts (kein Modal);
     alle Filter laufen rein clientseitig über $state/$derived — die Daten wurden
     beim Dashboard-Load bereits vollständig mitgeladen (?limit=100). -->
<script>
	import { fly, fade } from 'svelte/transition';

	/** @typedef {{ id?: string, titel: string, autor: string, isbn?: string, cover_url?: string, fachbereich?: string, systematik?: string, erscheinungsjahr?: number, count?: number, letzte_aus?: string }} StatRow */

	/** @type {{ kind: 'renner' | 'ladenhueter', items: StatRow[], onClose: () => void }} */
	let { kind, items, onClose } = $props();

	const title = $derived(kind === 'renner' ? 'Beliebteste Titel (Die Renner)' : 'Ladenhüter');
	const hint = $derived(
		kind === 'renner'
			? 'Nach Anzahl der Ausleihen im gewählten Zeitraum.'
			: 'Seit über 2 Jahren nicht ausgeliehen — oder noch nie.'
	);

	// Lokale Filter — reine Client-Reaktivität, keine API-Calls
	let filterFach = $state('alle');
	let filterSystematik = $state('alle');
	let suchbegriff = $state('');

	const fachOptionen = $derived(
		[...new Set(items.map((i) => i.fachbereich).filter(Boolean))].sort()
	);
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

	function handleKeydown(/** @type {KeyboardEvent} */ e) {
		if (e.key === 'Escape') onClose();
	}
</script>

<svelte:window onkeydown={handleKeydown} />

<!-- Backdrop: dezent, Klick schließt -->
<div
	class="fixed inset-0 z-40 bg-slate-900/15"
	transition:fade={{ duration: 150 }}
	onclick={onClose}
	aria-hidden="true"
></div>

<!-- Edge-to-Edge-Sidepanel: volle Höhe, flache Kante statt schwebender Kachel -->
<div
	class="fixed inset-y-0 right-0 z-50 w-full max-w-2xl bg-white border-l border-slate-200 flex flex-col"
	transition:fly={{ x: 480, duration: 220, opacity: 1 }}
	role="dialog"
	aria-label={title}
>
	<header class="px-8 pt-8 pb-5 border-b border-slate-100 shrink-0 space-y-4">
		<div class="flex items-start justify-between gap-4">
			<div>
				<h2 class="text-2xl font-extrabold text-slate-900 tracking-tight">{title}</h2>
				<p class="text-xs text-slate-500 mt-1">{hint}</p>
			</div>
			<button
				onclick={onClose}
				class="p-2 -m-2 text-slate-400 hover:text-slate-700 transition-colors cursor-pointer"
				aria-label="Detailansicht schließen"
			>
				<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"
					><path
						stroke-linecap="round"
						stroke-linejoin="round"
						stroke-width="2"
						d="M6 18L18 6M6 6l12 12"
					/></svg
				>
			</button>
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
		<p class="text-[11px] text-slate-400 font-medium">
			{gefiltert.length} von {items.length} Einträgen
		</p>
	</header>

	<div class="flex-1 overflow-y-auto">
		{#if gefiltert.length === 0}
			<div class="py-20 text-center text-slate-400 text-sm">Keine Einträge für diese Filter.</div>
		{:else}
			<ul class="divide-y divide-slate-100">
				{#each gefiltert as row (row.id ?? row.titel + (row.isbn ?? ''))}
					<li class="px-8 py-3.5 flex items-center gap-4">
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
								<span class="block text-[10px] text-slate-400 font-semibold uppercase"
									>geliehen</span
								>
							{:else}
								<span class="text-xs font-bold text-amber-600">{row.letzte_aus}</span>
							{/if}
						</div>
					</li>
				{/each}
			</ul>
		{/if}
	</div>
</div>
