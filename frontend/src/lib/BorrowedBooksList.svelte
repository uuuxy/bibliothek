<script>
	import { apiFetch } from './apiFetch.js';
	import Tabelle from './components/ui/Tabelle.svelte';
	import { SvelteSet } from 'svelte/reactivity';
	import { showToast } from '../inventur/lib/store.svelte.js';
	import { AlertTriangle, CalendarPlus, Undo2 } from '@lucide/svelte';
	import AusleiheRueckgabe from './AusleiheRueckgabe.svelte';
	import BuchCover from './components/ui/BuchCover.svelte';
	import CoverPeek from './components/ui/CoverPeek.svelte';
	import Ladekreis from './components/ui/Ladekreis.svelte';
	import StatusChip from './components/ui/StatusChip.svelte';

	/** @type {{ books: any[], onReturnClick?: (barcode: string) => void, onDamageClick?: (book: any) => void }} */
	let { books = [], onReturnClick = undefined, onDamageClick = undefined } = $props();

	const extendingIds = new SvelteSet();

	/** @param {any} book */
	async function handleExtend(book) {
		const id = book.ausleihe_id || book.id;
		if (!id || extendingIds.has(id)) return;
		extendingIds.add(id);

		try {
			const response = await apiFetch(`/api/ausleihen/${id}/verlaengern`, { method: 'POST' });
			if (response.ok) {
				const data = await response.json();
				book.rueckgabe_frist = data.neues_rueckgabe_datum;
				// Die Meldung nennt das neue Datum: Wer auf den Knopf sieht, sähe sonst nichts
				// passieren.
				const neu = new Date(data.neues_rueckgabe_datum).toLocaleDateString('de-DE');
				showToast(`Verlängert bis ${neu}.`, 'success');
			} else {
				const fehler = await response.json().catch(() => ({}));
				// Die Begründung des Servers („Ausleihe gesperrt (Grund)") sagt, was zu tun ist.
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

<!-- Eine Zeile je Buch (M3 Lists: „If the text doesn't fit on one line, it can wrap or be
     truncated"; „Lists can also show more or less content as they scale up and down in
     size"). Ein Schüler hat acht bis achtzehn Bücher: Der Titel bekommt die Breite, die die
     übrigen Spalten nicht brauchen; die Nummer des Exemplars bekommt ihre Spalte erst, wenn
     die Liste breit genug ist. Autor und Nummer stehen beim Zeigen auf dem Titel.
     Ohne eigenen Scrollkasten; gescrollt wird die Akte. -->
<div class="@container">
	<Tabelle beschriftung="Ausgeliehene Bücher">
		<thead>
			<tr>
				<th class="w-full">Titel</th>
				<th class="hidden @4xl:table-cell">Barcode</th>
				<th>Rückgabe</th>
				<th class="text-right">Aktion</th>
			</tr>
		</thead>
		<tbody>
			{#each books as book (book.id || book.barcode_id || Math.random())}
				<!-- Dauerleihe (Kollegium): keine Frist, nie überfällig — wie in der Sperr-Automatik. -->
				{@const ueberfaellig = !book.ist_dauerleihe && new Date(book.rueckgabe_frist) < new Date()}
				{@const ausleiheId = book.ausleihe_id || book.id}
				<tr>
					<!-- max-w-0 mit w-full: Die Zelle nimmt den Rest der Breite, und der Titel kürzt
					     sich, statt die Tabelle zu weiten. -->
					<td class="w-full max-w-0">
						<div class="flex items-center gap-3">
							<!-- Das Miniaturbild lässt die Zeile wiedererkennen und öffnet die Großansicht. -->
							<div class="shrink-0">
								<CoverPeek
									isbn={book.isbn || ''}
									coverUrl={book.cover_url || ''}
									titel={book.titel}
								>
									<BuchCover
										coverUrl={book.cover_url || ''}
										isbn={book.isbn || ''}
										titel={book.titel}
										groesse="klein"
										dekorativ
										nurGespeichert
									/>
								</CoverPeek>
							</div>
							<!-- Abgeschnitten statt überlappend: Reicht die Breite nicht einmal für das
							     Kennzeichen, ragt es nicht in die Spalte daneben. -->
							<div class="flex min-w-0 items-center gap-3 overflow-hidden">
								<span
									class="min-w-0 truncate font-semibold"
									data-tip={[book.titel, book.autor, book.barcode_id].filter(Boolean).join(' · ')}
								>
									{book.titel}
								</span>
								<!-- Die Sprechblase ist für Vorleseprogramme stumm; die Nummer steht hier nur,
								     solange ihre Spalte fehlt. -->
								<span class="sr-only">
									{#if book.autor}, {book.autor}{/if}<span class="@4xl:hidden"
										>, Barcode {book.barcode_id}</span
									>
								</span>
								<!-- Lernmittel kommt aus dem Feld (Migration 093), nicht aus dem Titeltext. -->
								{#if book.ist_lernmittel}
									<StatusChip text="Lernmittel" />
								{/if}
							</div>
						</div>
					</td>
					<td class="hidden whitespace-nowrap tabular-nums @4xl:table-cell">{book.barcode_id}</td>
					<td class="whitespace-nowrap"><AusleiheRueckgabe {book} {ueberfaellig} /></td>
					<!-- Symbole ohne Text: Die Spalte bleibt schmal, die Sprechblase nennt die Aktion,
					     und nach dem Klick sagt die Meldung, was geschehen ist. -->
					<td class="text-right whitespace-nowrap">
						<div class="flex items-center justify-end gap-1">
							<button
								type="button"
								class="icon-btn text-primary disabled:cursor-not-allowed disabled:text-on-surface/[0.38]"
								onclick={() => handleExtend(book)}
								disabled={extendingIds.has(ausleiheId)}
								data-tip="Um die Standard-Leihfrist verlängern"
								aria-label="Ausleihe verlängern"
							>
								{#if extendingIds.has(ausleiheId)}
									<Ladekreis size="sm" farbe="aktuell" />
								{:else}
									<CalendarPlus class="h-4 w-4" aria-hidden="true" />
								{/if}
							</button>
							{#if onDamageClick}
								<button
									type="button"
									class="icon-btn text-error"
									onclick={() => onDamageClick(book)}
									data-tip="Verlust oder Schaden melden"
									aria-label="Verlust oder Schaden melden"
								>
									<AlertTriangle class="h-4 w-4" aria-hidden="true" />
								</button>
							{/if}
							{#if onReturnClick}
								<button
									type="button"
									class="icon-btn text-success"
									onclick={() => onReturnClick(book.barcode_id)}
									data-tip="Buch zurückgeben"
									aria-label="Buch zurückgeben"
								>
									<Undo2 class="h-4 w-4" aria-hidden="true" />
								</button>
							{/if}
						</div>
					</td>
				</tr>
			{/each}
		</tbody>
	</Tabelle>
</div>
