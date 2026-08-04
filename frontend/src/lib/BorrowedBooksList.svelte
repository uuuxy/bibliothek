<script>
	import { apiFetch } from './apiFetch.js';
	import { SvelteSet } from 'svelte/reactivity';
	import { showToast } from '../inventur/lib/store.svelte.js';
	import { AlertTriangle, CalendarPlus, Loader2, Undo2 } from '@lucide/svelte';
	import Button from './components/ui/Button.svelte';
	import CoverPeek from './components/ui/CoverPeek.svelte';
	import { coverSrc } from './utils/coverSrc.js';

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
				showToast(
					`Rückgabedatum auf ${new Date(data.faellig_am).toLocaleDateString('de-DE')} gesetzt.`,
					'success'
				);
			} else {
				const fehler = await response.json().catch(() => ({}));
				showToast(fehler.error ?? 'Datum konnte nicht gespeichert werden.', 'error');
			}
		} catch (e) {
			console.error(e);
			showToast('Netzwerkfehler beim Speichern des Datums.', 'error');
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
				// Rückmeldung mit dem NEUEN Datum. Vorher änderte sich still eine Zahl in
				// einer anderen Spalte — wer auf den Knopf sah, sah nichts passieren und
				// hielt die Verlängerung für kaputt.
				const neu = new Date(data.neues_rueckgabe_datum).toLocaleDateString('de-DE');
				showToast(`Verlängert bis ${neu}.`, 'success');
			} else {
				const fehler = await response.json().catch(() => ({}));
				// Die Serverbegründung durchreichen: „Ausleihe gesperrt (Grund)" sagt, was zu
				// tun ist — „Fehler bei der Verlängerung" lässt den Nutzer ratlos zurück.
				showToast(fehler.error ?? 'Verlängerung nicht möglich.', 'error');
			}
		} catch (e) {
			console.error(e);
			showToast('Netzwerkfehler bei der Verlängerung.', 'error');
		} finally {
			extendingIds.delete(id);
		}
	}
</script>

