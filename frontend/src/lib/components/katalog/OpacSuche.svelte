<script>
	import { BookOpen, Search, ShieldCheck } from '@lucide/svelte';
	import Ladekreis from '../ui/Ladekreis.svelte';
	import LadeFehler from '../ui/LadeFehler.svelte';
	import LogoRelief from '../ui/LogoRelief.svelte';
	import Suchpille from '../ui/Suchpille.svelte';
	import SuchZustand from '../ui/SuchZustand.svelte';
	import BuchCover from '../ui/BuchCover.svelte';
	import NeueFassung from '../layout/NeueFassung.svelte';
	import { apiFetch, leseApiFehler } from '../../apiFetch.js';
	import { fehlertext } from '../../utils/fehlertext.js';

	let suche = $state('');
	/** @type {any[]} */
	let ergebnisse = $state.raw([]);
	let laedt = $state(false);
	let searched = $state(false);
	// Der Grund, wenn die Suche scheiterte: „Keine Bücher gefunden" hieße, der Katalog habe das Buch
	// nicht.
	let ladefehler = $state('');
	/** Alle Treffer der Suche — der Katalog zeigt höchstens 50 (Kopf X-Treffer-Gesamt, api/portal/opac.go). */
	let gesamt = $state(0);
	/** @type {ReturnType<typeof setTimeout> | undefined} */
	let debounce;

	async function starteSuche() {
		const q = suche.trim();
		ladefehler = '';
		if (!q) {
			ergebnisse = [];
			searched = false;
			return;
		}
		laedt = true;
		searched = true;
		try {
			// Ohne Anmeldung und Live-Leitung erfährt der Katalog ein Update nur aus dieser Antwort;
			// apiFetch merkt sich den Stand des Servers.
			const res = await apiFetch(`/api/public/opac/suche?q=${encodeURIComponent(q)}`);
			if (!res.ok) throw new Error(await leseApiFehler(res));
			ergebnisse = await res.json();
			gesamt = Number(res.headers.get('X-Treffer-Gesamt')) || ergebnisse.length;
		} catch (e) {
			ergebnisse = [];
			ladefehler = fehlertext(e);
		} finally {
			laedt = false;
		}
	}

	function onInput() {
		clearTimeout(debounce);
		debounce = setTimeout(starteSuche, 400);
	}
</script>

<main class="min-h-screen bg-surface flex flex-col relative overflow-x-hidden">
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

	<!-- Search bar: mittig wie die ganze Seite; die Pille selbst ist höchstens 720 px breit. -->
	<div class="w-full max-w-4xl mx-auto px-6 pt-10 pb-6 flex flex-col items-center relative z-10">
		<!-- Der OPAC steht als Katalog-Terminal im Raum, und dieses Feld ist der einzige
		     Zweck der Seite — deshalb Fokus beim Betreten. Ohne ihn tippt der erste
		     Anschlag ins Leere, und beim Barcode-Scanner geht der Scan verloren, ohne dass
		     jemand einen Fehler sieht. -->
		<!-- Der Hinweis nach einem Update steht über dem Inhalt wie in der Anwendung; zwischen
		     Suchfeld und Treffern trennte er die beiden. -->
		<NeueFassung />
		<Suchpille
			id="opac-suchfeld"
			bind:wert={suche}
			oninput={onInput}
			platzhalter="Titel, Autor oder ISBN eingeben …"
			etikett="Im Medienkatalog suchen"
			autofokus
			{nachlaufend}
		/>
	</div>

	<!-- Results / empty states -->
	<div class="flex-1 w-full max-w-4xl mx-auto px-6 pb-10 relative z-10">
		{#if ladefehler}
			<LadeFehler titel="Katalog nicht geladen" text={ladefehler} onerneut={starteSuche} />
		{:else if ergebnisse.length > 0}
			<!-- Ist die Antwort gekappt, nennt die Zeile beide Zahlen: Sonst stünde die Kappung
			     des Servers da wie die Zahl der Bücher. Derselbe Satz wie in „Mein Portal". -->
			<p class="text-xs text-on-surface-variant font-medium mb-4">
				{#if gesamt > ergebnisse.length}
					Gezeigt werden {ergebnisse.length} von {gesamt} Treffern — ein Suchwort grenzt ein.
				{:else}
					{ergebnisse.length} Treffer
				{/if}
			</p>
			<!-- Dieselbe Kachel wie im internen Katalog: Das Cover ist die Kachel, 2:3 wie ein
			     Buch, keine Karte, kein Rahmen; so wird es nicht beschnitten. -->
			<div class="grid grid-cols-[repeat(auto-fill,minmax(12rem,1fr))] gap-x-3 gap-y-4">
				{#each ergebnisse as book (book.id)}
					<div class="flex flex-col gap-3 p-3">
						<div class="relative">
							<BuchCover
								coverUrl={book.cover_url}
								isbn={book.isbn}
								titel={book.titel}
								groesse="kachel"
								nurGespeichert
							/>
							<div class="absolute top-2 right-2">
								{#if book.verfuegbar > 0}
									<span class="px-2 py-1 rounded-lg bg-success text-on-success text-xs font-bold">
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
		{:else if searched && !laedt}
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
</main>

<!-- Der Ladepunkt sitzt IN der Pille — dieselbe Stelle wie im Kollegiums-Portal. -->
{#snippet nachlaufend()}
	{#if laedt}
		<Ladekreis size="sm" />
	{/if}
{/snippet}
