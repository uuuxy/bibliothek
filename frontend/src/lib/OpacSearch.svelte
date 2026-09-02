<script>
	import { BookOpen, Search, ShieldCheck } from '@lucide/svelte';
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

<div class="min-h-screen bg-surface flex flex-col relative overflow-x-hidden">
	<LogoRelief />

	<!-- Header -->
	<header
		class="bg-surface-container-lowest border-b border-outline-variant px-6 py-4 flex items-center justify-between relative z-10"
	>
		<div class="flex items-center gap-3">
			<BookOpen class="h-5 w-5" aria-hidden="true" />
			<div>
				<h1 class="text-lg font-semibold text-on-surface leading-tight">Schulbibliothek</h1>
				<p class="text-xs text-on-surface-variant">Öffentlicher Medienkatalog</p>
			</div>
		</div>
		<div
			class="text-xs text-on-surface-variant font-medium flex items-center gap-1.5 bg-surface-container-high px-3 py-1.5 rounded-full"
		>
			<ShieldCheck class="h-3.5 w-3.5" aria-hidden="true" />
			DSGVO-konform · Keine Ausleihdaten sichtbar
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
			<p class="text-xs text-on-surface-variant font-medium mb-4">{results.length} Treffer</p>
			<!-- Dieselbe Kachel wie im internen Katalog (02.09.2026): das Cover IST die Kachel,
			     2:3 wie ein Buch, keine Karte, kein Rahmen. Vorher lag das Hochformat-Cover in
			     einer festen Querformat-Fläche und wurde oben und unten abgeschnitten. -->
			<div class="grid grid-cols-[repeat(auto-fill,minmax(12rem,1fr))] gap-x-3 gap-y-4">
				{#each results as book (book.id)}
					<div class="flex flex-col gap-3 p-3">
						<div
							class="relative aspect-2/3 w-full overflow-hidden rounded-lg bg-surface-container-low"
						>
							{#if coverSrc(book.cover_url, book.isbn)}
								<img
									src={coverSrc(book.cover_url, book.isbn)}
									alt={`Cover von ${book.titel}`}
									loading="lazy"
									class="h-full w-full object-cover"
								/>
							{:else}
								<div class="flex h-full w-full items-center justify-center">
									<span class="text-5xl font-extrabold text-on-surface-variant/50 select-none">
										{book.titel.charAt(0).toUpperCase()}
									</span>
								</div>
							{/if}
							<div class="absolute top-2 right-2">
								{#if book.verfuegbar > 0}
									<span class="px-2 py-1 rounded-lg bg-emerald-500 text-white text-xs font-bold">
										✓ Verfügbar
									</span>
								{:else}
									<span class="px-2 py-1 rounded-lg bg-error text-on-error text-xs font-bold">
										Ausgeliehen
									</span>
								{/if}
							</div>
						</div>
						<div class="flex flex-col gap-1 text-sm text-on-surface-variant">
							<h3
								class="line-clamp-2 text-base leading-snug font-semibold wrap-break-word text-on-surface"
								title={book.titel}
							>
								{book.titel}
							</h3>
							{#if book.autor}
								<p class="line-clamp-1">{book.autor}</p>
							{/if}
							<p>{book.verfuegbar} / {book.gesamt} verfügbar</p>
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
			class="shrink-0 w-4 h-4 border-2 border-primary/40 border-t-primary rounded-full animate-spin"
			aria-hidden="true"
		></div>
	{/if}
{/snippet}
