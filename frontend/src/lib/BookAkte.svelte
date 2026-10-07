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
	import LadeFehler from './components/ui/LadeFehler.svelte';
	import Reiter from './components/ui/Reiter.svelte';
	import { authStore } from './stores/authStore.svelte.js';
	import { hatRecht } from './menu.js';
	import { exemplarZahlen } from './components/exemplarStatus.js';
	import { ChevronLeft, FaceSlightlyFrowning } from '@lucide/svelte';
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
	/** @param {string} name @param {number} anzahl */
	const zaehler = (name, anzahl) =>
		`${name} (${akte.fehlendeListen.includes(name) ? '?' : anzahl})`;
	// „Exemplare" zählt den Bestand wie „1 von 2 verfügbar" im Kopf; bestellte und
	// ausgesonderte stehen im Reiter mit ihrem Wort an der Karte.
	const tabs = $derived([
		{ id: 'ausleiher', label: zaehler('Ausleiher', akte.borrowers.length) },
		{ id: 'exemplare', label: zaehler('Exemplare', exemplarZahlen(akte.exemplare).bestand) },
		...(darfVormerken
			? [{ id: 'vormerkungen', label: zaehler('Vormerkungen', akte.vormerkungen.length) }]
			: []),
		{ id: 'historie', label: 'Historie' }
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
				class="flex items-center gap-2 px-3 py-2 rounded-xl text-on-surface-variant hover:text-on-surface transition-colors text-sm font-semibold cursor-pointer"
			>
				<ChevronLeft class="w-4 h-4" aria-hidden="true" />
				Zurück zum Katalog
			</button>
			<span class="text-on-surface-variant">/</span>
			<span class="text-on-surface-variant text-sm truncate max-w-xs"
				>{akte.book?.title ?? 'Lade...'}</span
			>
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

		<Reiter
			etikett="Bereiche der Buchakte"
			reiter={tabs}
			aktiv={akte.activeTab}
			onwahl={(id) => (akte.activeTab = id)}
		/>

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
	{:else if akte.kopfFehler}
		<!-- „Nicht gefunden" wäre hier eine Aussage über den Bestand. Der Abruf ist
		     gescheitert — den Titel gibt es womöglich, es kam nur nichts an. -->
		<LadeFehler
			onerneut={() => bookId && akte.loadAll(bookId)}
			titel="Titel nicht geladen"
			text={akte.kopfFehler}
		/>
	{:else}
		<div class="py-24 flex flex-col items-center text-on-surface-variant gap-3">
			<FaceSlightlyFrowning class="w-12 h-12" aria-hidden="true" />
			<p class="font-semibold">Buch nicht gefunden.</p>
			<button
				onclick={onBack}
				class="text-primary text-sm font-semibold hover:underline cursor-pointer"
				>Zurück zum Katalog</button
			>
		</div>
	{/if}
</PageShell>