<div class="max-h-64 overflow-y-auto pr-2 custom-scrollbar">
	<table class="w-full text-left border-collapse table-fixed">
		<thead>
			<tr class="border-b border-slate-200 text-xs font-medium text-slate-600">
				<!-- table-fixed + PROZENT-Breiten (Summe 100%): Die Liste steht auch in schmalen
				     Panels (Schülerprofil ~590px). Mit festen px-Breiten (w-32/w-48/w-28/w-40 =
				     592px) blieb dort für die Titelspalte 0px übrig — der truncatete Titel
				     (overflow:hidden) wurde damit unsichtbar und die Aktions-Spalte abgeschnitten.
				     Prozente skalieren mit jeder Containerbreite und können nie kollabieren. -->
				<!-- truncate auch auf den Kopfzellen: bei table-fixed ragt zu langer Kopftext
				     sonst in die Nachbarspalte hinein (überlappte „RÜCKGABEDATUM|STATUS"). -->
				<th class="py-3 px-4 truncate {mode === 'loans' ? 'w-[30%]' : 'w-[60%]'}">Titel & Autor</th>
				<th class="py-3 px-4 truncate {mode === 'loans' ? 'w-[14%]' : 'w-[22%]'}">Barcode</th>
				{#if mode === 'loans'}
					<th class="py-3 px-4 truncate w-[20%]">Rückgabe</th>
					<th class="py-3 px-4 truncate w-[19%]">Status</th>
				{/if}
				<th class="py-3 px-4 truncate text-right {mode === 'loans' ? 'w-[17%]' : 'w-[18%]'}"
					>Aktion</th
				>
			</tr>
		</thead>
		<tbody class="divide-y divide-slate-100">
			{#each books as book (book.id || book.barcode_id || Math.random())}
				{@const isLMF = book.titel?.toLowerCase().startsWith('lmf-')}
				{@const isOverdue = mode === 'loans' && new Date(book.rueckgabe_frist) < new Date()}
				<!-- Miniatur und Großansicht ziehen ihre Quelle aus derselben Funktion. Vorher lud
				     die Miniatur den Pfad direkt, die Großansicht dagegen über den Cover-Proxy —
				     der lokale Pfade ablehnt. Ergebnis: sichtbares Cover in der Zeile, "Kein
				     Coverbild hinterlegt" in der Vergrößerung. -->
				{@const miniatur = coverSrc(book.cover_url, book.isbn)}
				<tr class="hover:bg-slate-50 transition-colors">
					<td class="py-3 px-4">
						<div class="flex items-center space-x-3">
							<!-- Das Miniaturbild ist der Auslöser für die Großansicht: 32×48 px reichen,
							     um eine Zeile wiederzuerkennen, nicht um ein Cover zu prüfen. Ein
							     dauerhaft größeres Bild kostete in dieser dichten Liste die Übersicht
							     (in der Bestellliste waren es 53 px Zeilenhöhe), deshalb erscheint es
							     nur auf Anforderung — und dann auch per Tastatur und auf dem iPad. -->
							<CoverPeek isbn={book.isbn || ''} coverUrl={book.cover_url || ''} titel={book.titel}>
								{#if miniatur}
									<img
										src={miniatur}
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
							</CoverPeek>
							<div class="flex-1 min-w-0">
								<div class="flex items-center gap-2 min-w-0">
									<h4 class="font-bold text-sm text-slate-900 truncate min-w-0" title={book.titel}>
										{book.titel}
									</h4>
									{#if isLMF}
										<span
											class="px-1.5 py-0.5 rounded text-label-small font-bold bg-indigo-50 text-indigo-700 border border-indigo-100 uppercase"
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
					<!-- truncate: lange Barcodes brechen sonst um und überlappen die Nachbarspalte -->
					<td
						class="py-3 px-4 text-sm font-semibold text-slate-700 truncate"
						title={book.barcode_id}>{book.barcode_id}</td
					>
					{#if mode === 'loans'}
						<td class="py-3 px-4 text-sm font-semibold text-slate-700 whitespace-nowrap">
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
										class="opacity-0 group-hover:opacity-100 p-0.5 text-slate-400 hover:text-blue-600 transition-opacity cursor-pointer"
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
						<td class="py-3 px-4 whitespace-nowrap">
							<!-- Farbe nur für die Ausnahme: „In Frist" ist der Normalfall einer
							     laufenden Ausleihe und braucht kein grünes Abzeichen. Grün auf jeder
							     Zeile heisst nur, dass Grün nichts mehr bedeutet. -->
							{#if isOverdue}
								<span class="text-sm font-medium text-rose-600">Überfällig</span>
							{:else}
								<span class="text-sm text-slate-500">In Frist</span>
							{/if}
						</td>
					{/if}
					<!-- Icons ohne Text, aber mit eindeutigem Symbol. Beschriftungen wurden
					     versucht und wieder verworfen: Die Liste steht im Schülerprofil in einem
					     ~590-px-Panel, die Aktionsspalte hat 17 % davon — „Verlängern" überdeckte
					     dort die Status-Spalte. Statt Platz zu erzwingen, trägt das Symbol jetzt
					     die Bedeutung: Kalender-Plus statt Uhr (eine Uhr heisst „Zeit", nicht
					     „Frist verlängern"). Die Rückmeldung nach dem Klick liefert der Toast. -->
					<td class="py-3 px-4 text-right">
						<div class="flex items-center justify-end gap-1">
							{#if mode === 'loans'}
								<Button
									variant="ghost"
									size="sm"
									onclick={() => handleExtend(book)}
									disabled={extendingIds.has(book.ausleihe_id || book.id)}
									title="Um die Standard-Leihfrist verlängern"
									aria-label="Ausleihe verlängern"
									class="px-2 text-blue-600 hover:bg-blue-50"
								>
									{#if extendingIds.has(book.ausleihe_id || book.id)}
										<Loader2 class="h-4 w-4 animate-spin" aria-hidden="true" />
									{:else}
										<CalendarPlus class="h-4 w-4" aria-hidden="true" />
									{/if}
								</Button>
								{#if onDamageClick}
									<Button
										variant="ghost"
										size="sm"
										onclick={() => onDamageClick(book)}
										title="Verlust oder Schaden melden"
										aria-label="Verlust oder Schaden melden"
										class="px-2 text-rose-700 hover:bg-rose-50"
									>
										<AlertTriangle class="h-4 w-4" aria-hidden="true" />
									</Button>
								{/if}
								{#if onReturnClick}
									<Button
										variant="ghost"
										size="sm"
										onclick={() => onReturnClick(book.barcode_id)}
										title="Buch zurückgeben"
										aria-label="Buch zurückgeben"
										class="px-2 text-emerald-700 hover:bg-emerald-50"
									>
										<Undo2 class="h-4 w-4" aria-hidden="true" />
									</Button>
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
