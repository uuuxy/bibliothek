<script>
	import { apiFetch, apiClient } from './apiFetch.js';

	let query = $state('');
	/** @type {any[]} */
	let results = $state.raw([]);
	let loading = $state(false);
	let searched = $state(false);
	/** @type {ReturnType<typeof setTimeout> | undefined} */
	let debounce;

	async function search() {
		const q = query.trim();
		if (!q) {
			results = [];
			searched = false;
			return;
		}
		loading = true;
		searched = true;
		try {
			const url = `/api/public/opac/suche?q=${encodeURIComponent(q)}`;
			const res = await fetch(url);
			if (res.ok) results = await res.json();
			else results = [];
		} catch {
			results = [];
		} finally {
			loading = false;
		}
	}

	function onInput() {
		clearTimeout(debounce);
		debounce = setTimeout(search, 400);
	}
</script>

<div class="min-h-screen bg-slate-50 flex flex-col">
	<!-- Header -->
	<header
		class="bg-white border-b border-slate-200 px-6 py-4 flex items-center justify-between shadow-xs"
	>
		<div class="flex items-center gap-3">
			<span class="text-2xl">📚</span>
			<div>
				<h1 class="text-lg font-bold text-slate-800 leading-tight">Schulbibliothek</h1>
				<p class="text-xs text-slate-400">Öffentlicher Medienkatalog</p>
			</div>
		</div>
		<div
			class="text-xs text-emerald-600 font-semibold flex items-center gap-1.5 bg-emerald-50 px-3 py-1.5 rounded-full border border-emerald-100"
		>
			🛡️ DSGVO-konform · Keine Ausleihdaten sichtbar
		</div>
	</header>

	<!-- Search bar -->
	<div class="w-full max-w-4xl mx-auto px-6 pt-10 pb-6 space-y-4">
		<div class="relative">
			<svg
				xmlns="http://www.w3.org/2000/svg"
				class="h-5 w-5 absolute left-4 top-1/2 -translate-y-1/2 text-slate-400 pointer-events-none"
				fill="none"
				viewBox="0 0 24 24"
				stroke="currentColor"
			>
				<path
					stroke-linecap="round"
					stroke-linejoin="round"
					stroke-width="2"
					d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"
				/>
			</svg>
			<input
				type="search"
				bind:value={query}
				oninput={onInput}
				placeholder="Titel, Autor oder ISBN eingeben …"
				class="w-full pl-12 pr-12 py-4 text-lg border border-slate-200 rounded-2xl bg-white shadow-sm focus:ring-2 focus:ring-slate-300 outline-none transition-shadow"
				autofocus
			/>
			{#if loading}
				<div
					class="absolute right-4 top-1/2 -translate-y-1/2 w-5 h-5 border-2 border-slate-300 border-t-slate-600 rounded-full animate-spin pointer-events-none"
				></div>
			{/if}
		</div>
	</div>

	<!-- Results / empty states -->
	<div class="flex-1 w-full max-w-4xl mx-auto px-6 pb-10">
		{#if results.length > 0}
			<p class="text-xs text-slate-400 font-medium mb-4">{results.length} Treffer</p>
			<div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
				{#each results as book (book.id)}
					<div
						class="bg-white rounded-2xl border border-slate-100 shadow-sm overflow-hidden hover:shadow-md transition-shadow flex flex-col"
					>
						<!-- Cover area -->
						<div
							class="h-52 bg-linear-to-br from-slate-100 to-slate-200 flex items-center justify-center relative overflow-hidden"
						>
							{#if book.cover_url}
								<img src={book.cover_url} alt="Buchcover" class="h-full w-full object-cover" />
							{:else}
								<span class="text-5xl font-extrabold text-slate-300 select-none">
									{book.titel.charAt(0).toUpperCase()}
								</span>
							{/if}
							<!-- Availability badge overlay -->
							<div class="absolute top-2 right-2">
								{#if book.verfuegbar > 0}
									<span
										class="px-2 py-1 rounded-lg bg-emerald-500 text-white text-xs font-bold shadow-sm"
									>
										✓ Verfügbar
									</span>
								{:else}
									<span
										class="px-2 py-1 rounded-lg bg-rose-500 text-white text-xs font-bold shadow-sm"
									>
										Ausgeliehen
									</span>
								{/if}
							</div>
						</div>
						<!-- Metadata -->
						<div class="p-4 flex-1 flex flex-col">
							<h3 class="font-bold text-slate-800 leading-snug mb-1 line-clamp-2">{book.titel}</h3>
							{#if book.autor}
								<p class="text-xs text-slate-500 mb-2">{book.autor}</p>
							{/if}
							<div class="mt-auto flex items-center justify-between pt-2">
								<span class="text-xs text-slate-400"
									>{book.verfuegbar} / {book.gesamt} verfügbar</span
								>
							</div>
						</div>
					</div>
				{/each}
			</div>
		{:else if searched && !loading}
			<div class="text-center py-20 text-slate-400">
				<span class="text-5xl mb-4 block select-none">📭</span>
				<p class="text-base font-medium">Keine Treffer für „{query}"</p>
				<p class="text-sm mt-1">Versuche es mit einem anderen Titel oder Autor.</p>
			</div>
		{:else if !searched}
			<div class="text-center py-20 text-slate-300 select-none">
				<span class="text-6xl mb-5 block">📚</span>
				<p class="text-xl font-semibold text-slate-400">Suche nach einem Buch</p>
				<p class="text-sm text-slate-300 mt-1">Titel, Autor oder ISBN eingeben</p>
			</div>
		{/if}
	</div>
</div>
