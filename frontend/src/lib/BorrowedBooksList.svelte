<script>
	import { apiFetch, apiClient } from './apiFetch.js';
	import { SvelteSet } from 'svelte/reactivity';

	/** @type {{ books: any[], onReturnClick?: (barcode: string) => void, onDamageClick?: (book: any) => void, mode?: "loans" | "scans" }} */
	let {
		books = [],
		onReturnClick = undefined,
		onDamageClick = undefined,
		mode = 'loans'
	} = $props();

	const extendingIds = new SvelteSet();

	let editingId = $state(null);
	let editingDate = $state('');
	let isSavingDate = $state(false);

	async function handleSaveDate(book) {
		const id = book.ausleihe_id || book.id;
		if (!id || !editingDate) return;

		isSavingDate = true;
		try {
			const response = await apiFetch(`/api/admin/ausleihen/${id}/faelligkeit`, {
				method: 'PATCH',
				body: JSON.stringify({ faellig_am: editingDate })
			});
			if (response.ok) {
				const data = await response.json();
				book.rueckgabe_frist = data.faellig_am;
				editingId = null;
			} else {
				alert('Fehler beim Speichern des Datums');
			}
		} catch (e) {
			console.error(e);
			alert('Netzwerkfehler');
		} finally {
			isSavingDate = false;
		}
	}

	async function handleExtend(book) {
		const id = book.ausleihe_id || book.id;
		if (!id || extendingIds.has(id)) return;

		extendingIds.add(id);

		try {
			const response = await apiFetch(`/api/ausleihen/${id}/verlaengern`, { method: 'POST' });
			if (response.ok) {
				const data = await response.json();
				book.rueckgabe_frist = data.neues_rueckgabe_datum;
			} else {
				alert('Fehler bei der Verlängerung');
			}
		} catch (e) {
			console.error(e);
			alert('Netzwerkfehler');
		} finally {
			extendingIds.delete(id);
		}
	}
</script>

