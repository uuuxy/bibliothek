<!-- 
  IsbnFeld.svelte
  Verwaltet die Eingabe einer ISBN. Beinhaltet zwei Buttons: 
  Einen für die API-Abfrage der Metadaten und einen für den Barcode-Scanner.
-->
<script>
	import { apiFetch } from '../../../../lib/apiFetch.js';
	import { showToast } from '$lib/store.svelte.js';
	import { Camera, RefreshCw } from '@lucide/svelte';
	import Feld from '../../../../lib/components/ui/Feld.svelte';
	let { formular = $bindable(), wirdGescannt = $bindable() } = $props();

	let isLookupActive = $state(false);

	async function aktualisiereMetadaten() {
		if (!formular.isbn) {
			showToast('Bitte zuerst eine ISBN eingeben.', 'error');
			return;
		}
		isLookupActive = true;
		try {
			const antwort = await apiFetch(`/api/lookup/${formular.isbn}`);
			if (!antwort.ok) {
				showToast(
					antwort.status === 404
						? 'Keine Metadaten zu dieser ISBN gefunden.'
						: 'Metadaten-Abruf fehlgeschlagen (Server).',
					'error'
				);
				return;
			}
			const json = await antwort.json();
			const daten = json.data ?? {};
			if (daten.title) formular.title = daten.title;
			if (daten.author) formular.author = daten.author;
			if (daten.verlag) formular.verlag = daten.verlag;
			if (daten.jahr) formular.erscheinungsjahr = parseInt(daten.jahr) || formular.erscheinungsjahr;
			if (daten.coverUrl) formular.coverUrl = daten.coverUrl;
			if (daten.subject) formular.subject = daten.subject;
			if (daten.grade) {
				const parsedGrade = parseInt(daten.grade);
				if (!Number.isNaN(parsedGrade)) {
					formular.gradeLevel = parsedGrade;
				}
			}
			// DNB-Altersstufe ("Kinderbuch", "Jugendbücher ab 12 Jahre") →
			// Signatur-Vorschlag "BIB {Kategorie}". Nur solange das Feld leer
			// ist — eine vorhandene Signatur wird nie überschrieben.
			if (daten.bibKategorie && !(formular.signatur ?? '').trim()) {
				formular.signatur = `BIB ${daten.bibKategorie}`;
			}
			// Klares Feedback statt stillem "nichts passiert"
			showToast(
				daten.title
					? `Metadaten übernommen: ${daten.title}`
					: 'Keine verwertbaren Metadaten gefunden.',
				daten.title ? 'success' : 'error'
			);
		} catch (fehler) {
			console.error('Fehler beim Nachschlagen der ISBN', fehler);
			showToast('Netzwerkfehler beim ISBN-Nachschlagen.', 'error');
		} finally {
			isLookupActive = false;
		}
	}

	async function beiVerlassen() {
		if (formular.isbn && !formular.title) {
			await aktualisiereMetadaten();
		}
	}
</script>

<!-- Eigene Subgrid-Hülle statt `label=` am Feld: Die zwei Knöpfe liegen IM Feld und
     brauchen dafür einen relative-Kasten um die Feldzeile. Drei Zeilen wie beim Feld
     mit label, damit der Autor daneben auf gleicher Höhe steht. -->
<div class="row-span-3 grid grid-rows-subgrid gap-y-1.5">
	<label for="buch-isbn" class="text-sm font-medium text-on-surface-variant">
		{formular.medientyp === 'CD' || formular.medientyp === 'DVD' ? 'EAN' : 'ISBN'}
	</label>
	<div class="relative">
		<Feld id="buch-isbn" bind:value={formular.isbn} onblur={beiVerlassen} feld="pr-20" />
		<button
			type="button"
			onclick={aktualisiereMetadaten}
			disabled={isLookupActive}
			class="absolute right-10 top-1/2 -translate-y-1/2 text-slate-400 hover:text-emerald-600 p-0.5 rounded-full hover:bg-slate-200 transition-colors disabled:opacity-50"
			title="Daten aus dem Internet aktualisieren"
			aria-label="Daten aus dem Internet aktualisieren"
		>
			{#if isLookupActive}
				<div
					class="w-5 h-5 border-2 border-slate-300 border-t-emerald-600 rounded-full animate-spin"
				></div>
			{:else}
				<RefreshCw class="w-5 h-5" aria-hidden="true" />
			{/if}
		</button>
		<button
			type="button"
			onclick={() => (wirdGescannt = true)}
			aria-pressed={wirdGescannt}
			class="absolute right-2 top-1/2 -translate-y-1/2 text-slate-400 hover:text-emerald-600 p-0.5 rounded-full hover:bg-slate-200 transition-colors"
			title="Scan ISBN"
			aria-label="Scan ISBN"
		>
			<Camera class="w-5 h-5" aria-hidden="true" />
		</button>
	</div>
</div>
