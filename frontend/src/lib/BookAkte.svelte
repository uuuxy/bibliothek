<script>
	import { showToast } from '../inventur/lib/store.svelte.js';
	import Ladekreis from './components/ui/Ladekreis.svelte';
	import BookBorrowersTab from './BookBorrowersTab.svelte';
	import BookExemplareTab from './BookExemplareTab.svelte';
	import BookHistoryTab from './BookHistoryTab.svelte';
	import BookVormerkungenTab from './BookVormerkungenTab.svelte';
	import BookAkteMeta from './BookAkteMeta.svelte';
	import { useBookAkte } from './useBookAkte.svelte.js';
	import PageShell from './components/layout/PageShell.svelte';
	import { authStore } from './stores/authStore.svelte.js';
	import { hatRecht } from './menu.js';
	import { ChevronLeft, Frown } from '@lucide/svelte';
	import { untrack } from 'svelte';

	/** @type {{ bookId: string | null, onBack: () => void }} */
	let { bookId, onBack } = $props();

	const akte = useBookAkte();

	// Enges Theken-Recht (db/seed.go): Ohne manage_vormerkungen antwortet die API
	// mit 403 — dann den Reiter gar nicht erst anbieten statt leer scheitern lassen.
	const darfVormerken = $derived(hatRecht(authStore.currentUser, 'manage_vormerkungen'));
	// Die Aktionen der Akte hängen am Recht — vorher sah jeder „Titel bearbeiten" und
	// „Gesamten Titel löschen", auch wer beides nicht durfte (Knopf → 403).
	const darfBearbeiten = $derived(hatRecht(authStore.currentUser, 'edit_books'));
	const darfLoeschen = $derived(hatRecht(authStore.currentUser, 'delete_books'));
	// Eine Zahl, die niemand kennt, ist ein Fragezeichen — kein „(0)". Der Abruf einer
	// Liste kann scheitern (Sweep „verschluckte Fehlantwort", 06.09.2026), und „Ausleiher
	// (0)" wäre dann eine Aussage über den Bestand, die niemand geprüft hat.
	/** @param {string} name @param {any[]} liste */
	const zaehler = (name, liste) =>
		`${name} (${akte.fehlendeListen.includes(name) ? '?' : liste.length})`;
	const tabs = $derived([
		['ausleiher', zaehler('Ausleiher', akte.borrowers)],
		['exemplare', zaehler('Exemplare', akte.exemplare)],
		...(darfVormerken ? [['vormerkungen', zaehler('Vormerkungen', akte.vormerkungen)]] : []),
		['historie', 'Historie']
	]);

	// Der Effekt hängt an GENAU einer Sache: der Titel-ID. untrack sorgt dafür, dass
	// nichts, was loadAll unterwegs liest (appState.selectedBook, sein eigener Zustand),
	// je wieder zum Auslöser dieses Effekts wird — am 06.09.2026 tat es das, und die
	// Akte lief in einer Endlosschleife, bis Svelte sie abbrach (siehe useBookAkte).
	$effect(() => {
		const id = bookId;
		if (id) untrack(() => akte.loadAll(id));
	});
</script>

<PageShell>
	<!-- Back Button + Breadcrumb -->
	<div class="flex items-center justify-between">
		<div class="flex items-center gap-3">
			<button
				onclick={onBack}
				class="flex items-center gap-2 px-3 py-2 rounded-xl text-slate-500 hover:text-slate-800 hover:bg-slate-100 transition-all text-sm font-semibold cursor-pointer"
			>
				<ChevronLeft class="w-4 h-4" aria-hidden="true" />
				Zurück zum Katalog
			</button>
			<span class="text-slate-400">/</span>
			<span class="text-slate-500 text-sm truncate max-w-xs">{akte.book?.title ?? 'Lade...'}</span>
		</div>
	</div>

	{#if akte.isLoading}
		<div class="flex justify-center items-center py-32">
			<Ladekreis size="lg" />
		</div>
	{:else if akte.book}
		<BookAkteMeta
			book={akte.book}
			borrowers={akte.borrowers}
			exemplare={akte.exemplare}
			coverSrc={akte.coverSrc}
			coverFailed={akte.coverFailed}
			onCoverError={akte.onCoverError}
			onCoverLoad={akte.onCoverLoad}
			onEdit={darfBearbeiten ? akte.editTitle : undefined}
			onDelete={darfLoeschen ? () => akte.deleteTitle(showToast, onBack) : undefined}
		/>

		{#if akte.fehlendeListen.length > 0}
			<p class="text-sm font-semibold text-error" role="alert">
				Nicht geladen: {akte.fehlendeListen.join(', ')}. Diese Reiter sind leer, weil der Abruf
				gescheitert ist — nicht, weil nichts da wäre.
			</p>
		{/if}

		<!-- Tabs -->
		<div class="border-b border-slate-200">
			<nav class="flex gap-6 overflow-x-auto no-scrollbar" aria-label="Buch-Akte Tabs">
				{#each tabs as [id, label] (id)}
					<button
						onclick={() => (akte.activeTab = id)}
						class="relative pb-3 text-sm font-semibold transition-colors cursor-pointer {akte.activeTab ===
						id
							? 'text-blue-600'
							: 'text-slate-500 hover:text-slate-700'}"
						role="tab"
						aria-selected={akte.activeTab === id}
					>
						{label}
						{#if akte.activeTab === id}
							<span class="absolute bottom-0 left-0 right-0 h-0.5 bg-blue-600 rounded-full"></span>
						{/if}
					</button>
				{/each}
			</nav>
		</div>

		<!-- Tab Content -->
		<div class="w-full">
			{#if akte.activeTab === 'ausleiher'}
				<BookBorrowersTab borrowers={akte.borrowers} book={akte.book} {onBack} />
			{:else if akte.activeTab === 'exemplare'}
				<BookExemplareTab bind:exemplare={akte.exemplare} book={akte.book} loadAll={akte.loadAll} />
			{:else if akte.activeTab === 'historie'}
				<BookHistoryTab history={akte.history} />
			{:else if akte.activeTab === 'vormerkungen'}
				<BookVormerkungenTab bind:vormerkungen={akte.vormerkungen} book={akte.book} />
			{/if}
		</div>
	{:else}
		<div class="py-24 flex flex-col items-center text-slate-400 gap-3">
			<Frown class="w-12 h-12" aria-hidden="true" />
			<p class="font-semibold">Buch nicht gefunden.</p>
			<button
				onclick={onBack}
				class="text-blue-600 text-sm font-semibold hover:underline cursor-pointer"
				>Zurück zum Katalog</button
			>
		</div>
	{/if}
</PageShell>
