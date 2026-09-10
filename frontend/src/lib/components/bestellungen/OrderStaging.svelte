<!-- @component Das Staging-Fenster der Bestellsuche: ein Treffer wird geprüft, bevor er
     in den Warenkorb kommt — Signatur, Menge, Barcodes und seit dem 10.09.2026 die Frage
     „Lernmittel?".

     Warum die Frage hier steht: Ein über die DNB neu angelegter Titel entstand bis dahin
     OHNE Lernmittel-Kennzeichen und blieb so — ein neues Schulbuch war damit ein
     Bücherei-Titel: falsche Frist, falscher Katalog, unsichtbar im Bestellbedarf, und mit
     Migration 109 im falschen Topf der Bestellung. Das Fenster ist der Moment, in dem
     jemand den Titel ohnehin ansieht.

     Eigene Datei, seit OrderSearch mit dem Fenster über der 200-Zeilen-Marke lag. -->
<script>
	import { apiPut } from '../../apiFetch.js';
	import { toastStore } from '../../stores/toastStore.svelte.js';
	import { orderStore } from '../../stores/orderStore.svelte.js';
	import Button from '../ui/Button.svelte';
	import Kaestchen from '../ui/Kaestchen.svelte';
	import Feld from '../ui/Feld.svelte';
	import BuchCover from '../ui/BuchCover.svelte';
	import { untrack } from 'svelte';

	/**
	 * @type {{ book: any, onDone: () => void }}
	 * book — der Treffer (lokal oder eben über /aus-isbn angelegt); onDone schließt das
	 * Fenster, ob übernommen oder abgebrochen.
	 */
	let { book, onDone } = $props();

	// Bewusst der ANFANGSWERT des Treffers (untrack): Das Fenster ist ein Formular über
	// einem Treffer, OrderSearch hängt es je Treffer neu ein ({#key}). Folgte es dem Prop
	// weiter, überschriebe ein Nachladen die Eingabe.
	let menge = $state(1);
	let generateBarcodes = $state(true);
	let signatur = $state(untrack(() => book.signatur ?? ''));
	let istLernmittel = $state(untrack(() => Boolean(book.ist_lernmittel)));
	// Vergleichswerte, um beim Bestätigen zu erkennen, ob tatsächlich bearbeitet wurde —
	// unverändert übernommen wird nie ein zusätzlicher Request ausgelöst (weder für einen
	// unangetasteten Vorschlag noch für eine bereits vorhandene Angabe).
	const signaturBeiStart = untrack(() => book.signatur ?? '');
	const lernmittelBeiStart = untrack(() => Boolean(book.ist_lernmittel));

	async function uebernehmen() {
		const neueSignatur = signatur.trim();
		if (neueSignatur !== signaturBeiStart) {
			try {
				await apiPut(`/api/buecher/titel/${book.id}/signatur`, { signatur: neueSignatur });
			} catch {
				toastStore.addToast(
					'Signatur konnte nicht gespeichert werden — Titel wird trotzdem bestellt.',
					'error'
				);
			}
		}
		if (istLernmittel !== lernmittelBeiStart) {
			try {
				await apiPut(`/api/buecher/titel/${book.id}/lernmittel`, { ist_lernmittel: istLernmittel });
			} catch {
				toastStore.addToast(
					'Lernmittel-Kennzeichen konnte nicht gespeichert werden — Titel wird trotzdem bestellt.',
					'error'
				);
			}
		}
		// Der Warenkorb bekommt den GEPRÜFTEN Stand, nicht den Treffer: Daraus wird der
		// Vorschlag für den Topf der Bestellung.
		orderStore.addToCart(
			{ ...book, signatur: neueSignatur, ist_lernmittel: istLernmittel },
			menge,
			generateBarcodes
		);
		onDone();
	}
</script>

<div class="mt-3 p-4 rounded-xl border border-blue-200 bg-blue-50/60 space-y-3.5 animate-fade-in">
	<div class="flex items-center gap-3 min-w-0">
		<BuchCover coverUrl={book.cover_url} isbn={book.isbn} titel={book.titel} klasse="shrink-0" />
		<div class="min-w-0">
			<div class="font-bold text-slate-900 text-sm truncate">{book.titel}</div>
			<div class="text-xs text-slate-500 truncate">{book.autor}</div>
		</div>
	</div>

	<div class="space-y-1">
		<label for="stagedSignaturInput" class="text-xs font-medium text-slate-500">
			Signatur
			{#if !signaturBeiStart}
				<span class="text-amber-600 font-normal">(Vorschlag, bitte prüfen)</span>
			{/if}
		</label>
		<Feld
			id="stagedSignaturInput"
			bind:value={signatur}
			placeholder="z. B. BIB Jugendbuch"
			feld="font-medium"
		/>
	</div>

	<!-- Die Antwort entscheidet über den Topf: Lernmittel bestellt das Land (Lernmittel-
	     freiheit), alles andere die Schülerbücherei aus Mitteln des Schulträgers. -->
	<Kaestchen bind:checked={istLernmittel} label="Lernmittel (Schulbuch der Lernmittelfreiheit)" />

	<div class="flex items-center justify-between gap-3">
		<div class="flex items-center gap-2">
			<label for="stagedMengeInput" class="text-xs font-medium text-slate-500">Menge</label>
			<Feld
				id="stagedMengeInput"
				type="number"
				min="1"
				bind:value={menge}
				feld="w-16 text-center font-bold"
			/>
		</div>
		<Kaestchen bind:checked={generateBarcodes} label="Barcodes generieren" />
	</div>

	<div class="flex items-center gap-2">
		<Button variant="ghost" onclick={onDone}>Abbrechen</Button>
		<Button onclick={uebernehmen} class="flex-1">In den Warenkorb</Button>
	</div>
</div>
