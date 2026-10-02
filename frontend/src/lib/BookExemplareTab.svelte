<script>
	import { showToast } from '../inventur/lib/store.svelte.js';
	import { loeschenBestaetigen } from './stores/bestaetigung.svelte.js';
	import { authStore } from './stores/authStore.svelte.js';
	import { erlaubteTabs, hatRecht } from './menu.js';
	import { printQueue } from './stores/printQueue.svelte.js';
	import { SvelteSet } from 'svelte/reactivity';
	import { apiFetch } from './apiFetch.js';
	import BookExemplarCard from './components/BookExemplarCard.svelte';
	import ExemplarEigentumDialog from './components/ExemplarEigentumDialog.svelte';
	import AuswahlLeiste from './components/ui/AuswahlLeiste.svelte';
	import Button from './components/ui/Button.svelte';
	import { BookOpen, Trash2 } from '@lucide/svelte';

	/** @type {{ exemplare: any[], book: any, loadAll: (id: string) => void }} */
	let { exemplare = $bindable([]), book, loadAll } = $props();

	const selectedExemplare = new SvelteSet();
	// Auswahl/Löschen/Barcode/Status hängen an edit_books — nicht an der Rolle.
	const darfBearbeiten = $derived(hatRecht(authStore.currentUser, 'edit_books'));
	// Löschen hängt am Server an delete_books (routes_books.go) — die Leiste zeigt den Knopf
	// nur, wer ihn auch benutzen darf (UI entscheidet nach Recht).
	const darfLoeschen = $derived(hatRecht(authStore.currentUser, 'delete_books'));
	// Eigentum markierter Exemplare (4.24, Stufe 3): Dialog wie „Topf der Bestellung ändern".
	let eigentumOffen = $state(false);
	// Das Etikett entsteht im Druck-Center, auf dem Bogen nach der Vorlage (OFFEN.md 5.5). Wer
	// den Bildschirm nicht öffnen darf, bekäme statt des Bogens den ersten erlaubten: Der
	// Router stellt einen gesperrten Reiter zurück.
	const darfEtikett = $derived(
		darfBearbeiten && erlaubteTabs(authStore.currentUser).has('druck-center')
	);

	/** Dieselbe Übergabe wie Wareneingang und Nachdruck — das Druck-Center öffnet sich selbst. */
	function etikettDrucken(/** @type {any} */ ex) {
		printQueue.copies = [{ barcode_id: ex.barcode_id, titel: book.title, autor: book.author }];
	}

	/** @param {string} id */
	function toggleSelect(id) {
		if (selectedExemplare.has(id)) {
			selectedExemplare.delete(id);
		} else {
			selectedExemplare.add(id);
		}
	}

	/** Wie Littera „Exemplare: Alle" — ein Klassensatz hat 30 Bände und mehr. */
	function alleAuswaehlen() {
		for (const ex of exemplare) selectedExemplare.add(ex.id);
	}

	/** Nach dem Ändern: neu laden (Eigentum und Herkunft kommen vom Server) und Markierung weg. */
	async function eigentumGeaendert() {
		selectedExemplare.clear();
		if (book?.id) await loadAll(book.id);
	}

	/** @param {any} ex */
	async function deleteCopy(ex) {
		if (!(await loeschenBestaetigen(`Exemplar ${ex.barcode_id} löschen?`))) return;
		try {
			const res = await apiFetch(`/api/buecher/exemplare/${ex.id}`, {
				method: 'DELETE',
				credentials: 'include'
			});
			if (res.ok) {
				exemplare = exemplare.filter((e) => e.id !== ex.id);
				if (book) {
					book.gesamt = Math.max(0, (book.gesamt || 0) - 1);
					if (book.verfuegbar !== undefined && ex.ist_ausleihbar) {
						book.verfuegbar = Math.max(0, (book.verfuegbar || 0) - 1);
					}
				}
				showToast('Exemplar erfolgreich gelöscht', 'success');
			} else {
				const err = await res.json().catch(() => ({}));
				showToast(err.error || 'Fehler beim Löschen des Exemplars.', 'error');
			}
		} catch {
			showToast('Netzwerkfehler beim Löschen.', 'error');
		}
	}

	async function deleteSelectedCopies() {
		if (selectedExemplare.size === 0) return;
		if (!(await loeschenBestaetigen(`${selectedExemplare.size} Exemplare löschen?`))) return;

		let successCount = 0;

		const results = await Promise.allSettled(
			Array.from(selectedExemplare).map(async (id) => {
				const res = await apiFetch(`/api/buecher/exemplare/${id}`, {
					method: 'DELETE',
					credentials: 'include'
				});
				if (!res.ok) throw new Error('not ok');
				return id;
			})
		);

		for (const result of results) {
			if (result.status === 'fulfilled') {
				const id = result.value;
				exemplare = exemplare.filter((e) => e.id !== id);
				successCount++;
			} else {
				console.error('Fehler beim Löschen:', result.reason);
			}
		}
		selectedExemplare.clear();
		if (successCount > 0) {
			showToast(`${successCount} Exemplare erfolgreich gelöscht`, 'success');
			if (book && book.id) loadAll(book.id);
		}
	}
</script>

{#if exemplare.length === 0}
	<div class="py-16 flex flex-col items-center text-on-surface-variant gap-3">
		<BookOpen class="w-10 h-10" aria-hidden="true" />
		<p class="font-semibold text-sm">Keine physischen Exemplare mit Barcodes angelegt.</p>
	</div>
{:else}
	<div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
		{#each exemplare as ex (ex.id)}
			<BookExemplarCard
				{darfBearbeiten}
				{ex}
				selected={selectedExemplare.has(ex.id)}
				onToggleSelect={() => toggleSelect(ex.id)}
				onDelete={() => deleteCopy(ex)}
				onEtikett={darfEtikett ? () => etikettDrucken(ex) : undefined}
			/>
		{/each}
	</div>
	<!-- M3 Selection/Toolbars: die gemeinsame schwebende Leiste unter der Liste, wie auf der
	     Pflegeseite der Schlagworte. Bis zum 29.09.2026 stand hier ein eigener Balken oben. -->
	{#if selectedExemplare.size > 0 && darfBearbeiten}
		<AuswahlLeiste
			satz="{selectedExemplare.size} markiert"
			beschriftung="Aktionen für die markierten Exemplare"
			onleeren={() => selectedExemplare.clear()}
		>
			{#if selectedExemplare.size < exemplare.length}
				<Button variant="ghost" onclick={alleAuswaehlen}>Alle auswählen</Button>
			{/if}
			<Button onclick={() => (eigentumOffen = true)}>Eigentum ändern</Button>
			{#if darfLoeschen}
				<Button variant="danger" onclick={deleteSelectedCopies}>
					<Trash2 class="h-4 w-4" aria-hidden="true" />
					Löschen
				</Button>
			{/if}
		</AuswahlLeiste>
	{/if}
{/if}

<ExemplarEigentumDialog
	open={eigentumOffen}
	exemplarIds={Array.from(selectedExemplare)}
	onclose={() => (eigentumOffen = false)}
	onGeaendert={eigentumGeaendert}
/>
