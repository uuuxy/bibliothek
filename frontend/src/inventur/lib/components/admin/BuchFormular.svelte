<script>
	import { apiFetch } from '../../../../lib/apiFetch.js';
	import { fade } from 'svelte/transition';
	import StrichcodeScannerOverlay from '$lib/components/scanner/StrichcodeScannerOverlay.svelte';
	import BuchCoverUpload from './BuchCoverUpload.svelte';
	import BuchEingabefelder from './BuchEingabefelder.svelte';
	import BuchExemplareListe from './BuchExemplareListe.svelte';
	import Button from '../../../../lib/components/ui/Button.svelte';
	import { BookOpen, Printer, Trash2, X } from '@lucide/svelte';

	/** onDelete kommt nur mit dem Recht delete_books (admin/+page) — ohne Recht gibt es den Knopf nicht. */
	let {
		formular = $bindable(),
		onClose,
		onSave,
		onCoverUpload,
		onAssignClass,
		onDelete = undefined
	} = $props();

	let wirdGescannt = $state(false);

	// Neuanlage eines Bibliotheksbuchs ohne Signatur ist gesperrt — die Signatur
	// muss aufs Rücken-Etikett. Lernmittel tragen keins (Migration 093). Die DNB
	// liefert höchstens einen Kategorie-Vorschlag (Kinder-/Jugendbuch → "BIB …"),
	// die Entscheidung bleibt beim Menschen. Altbestand (formular.id) bleibt
	// speicherbar, damit leere Littera-Importe pflegbar sind.
	const speichernGesperrt = $derived(
		!formular.id && !formular.istLernmittel && !(formular.signatur ?? '').trim()
	);

	/** @param {string} code */
	async function handleScan(code) {
		formular.isbn = code;
		if (!formular.title) {
			try {
				const res = await apiFetch(`/api/lookup/${code}`);
				if (res.ok) {
					const json = await res.json();
					const data = json.data;
					if (data.title) formular.title = data.title;
					if (data.author) formular.author = data.author;
					if (data.verlag) formular.verlag = data.verlag;
					if (data.jahr)
						formular.erscheinungsjahr = parseInt(data.jahr) || formular.erscheinungsjahr;
					if (data.coverUrl) formular.coverUrl = data.coverUrl;
					if (data.subject) formular.subject = data.subject;
					if (data.grade) formular.gradeLevel = parseInt(data.grade) || formular.gradeLevel;
					// DNB-Altersstufe → Signatur-Vorschlag "BIB {Kategorie}",
					// nur solange das Pflichtfeld noch leer ist.
					if (data.bibKategorie && !(formular.signatur ?? '').trim()) {
						formular.signatur = `BIB ${data.bibKategorie}`;
					}
				}
			} catch (e) {
				console.error('Lookup failed', e);
			}
		}
	}
</script>

<StrichcodeScannerOverlay bind:isScanning={wirdGescannt} onScan={handleScan} />

<!-- Full width block -->
<div class="flex flex-col w-full my-4" transition:fade={{ duration: 200 }}>
	<!-- Drawer Header -->
	<div
		class="px-6 py-5 border-b border-slate-100 flex items-center justify-between bg-white sticky top-0 z-10 rounded-t-3xl"
	>
		<h2 class="text-xl font-bold text-slate-900">
			{formular.id ? 'Buch bearbeiten' : 'Neues Buch'}
		</h2>
		<button
			onclick={onClose}
			class="p-2 hover:bg-slate-100 rounded-full text-slate-500 transition"
			aria-label="Schließen"
		>
			<X class="w-6 h-6" aria-hidden="true" />
		</button>
	</div>

	<!-- Zwei Spalten wie eine Play-Store-Detailseite (Peter, 02.09.2026): Felder links,
	     Cover rechts und beim Scrollen stehend. Vorher saß das Cover klein oben in der
	     Mitte und jedes Feld lief über die volle Breite — zwei Meter Formular. -->
	<div class="flex-1 p-6 lg:grid lg:grid-cols-[minmax(0,1fr)_14rem] lg:gap-10">
		<div class="space-y-8">
			<BuchEingabefelder bind:formular bind:wirdGescannt />
			{#if formular.id}
				<BuchExemplareListe bind:formular />
			{/if}
		</div>
		<aside class="mt-8 lg:mt-0 lg:sticky lg:top-24 lg:self-start">
			<BuchCoverUpload bind:formular {onCoverUpload} />
		</aside>
	</div>

	<!-- Drawer Footer -->
	<div
		class="p-6 border-t border-slate-100 bg-slate-50 flex justify-end gap-3 sticky bottom-0 rounded-b-2xl"
	>
		{#if formular.id && onDelete}
			<!-- Löschen gehörte bisher nur der Mehrfachauswahl in der Titel-Liste; wer schon in
			     der Maske steht, musste zurück. Der Server verweigert bei verliehenen Exemplaren. -->
			<Button
				variant="ghost"
				size="lg"
				onclick={onDelete}
				class="text-error"
				title="Diesen Titel mit allen Exemplaren löschen"
			>
				<Trash2 class="w-4 h-4" aria-hidden="true" />
				Titel löschen
			</Button>
		{/if}
		{#if formular.id}
			<Button
				variant="secondary"
				size="lg"
				onclick={() => window.open(`/api/buecher/titel/${formular.id}/etiketten`, '_blank')}
				class="px-5"
				title="A4 Zweckform Etikettenbogen für dieses Buch generieren"
			>
				<Printer class="w-4 h-4" aria-hidden="true" />
				Barcodes drucken
			</Button>
			<Button
				variant="secondary"
				size="lg"
				onclick={onAssignClass}
				class="mr-auto px-5"
				title="Diesen Titel dem Klassensatz einer Schulklasse hinzufügen"
			>
				<BookOpen class="w-4 h-4" aria-hidden="true" />
				Zum Klassensatz hinzufügen
			</Button>
		{/if}

		<Button variant="ghost" size="lg" onclick={onClose} class="px-5">Abbrechen</Button>
		<Button
			size="lg"
			onclick={onSave}
			disabled={speichernGesperrt}
			title={speichernGesperrt ? 'Signatur eintragen, um zu speichern' : undefined}
			class="px-5 bg-emerald-600 hover:bg-emerald-700"
		>
			Speichern
		</Button>
	</div>
</div>
