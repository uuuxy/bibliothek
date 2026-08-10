<script>
	import { apiFetch } from './apiFetch.js';
	import { coverSrc } from './utils/coverSrc.js';
	import Button from './components/ui/Button.svelte';
	import PageShell from './components/layout/PageShell.svelte';
	import Suchpille from './components/ui/Suchpille.svelte';
	import SuchZustand from './components/ui/SuchZustand.svelte';
	import { BookOpen, Search } from '@lucide/svelte';
	/** @type {{ user: any }} */
	let { user } = $props();

	let searchQuery = $state('');
	let searchResults = $state.raw(/** @type {any[]} */ ([]));
	let isSearching = $state(false);

	// Per-book form state
	let reservierungForms = $state(
		/** @type {Record<string, { open: boolean, klasse: string, anzahl: number, notiz: string, loading: boolean, success: string|null, error: string|null }>} */ ({})
	);

	let searchTimeout = /** @type {any} */ (null);

	$effect(() => {
		const q = searchQuery;
		clearTimeout(searchTimeout);
		if (q.trim().length < 2) {
			searchResults = [];
			return () => clearTimeout(searchTimeout);
		}
		searchTimeout = setTimeout(async () => {
			isSearching = true;
			try {
				// Bewusst der OPAC und nicht /api/search: Nur der OPAC rechnet die
				// Verfügbarkeit aus. /api/search liefert `BookTitle` — dort gibt es KEIN
				// Bestandsfeld, weshalb das Abzeichen unten still übersprungen wurde und
				// Lehrkräfte nie erfahren haben, ob ein Klassensatz überhaupt frei ist.
				//
				// Der OPAC passt auch fachlich: ausdrücklich nur Titel, Autor und Verfügbarkeit,
				// keine Ausleih- oder Personendaten — genau das, was eine Lehrkraft sehen darf.
				const res = await apiFetch(`/api/public/opac/suche?q=${encodeURIComponent(q)}`);
				if (res.ok) {
					const data = await res.json();
					searchResults = Array.isArray(data) ? data : (data.books ?? []);
				}
			} catch {
				/* ignore */
			} finally {
				isSearching = false;
			}
		}, 300);
		return () => clearTimeout(searchTimeout);
	});

	/**
	 * Legt das Formular-Objekt für einen Titel an, falls es fehlt.
	 * Darf NUR aus Event-Handlern/asynchronem Code aufgerufen werden —
	 * eine Zuweisung an $state während des Template-Renderns wirft in
	 * Svelte 5 `state_unsafe_mutation` und bricht das Rendern der
	 * Suchtreffer komplett ab (so konnten Lehrkräfte real nicht suchen).
	 * @param {string} titelId
	 */
	function ensureForm(titelId) {
		if (!reservierungForms[titelId]) {
			reservierungForms[titelId] = {
				open: false,
				klasse: user?.klasse ?? '',
				anzahl: 1,
				notiz: '',
				loading: false,
				success: null,
				error: null
			};
		}
		return reservierungForms[titelId];
	}

	/**
	 * Reine Lese-Sicht fürs Template — mutiert nie.
	 * @param {string} titelId
	 */
	function getForm(titelId) {
		return (
			reservierungForms[titelId] ?? {
				open: false,
				klasse: user?.klasse ?? '',
				anzahl: 1,
				notiz: '',
				loading: false,
				success: null,
				error: null
			}
		);
	}

	/**
	 * @param {string} titelId
	 */
	function toggleForm(titelId) {
		const f = ensureForm(titelId);
		f.open = !f.open;
		f.success = null;
		f.error = null;
	}

	/**
	 * @param {string} titelId
	 */
	async function submitReservierung(titelId) {
		const f = ensureForm(titelId);
		if (!f.klasse.trim()) {
			f.error = 'Bitte Klasse angeben.';
			return;
		}
		f.loading = true;
		f.error = null;
		f.success = null;
		try {
			const res = await apiFetch('/api/reservierungen/klassensatz', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({
					titel_id: titelId,
					klasse: f.klasse,
					anzahl: f.anzahl,
					notiz: f.notiz
				})
			});
			if (res.ok) {
				f.success = 'Reservierungsanfrage wurde gesendet!';
				f.open = false;
			} else {
				const txt = await res.text();
				f.error = txt || 'Fehler beim Senden.';
			}
		} catch (e) {
			f.error = String(e);
		} finally {
			f.loading = false;
		}
	}
</script>

