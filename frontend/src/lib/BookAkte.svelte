<script>
	import { showToast } from '../inventur/lib/store.svelte.js';
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
	const tabs = $derived([
		['ausleiher', `Ausleiher (${akte.borrowers.length})`],
		['exemplare', `Exemplare (${akte.exemplare.length})`],
		...(darfVormerken ? [['vormerkungen', `Vormerkungen (${akte.vormerkungen.length})`]] : []),
		['historie', 'Historie']
	]);

	$effect(() => {
		if (bookId) akte.loadAll(bookId);
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
			<div
				class="w-10 h-10 border-4 border-blue-600 border-t-transparent rounded-full animate-spin"
			></div>
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
