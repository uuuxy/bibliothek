<script>
	import { showToast } from '../inventur/lib/store.svelte.js';
	import { authStore } from './stores/authStore.svelte.js';
	import { hatRecht } from './menu.js';
	import { SvelteSet } from 'svelte/reactivity';
	import { apiFetch } from './apiFetch.js';
	import BookExemplarCard from './components/BookExemplarCard.svelte';
	import Button from './components/ui/Button.svelte';
	import { BookOpen } from '@lucide/svelte';

	/** @type {{ exemplare: any[], book: any, loadAll: (id: string) => void }} */
	let { exemplare = $bindable([]), book, loadAll } = $props();

	const selectedExemplare = new SvelteSet();
	// Auswahl/Löschen/Barcode/Status hängen an edit_books — nicht an der Rolle.
	const darfBearbeiten = $derived(hatRecht(authStore.currentUser, 'edit_books'));

	/** @param {string} id */
	function toggleSelect(id) {
		if (selectedExemplare.has(id)) {
			selectedExemplare.delete(id);
		} else {
			selectedExemplare.add(id);
		}
	}

	/** @param {any} ex */
	async function deleteCopy(ex) {
		if (!confirm(`Möchtest du das Exemplar ${ex.barcode_id} wirklich unwiderruflich löschen?`))
			return;
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
				alert(err.error || 'Fehler beim Löschen des Exemplars.');
			}
		} catch {
			alert('Netzwerkfehler beim Löschen.');
		}
	}

	async function deleteSelectedCopies() {
		if (selectedExemplare.size === 0) return;
		if (
			!confirm(
				`Möchtest du die ${selectedExemplare.size} ausgewählten Exemplare unwiderruflich löschen?`
			)
		)
			return;

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
	<div class="py-16 flex flex-col items-center text-slate-400 gap-3">
		<BookOpen class="w-10 h-10" aria-hidden="true" />
		<p class="font-semibold text-sm">Keine physischen Exemplare mit Barcodes angelegt.</p>
	</div>
{:else}
	{#if selectedExemplare.size > 0 && darfBearbeiten}
		<div
			class="mb-4 p-3 bg-rose-50 border border-rose-100 rounded-xl flex items-center justify-between animate-fade-in"
		>
			<span class="text-sm font-semibold text-rose-800"
				>{selectedExemplare.size} Exemplare ausgewählt</span
			>
			<Button variant="danger-solid" size="sm" onclick={deleteSelectedCopies}>
				Ausgewählte löschen
			</Button>
		</div>
	{/if}
	<div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
		{#each exemplare as ex (ex.id)}
			<BookExemplarCard
				{darfBearbeiten}
				{ex}
				selected={selectedExemplare.has(ex.id)}
				onToggleSelect={() => toggleSelect(ex.id)}
				onDelete={() => deleteCopy(ex)}
			/>
		{/each}
	</div>
{/if}