<div class="max-h-64 overflow-y-auto pr-2 custom-scrollbar">
	<table class="w-full text-left border-collapse">
		<thead>
			<tr
				class="border-b border-slate-200 text-xs font-bold text-slate-600 uppercase tracking-wider"
			>
				<th class="py-3 px-4">Titel & Autor</th>
				<th class="py-3 px-4">Barcode</th>
				{#if mode === 'loans'}
					<th class="py-3 px-4">Rückgabedatum</th>
					<th class="py-3 px-4">Status</th>
				{/if}
				<th class="py-3 px-4 text-right">Aktion</th>
			</tr>
		</thead>
		<tbody class="divide-y divide-slate-100">
			{#each books as book (book.id || book.barcode_id || Math.random())}
				{@const isLMF = book.titel?.toLowerCase().startsWith('lmf-')}
				{@const isOverdue = mode === 'loans' && new Date(book.rueckgabe_frist) < new Date()}
				<tr class="hover:bg-slate-50 transition-colors">
					<td class="py-3 px-4">
						<div class="flex items-center space-x-3">
							{#if book.cover_url}
								<img
									src={book.cover_url}
									class="w-8 h-12 object-cover rounded shadow-sm border border-slate-100"
									alt="Cover"
								/>
							{:else}
								<div
									class="w-8 h-12 rounded shadow-sm flex items-center justify-center font-bold text-white bg-linear-to-br from-indigo-500 to-purple-600 text-xs border border-indigo-600/10"
								>
									{book.titel ? book.titel.charAt(0).toUpperCase() : '?'}
								</div>
							{/if}
							<div class="flex-1 min-w-0">
								<div class="flex items-center gap-2">
									<h4 class="font-bold text-sm text-slate-900 truncate">{book.titel}</h4>
									{#if isLMF}
										<span
											class="px-1.5 py-0.5 rounded text-[10px] font-bold bg-indigo-50 text-indigo-700 border border-indigo-100 uppercase"
											>LMF</span
										>
									{/if}
								</div>
								{#if mode === 'loans'}
									<div class="text-xs text-slate-600 truncate mt-0.5">{book.autor}</div>
								{/if}
							</div>
						</div>
					</td>
					<td class="py-3 px-4 text-sm font-semibold text-slate-700">{book.barcode_id}</td>
					{#if mode === 'loans'}
						<td class="py-3 px-4 text-sm font-semibold text-slate-700">
							{#if editingId === (book.ausleihe_id || book.id)}
								<div class="flex items-center gap-1.5">
									<input
										type="date"
										bind:value={editingDate}
										class="border border-slate-300 rounded px-1.5 py-0.5 text-xs bg-white text-slate-700 outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500 transition-shadow disabled:opacity-50"
										disabled={isSavingDate}
									/>
									<button
										onclick={() => handleSaveDate(book)}
										disabled={isSavingDate}
										class="p-1 text-emerald-600 hover:bg-emerald-50 rounded disabled:opacity-50 transition-colors cursor-pointer"
										title="Speichern"
										aria-label="Rückgabedatum speichern"
									>
										<svg
											class="w-4 h-4"
											fill="none"
											viewBox="0 0 24 24"
											stroke="currentColor"
											aria-hidden="true"
											><path
												stroke-linecap="round"
												stroke-linejoin="round"
												stroke-width="2.5"
												d="M5 13l4 4L19 7"
											/></svg
										>
									</button>
									<button
										onclick={() => (editingId = null)}
										disabled={isSavingDate}
										class="p-1 text-rose-600 hover:bg-rose-50 rounded disabled:opacity-50 transition-colors cursor-pointer"
										title="Abbrechen"
										aria-label="Bearbeiten abbrechen"
									>
										<svg
											class="w-4 h-4"
											fill="none"
											viewBox="0 0 24 24"
											stroke="currentColor"
											aria-hidden="true"
											><path
												stroke-linecap="round"
												stroke-linejoin="round"
												stroke-width="2.5"
												d="M6 18L18 6M6 6l12 12"
											/></svg
										>
									</button>
								</div>
							{:else}
								<div class="flex items-center gap-2 group">
									<span>{new Date(book.rueckgabe_frist).toLocaleDateString('de-DE')}</span>
									<button
										onclick={() => {
											editingId = book.ausleihe_id || book.id;
											editingDate = book.rueckgabe_frist.split('T')[0];
										}}
										class="opacity-0 group-hover:opacity-100 p-0.5 text-gray-400 hover:text-blue-600 transition-opacity cursor-pointer"
										title="Datum bearbeiten"
										aria-label="Rückgabedatum bearbeiten"
									>
										<svg
											class="w-3.5 h-3.5"
											fill="none"
											viewBox="0 0 24 24"
											stroke="currentColor"
											aria-hidden="true"
											><path
												stroke-linecap="round"
												stroke-linejoin="round"
												stroke-width="2"
												d="M15.232 5.232l3.536 3.536m-2.036-5.036a2.5 2.5 0 113.536 3.536L6.5 21.036H3v-3.572L16.732 3.732z"
											/></svg
										>
									</button>
								</div>
								<div class="text-xs font-normal text-slate-600 mt-0.5">
									Geliehen: {new Date(book.ausgeliehen_am).toLocaleDateString('de-DE')}
								</div>
							{/if}
						</td>
						<td class="py-3 px-4">
							{#if isOverdue}
								<span
									class="px-2 py-1 bg-rose-50 text-rose-600 text-xs font-bold rounded-full border border-rose-100"
									>Überfällig</span
								>
							{:else}
								<span
									class="px-2 py-1 bg-emerald-50 text-emerald-600 text-xs font-bold rounded-full border border-emerald-100"
									>In Frist</span
								>
							{/if}
						</td>
					{/if}
					<td class="py-3 px-4 text-right">
						<div class="flex items-center justify-end gap-2">
							{#if mode === 'loans'}
								<button
									onclick={() => handleExtend(book)}
									disabled={extendingIds.has(book.ausleihe_id || book.id)}
									class="p-2 bg-blue-50 hover:bg-blue-100 text-blue-600 disabled:opacity-50 rounded-full transition-colors cursor-pointer"
									title="Verlängern"
									aria-label="Ausleihe verlängern"
								>
									{#if extendingIds.has(book.ausleihe_id || book.id)}
										<svg
											class="w-4 h-4 animate-spin text-blue-400"
											xmlns="http://www.w3.org/2000/svg"
											fill="none"
											viewBox="0 0 24 24"
											aria-hidden="true"
											><circle
												class="opacity-25"
												cx="12"
												cy="12"
												r="10"
												stroke="currentColor"
												stroke-width="4"
											></circle><path
												class="opacity-75"
												fill="currentColor"
												d="M4 12a8 8 0 018-8v8H4z"
											></path></svg
										>
									{:else}
										<svg
											class="w-4 h-4"
											fill="none"
											viewBox="0 0 24 24"
											stroke="currentColor"
											aria-hidden="true"
											><path
												stroke-linecap="round"
												stroke-linejoin="round"
												stroke-width="2.5"
												d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"
											/></svg
										>
									{/if}
								</button>
								{#if onDamageClick}
									<button
										onclick={() => onDamageClick(book)}
										class="p-2 bg-rose-100 hover:bg-rose-200 text-rose-700 rounded-full transition-colors cursor-pointer"
										title="Verlust/Schaden melden"
										aria-label="Verlust oder Schaden melden"
									>
										<svg
											class="w-4 h-4"
											fill="none"
											viewBox="0 0 24 24"
											stroke="currentColor"
											aria-hidden="true"
											><path
												stroke-linecap="round"
												stroke-linejoin="round"
												stroke-width="2.5"
												d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"
											/></svg
										>
									</button>
								{/if}
								{#if onReturnClick}
									<button
										onclick={() => onReturnClick(book.barcode_id)}
										class="p-2 bg-emerald-100 hover:bg-emerald-200 text-emerald-700 rounded-full transition-colors cursor-pointer"
										title="Buch zurückgeben"
										aria-label="Buch zurückgeben"
									>
										<svg
											class="w-4 h-4"
											fill="none"
											viewBox="0 0 24 24"
											stroke="currentColor"
											aria-hidden="true"
											><path
												stroke-linecap="round"
												stroke-linejoin="round"
												stroke-width="2.5"
												d="M3 10h10a8 8 0 018 8v2M3 10l6 6m-6-6l6-6"
											/></svg
										>
									</button>
								{/if}
							{:else if mode === 'scans'}
								<svg
									class="w-5 h-5 text-emerald-500"
									fill="none"
									viewBox="0 0 24 24"
									stroke="currentColor"
									><path
										stroke-linecap="round"
										stroke-linejoin="round"
										stroke-width="2.5"
										d="M5 13l4 4L19 7"
									/></svg
								>
							{/if}
						</div>
					</td>
				</tr>
			{/each}
		</tbody>
	</table>
</div>
