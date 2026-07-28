<script>
	import { apiFetch } from '../../../../lib/apiFetch.js';
	import { fade } from 'svelte/transition';
	import StrichcodeScannerOverlay from '$lib/components/scanner/StrichcodeScannerOverlay.svelte';
	import BuchCoverUpload from './BuchCoverUpload.svelte';
	import BuchEingabefelder from './BuchEingabefelder.svelte';
	import BuchExemplareListe from './BuchExemplareListe.svelte';
	import Button from '../../../../lib/components/ui/Button.svelte';

	let { formular = $bindable(), onClose, onSave, onCoverUpload, onAssignClass } = $props();

	let wirdGescannt = $state(false);

	// Neuanlage ohne Signatur ist gesperrt — die Signatur muss aufs
	// Rücken-Etikett. Die DNB liefert höchstens einen Kategorie-Vorschlag
	// (Kinder-/Jugendbuch → "BIB …"), die Entscheidung bleibt beim Menschen.
	// Altbestand (formular.id) bleibt speicherbar, damit leere
	// Littera-Importe pflegbar sind.
	const speichernGesperrt = $derived(!formular.id && !(formular.signatur ?? '').trim());

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
		class="px-6 py-5 border-b border-slate-100 flex items-center justify-between bg-white sticky top-0 z-10 rounded-t-2xl"
	>
		<h2 class="text-xl font-bold text-slate-900">
			{formular.id ? 'Buch bearbeiten' : 'Neues Buch'}
		</h2>
		<button
			onclick={onClose}
			class="p-2 hover:bg-slate-100 rounded-full text-slate-500 transition"
			aria-label="Schließen"
		>
			<svg class="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
				<path
					stroke-linecap="round"
					stroke-linejoin="round"
					stroke-width="2"
					d="M6 18L18 6M6 6l12 12"
				/>
			</svg>
		</button>
	</div>

	<!-- Form Content -->
	<div class="p-6 space-y-8 flex-1">
		<BuchCoverUpload bind:formular {onCoverUpload} />
		<BuchEingabefelder bind:formular bind:wirdGescannt />
		{#if formular.id}
			<BuchExemplareListe bind:formular />
		{/if}
	</div>

	<!-- Drawer Footer -->
	<div
		class="p-6 border-t border-slate-100 bg-slate-50 flex justify-end gap-3 sticky bottom-0 rounded-b-2xl"
	>
		{#if formular.id}
			<Button
				variant="secondary"
				size="lg"
				onclick={() => window.open(`/api/buecher/titel/${formular.id}/etiketten`, '_blank')}
				class="px-5"
				title="A4 Zweckform Etikettenbogen für dieses Buch generieren"
			>
				<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"
					><path
						stroke-linecap="round"
						stroke-linejoin="round"
						stroke-width="2"
						d="M17 17h2a2 2 0 002-2v-4a2 2 0 00-2-2H5a2 2 0 00-2 2v4a2 2 0 002 2h2m2 4h6a2 2 0 002-2v-4a2 2 0 00-2-2H9a2 2 0 00-2 2v4a2 2 0 002 2zm8-12V5a2 2 0 00-2-2H9a2 2 0 00-2 2v4h10z"
					/></svg
				>
				Barcodes drucken
			</Button>
			<Button
				variant="secondary"
				size="lg"
				onclick={onAssignClass}
				class="mr-auto px-5"
				title="Dieses Buch einer Schulklasse zuweisen"
			>
				<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"
					><path
						stroke-linecap="round"
						stroke-linejoin="round"
						stroke-width="2"
						d="M12 6.253v13m0-13C10.832 5.477 9.246 5 7.5 5S4.168 5.477 3 6.253v13C4.168 18.477 5.754 18 7.5 18s3.332.477 4.5 1.253m0-13C13.168 5.477 14.754 5 16.5 5c1.747 0 3.332.477 4.5 1.253v13C19.832 18.477 18.247 18 16.5 18c-1.746 0-3.332.477-4.5 1.253"
					/></svg
				>
				Klasse zuweisen
			</Button>
		{/if}

		<Button variant="ghost" size="lg" onclick={onClose} class="px-5">Abbrechen</Button>
		<Button
			size="lg"
			onclick={onSave}
			disabled={speichernGesperrt}
			title={speichernGesperrt ? 'Signatur eintragen, um zu speichern' : undefined}
			class="px-5 bg-emerald-600 hover:bg-emerald-700 disabled:bg-slate-300"
		>
			Speichern
		</Button>
	</div>
</div>
