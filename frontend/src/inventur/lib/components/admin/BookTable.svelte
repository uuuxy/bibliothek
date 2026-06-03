<script>
	import BookTableToolbar from "$lib/components/admin/BookTableToolbar.svelte";
	import BookTableZeile from "$lib/components/admin/BookTableZeile.svelte";
	import { csrfHeader } from "$lib/csrf.js";

	/**
	 * @type {{
	 *   books: any[],
	 *   loading: boolean,
	 *   onOpenDetail: (book: any) => void,
	 *   onImportExcel: (event: Event) => void,
	 *   onCreateNew: () => void,
	 *   onScan: () => void,
	 *   onDelete: (ids: string[]) => void,
	 *   onRetryCovers: () => void
	 * }}
	 */
	let {
		books,
		loading,
		onOpenDetail,
		onImportExcel,
		onCreateNew,
		onScan,
		onDelete,
		onRetryCovers,
	} = $props();

	/** @type {string[]} */
	let selectedIds = $state([]);
	/** @type {number|null} */
	let draggedIndex = $state(null);
	/** @type {number|null} */
	let dragOverIndex = $state(null);

	/**
	 * @param {{ type: string, message: string }} param0
	 */
	function addToast({ type, message }) {
		if (type === "error") {
			console.error(message);
		}
	}

	function toggleSelectAll() {
		if (selectedIds.length === books.length) {
			selectedIds = [];
			return;
		}
		selectedIds = books.map((book) => book.id);
	}

	/**
	 * @param {string} id
	 */
	function toggleSelect(id) {
		if (selectedIds.includes(id)) {
			selectedIds = selectedIds.filter((selectedId) => selectedId !== id);
			return;
		}
		selectedIds = [...selectedIds, id];
	}

	function handleDelete() {
		onDelete(selectedIds);
		selectedIds = [];
	}

	/**
	 * @param {DragEvent} event
	 * @param {number} index
	 */
	function onDragStart(event, index) {
		draggedIndex = index;
		if (event.dataTransfer) {
			event.dataTransfer.effectAllowed = "move";
			event.dataTransfer.setData("text/plain", String(index));
		}
		const target = /** @type {HTMLElement} */ (event.target);
		setTimeout(() => {
			target.classList.add("opacity-50");
		}, 0);
	}

	/**
	 * @param {DragEvent} event
	 * @param {number} index
	 */
	function onDragOver(event, index) {
		event.preventDefault();
		if (event.dataTransfer) {
			event.dataTransfer.dropEffect = "move";
		}
		if (draggedIndex === null || draggedIndex === index) return;
		dragOverIndex = index;
	}

	/**
	 * @param {DragEvent} event
	 * @param {number} index
	 */
	function onDragLeave(event, index) {
		if (dragOverIndex === index) {
			dragOverIndex = null;
		}
	}

	/**
	 * @param {DragEvent} event
	 */
	function onDragEnd(event) {
		const target = /** @type {HTMLElement} */ (event.target);
		target.classList.remove("opacity-50");
		draggedIndex = null;
		dragOverIndex = null;
	}

	/**
	 * @param {DragEvent} event
	 * @param {number} index
	 */
	async function onDrop(event, index) {
		event.preventDefault();
		if (draggedIndex === null || draggedIndex === index) return;

		const movedBook = books[draggedIndex];
		const reorderedBooks = [...books];
		reorderedBooks.splice(draggedIndex, 1);
		reorderedBooks.splice(index, 0, movedBook);

		books.length = 0;
		books.push(...reorderedBooks);
		draggedIndex = null;
		dragOverIndex = null;

		try {
			const bookIds = books.map((book) => book.id);
			const response = await fetch("/api/admin/books/reorder", {
				method: "PUT",
				credentials: "include",
				headers: /** @type {HeadersInit} */ ({
					"Content-Type": "application/json",
					...csrfHeader(),
				}),
				body: JSON.stringify({ bookIds }),
			});

			if (!response.ok) {
				throw new Error("Network response was not ok");
			}
			addToast({ type: "success", message: "Sortierung gespeichert" });
		} catch (error) {
			console.error("Fehler beim Speichern der Sortierung:", error);
			addToast({ type: "error", message: "Sortierung konnte nicht gespeichert werden" });
		}
	}
</script>

<div class="bg-white rounded-2xl border border-slate-100 overflow-hidden shadow-xs">
	<BookTableToolbar
		booksLength={books.length}
		selectedCount={selectedIds.length}
		onDelete={handleDelete}
		onScan={onScan}
		onImportExcel={onImportExcel}
		onCreateNew={onCreateNew}
		onRetryCovers={onRetryCovers}
	/>

	<div class="overflow-x-auto">
		<table class="w-full text-left text-base text-slate-700">
			<thead
				class="bg-slate-50 border-b border-slate-100 uppercase tracking-wider text-[10px] font-bold text-slate-500 font-sans"
			>
				<tr>
					<th class="px-6 py-4 w-10">
						<input
							type="checkbox"
							class="rounded border-slate-200 bg-white text-blue-600 focus:ring-blue-500/20 cursor-pointer"
							checked={books.length > 0 && selectedIds.length === books.length}
							onclick={toggleSelectAll}
						/>
					</th>
					<th class="px-6 py-4 w-20">Cover</th>
					<th class="px-6 py-4">Titel</th>
					<th class="px-6 py-4">Fach</th>
					<th class="px-6 py-4">Klasse</th>
					<th class="px-6 py-4">Zweig</th>
					<th class="px-6 py-4">Standort</th>
					<th class="px-6 py-4 text-right">Zuletzt geprüft</th>
					<th class="px-6 py-4 text-right">Bestand</th>
					<th class="px-6 py-4 w-10"></th>
				</tr>
			</thead>

			<tbody class="divide-y divide-slate-100">
				{#each books as book, index (book.id)}
					<BookTableZeile
						{book}
						{index}
						{dragOverIndex}
						isSelected={selectedIds.includes(book.id)}
						onOpenDetail={onOpenDetail}
						onToggleSelect={toggleSelect}
						onDragStart={onDragStart}
						onDragOver={onDragOver}
						onDragLeave={onDragLeave}
						onDrop={onDrop}
						onDragEnd={onDragEnd}
					/>
				{/each}

				{#if books.length === 0 && !loading}
					<tr>
						<td colspan="10" class="px-6 py-12 text-center text-slate-400 font-medium">
							Keine Bücher gefunden.
						</td>
					</tr>
				{/if}
			</tbody>
		</table>
	</div>
</div>

