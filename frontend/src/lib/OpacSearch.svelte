<script>
	import { BookOpen, Search } from '@lucide/svelte';
	import LogoRelief from './components/ui/LogoRelief.svelte';
	import Suchpille from './components/ui/Suchpille.svelte';
	import SuchZustand from './components/ui/SuchZustand.svelte';
	import { coverSrc } from './utils/coverSrc.js';

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

<div class="min-h-screen bg-slate-50 flex flex-col relative overflow-x-hidden">
	<LogoRelief />

	<!-- Header -->
	<header
		class="bg-white border-b border-slate-200 px-6 py-4 flex items-center justify-between shadow-xs relative z-10"
	>
		<div class="flex items-center gap-3">
			<BookOpen class="h-5 w-5" aria-hidden="true" />
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
	<div class="w-full max-w-4xl mx-auto px-6 pt-10 pb-6 space-y-4 relative z-10">
		<!-- Der OPAC steht als Katalog-Terminal im Raum, und dieses Feld ist der einzige
		     Zweck der Seite — deshalb Fokus beim Betreten. Ohne ihn tippt der erste
		     Anschlag ins Leere, und beim Barcode-Scanner geht der Scan verloren, ohne dass
		     jemand einen Fehler sieht. -->
		<Suchpille
			id="opac-suchfeld"
			bind:wert={query}
			oninput={onInput}
			platzhalter="Titel, Autor oder ISBN eingeben …"
			etikett="Im Medienkatalog suchen"
			autofokus
			{nachlaufend}
		/>
	</div>

	<!-- Results / empty states -->
	<div class="flex-1 w-full max-w-4xl mx-auto px-6 pb-10 relative z-10">
		{#if results.length > 0}
			<p class="text-xs text-slate-400 font-medium mb-4">{results.length} Treffer</p>
			<div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
				{#each results as book (book.id)}
					<div
						class="bg-white rounded-xl border border-slate-100 shadow-sm overflow-hidden hover:shadow-md transition-shadow flex flex-col"
					>
						<!-- Cover area -->
						<div
							class="h-52 bg-linear-to-br from-slate-100 to-slate-200 flex items-center justify-center relative overflow-hidden"
						>
							{#if coverSrc(book.cover_url, book.isbn)}
								<img
									src={coverSrc(book.cover_url, book.isbn)}
									alt="Buchcover"
									class="h-full w-full object-cover"
								/>
							{:else}
								<span class="text-5xl font-extrabold text-slate-400 select-none">
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
										class="px-2 py-1 rounded-lg bg-rose-600 text-white text-xs font-bold shadow-sm"
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
			<SuchZustand
				symbol={Search}
				titel="Keine Bücher gefunden"
				hinweis="Versuche es mit einem anderen Titel oder Autor."
			/>
		{:else if !searched}
			<SuchZustand
				symbol={BookOpen}
				titel="Suche nach einem Buch"
				hinweis="Titel, Autor oder ISBN eingeben"
			/>
		{/if}
	</div>
</div>

<!-- Der Ladepunkt sitzt IN der Pille — dieselbe Stelle wie im Kollegiums-Portal. -->
{#snippet nachlaufend()}
	{#if loading}
		<div
			class="shrink-0 w-4 h-4 border-2 border-blue-500/40 border-t-blue-500 rounded-full animate-spin"
			aria-hidden="true"
		></div>
	{/if}
{/snippet}