<PageShell>
	<Suchpille
		id="portal-suchfeld"
		bind:wert={searchQuery}
		platzhalter="Titel, Autor oder ISBN eingeben …"
		etikett="Bücher für einen Klassensatz suchen"
		autofokus
		{nachlaufend}
	/>

	<!-- Results -->
	{#if searchResults.length > 0}
		<div class="space-y-4">
			{#each searchResults as book (book.id ?? book.titel_id)}
				{@const titelId = book.id ?? book.titel_id}
				{@const form = getForm(titelId)}
				<div class="w-full">
					<div class="flex gap-4 p-4">
						<!-- Cover -->
						<div
							class="w-16 h-20 rounded-xl bg-slate-100 border border-slate-200 shrink-0 overflow-hidden flex items-center justify-center"
						>
							{#if coverSrc(book.cover_url, book.isbn)}
								<img
									src={coverSrc(book.cover_url, book.isbn)}
									alt="Cover"
									class="w-full h-full object-cover"
									loading="lazy"
								/>
							{:else}
								<BookOpen class="h-7 w-7 text-slate-300" aria-hidden="true" />
							{/if}
						</div>

						<!-- Info -->
						<div class="flex-1 min-w-0">
							<h3 class="font-semibold text-slate-800 text-sm leading-tight truncate">
								{book.titel ?? book.title ?? 'Unbekannter Titel'}
							</h3>
							<p class="text-xs text-slate-500 mt-0.5">{book.autor ?? book.author ?? ''}</p>
							{#if book.isbn}
								<p class="text-label-small text-slate-400 mt-1">ISBN {book.isbn}</p>
							{/if}
							<!-- Für einen Klassensatz zählt beides: wie viele gerade frei sind UND wie
							     viele es überhaupt gibt. „3 verfügbar" allein sagt einer Lehrkraft
							     nicht, ob der Titel für 28 Schüler je reichen kann. -->
							{#if book.verfuegbar != null}
								<p class="text-xs mt-1.5">
									<span
										class="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-label-small font-semibold {book.verfuegbar >
										0
											? 'bg-emerald-50 text-emerald-700'
											: 'bg-rose-50 text-rose-600'}"
									>
										{book.verfuegbar > 0
											? `${book.verfuegbar} von ${book.gesamt} verfügbar`
											: `nicht verfügbar (${book.gesamt} im Bestand)`}
									</span>
								</p>
							{/if}
						</div>

						<!-- Action -->
						<!-- Die Bestätigung ERSETZT den Knopf nicht: Eine Lehrkraft, die denselben
						     Titel für 8a bestellt hat, braucht ihn direkt danach für 8b. Vorher blieb
						     „✓ Gesendet" für immer stehen und der einzige Weg zurück war ein Reload —
						     ausgerechnet toggleForm, das den Zustand aufräumt, war nicht mehr
						     erreichbar. -->
						<div class="shrink-0 flex flex-col items-end justify-between gap-2">
							{#if form.success}
								<span class="text-xs text-emerald-600 font-semibold" title={form.success}
									>✓ Gesendet</span
								>
							{/if}
							<Button
								variant={form.open || form.success ? 'secondary' : 'primary'}
								size="sm"
								onclick={() => toggleForm(titelId)}
							>
								{#if form.open}
									Abbrechen
								{:else if form.success}
									Weitere Klasse reservieren
								{:else}
									Klassensatz reservieren
								{/if}
							</Button>
						</div>
					</div>

					<!-- Inline reservation form -->
					{#if form.open}
						<div class="border-t border-slate-100 bg-slate-50 px-4 py-4">
							<p class="text-xs font-semibold text-slate-600 mb-3">Klassensatz-Reservierung</p>
							<div class="grid grid-cols-2 gap-3">
								<div>
									<label
										for="klasse-{titelId}"
										class="block text-xs font-medium text-slate-500 mb-1">Klasse *</label
									>
									<input
										id="klasse-{titelId}"
										type="text"
										bind:value={form.klasse}
										placeholder="z. B. 8b"
										class="w-full px-3 py-2 rounded-xl border border-slate-200 bg-white text-sm text-slate-800 focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-400"
									/>
								</div>
								<div>
									<label
										for="anzahl-{titelId}"
										class="block text-xs font-medium text-slate-500 mb-1">Anzahl</label
									>
									<input
										id="anzahl-{titelId}"
										type="number"
										bind:value={form.anzahl}
										min="1"
										max="200"
										class="w-full px-3 py-2 rounded-xl border border-slate-200 bg-white text-sm text-slate-800 focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-400"
									/>
								</div>
							</div>
							<div class="mt-3">
								<label for="notiz-{titelId}" class="block text-xs font-medium text-slate-500 mb-1"
									>Notiz (optional)</label
								>
								<textarea
									id="notiz-{titelId}"
									bind:value={form.notiz}
									rows="2"
									placeholder="z. B. Benötigt ab 15. September …"
									class="w-full px-3 py-2 rounded-xl border border-slate-200 bg-white text-sm text-slate-800 focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-400 resize-none"
								></textarea>
							</div>
							{#if form.error}
								<p class="text-xs text-rose-500 mt-2">{form.error}</p>
							{/if}
							<div class="mt-3 flex justify-end">
								<Button onclick={() => submitReservierung(titelId)} disabled={form.loading}>
									{#if form.loading}
										<div
											class="w-3.5 h-3.5 border-2 border-white/40 border-t-white rounded-full animate-spin"
										></div>
									{/if}
									Anfrage senden
								</Button>
							</div>
						</div>
					{/if}
				</div>
			{/each}
		</div>
	{:else if searchQuery.trim().length >= 2 && !isSearching}
		<SuchZustand
			symbol={Search}
			titel="Keine Bücher gefunden"
			hinweis="Versuche es mit einem anderen Titel oder Autor."
		/>
	{:else if searchQuery.trim().length === 0}
		<SuchZustand
			symbol={BookOpen}
			titel="Suche nach einem Buch"
			hinweis="Titel, Autor oder ISBN eingeben"
		/>
	{/if}
</PageShell>

<!-- Der Ladepunkt sitzt IN der Pille, nicht darüber. Vorher lag er absolut positioniert
     bei right-4, während das Feld nur pr-4 Innenabstand hatte — ein langer Suchbegriff
     lief also unter den Punkt. -->
{#snippet nachlaufend()}
	{#if isSearching}
		<div
			class="shrink-0 w-4 h-4 border-2 border-blue-500/40 border-t-blue-500 rounded-full animate-spin"
			aria-hidden="true"
		></div>
	{/if}
{/snippet}
