<!-- @component StudentCreateModal — „Neuen Leser anlegen".

     Fragt ZUERST nach der Art (Peter, 16.09.2026): An ihr hängt, welche Angaben Pflicht
     sind. Ein Schüler braucht Klasse und Geburtsdatum — der LUSD-Import erkennt ihn nur
     daran wieder —, ein Kollege hat beides nicht und bekommt statt Stammdaten den
     Hinweis, dass hier KEIN Zugang entsteht.

     Die Regeln stehen doppelt: hier als verständliche Meldung vor dem Absenden, im
     Backend als Bedingung (pruefeLeserAngaben). Das Backend ist die Wahrheit; diese
     Seite erspart nur den Umweg über eine Fehlermeldung vom Server. -->
<script>
	import Modal from './Modal.svelte';
	import { apiClient } from './apiFetch.js';
	import Button from './components/ui/Button.svelte';
	import StudentFormFelder from './components/StudentFormFelder.svelte';
	import LeserArtWahl from './components/students/LeserArtWahl.svelte';
	import KollegiumFormFelder from './components/students/KollegiumFormFelder.svelte';
	import { leserArtText, istKollegium } from './leserArt.js';
	import { TriangleAlert } from '@lucide/svelte';

	let { open = false, klassen = [], onclose, onsuccess } = $props();

	let art = $state('schueler');
	let newVorname = $state('');
	let newNachname = $state('');
	let newKlasse = $state('');
	let customKlasseInput = $state(false);
	let newBarcode = $state('');
	let newGeburtsdatum = $state('');
	let createError = $state('');
	let duplicateConflict = $state('');
	let isSaving = $state(false);

	const kollege = $derived(istKollegium({ art }));

	// Formular zurücksetzen, sobald der Dialog aufgeht — samt Art: Wer zuletzt eine
	// Lehrkraft angelegt hat, legt beim nächsten Mal nicht ungewollt die zweite an.
	$effect(() => {
		if (open) {
			art = 'schueler';
			newVorname = '';
			newNachname = '';
			newKlasse = '';
			newBarcode = '';
			newGeburtsdatum = '';
			createError = '';
			duplicateConflict = '';
			customKlasseInput = false;
		}
	});

	/** Die Pflichtangaben — an die Art gepaart, wie im Backend. @returns {string} leer = in Ordnung */
	function fehlendeAngabe() {
		if (!newVorname.trim() || !newNachname.trim())
			return 'Vorname und Nachname sind Pflichtfelder.';
		if (kollege) return '';
		if (!newKlasse.trim()) return 'Klasse ist ein Pflichtfeld.';
		if (!newGeburtsdatum.trim())
			return 'Geburtsdatum fehlt. Ohne Geburtsdatum kann der LUSD-Import diesen Schüler später nicht wiedererkennen — er würde doppelt angelegt.';
		return '';
	}

	async function legeAn() {
		createError = fehlendeAngabe();
		duplicateConflict = '';
		if (createError) return;

		isSaving = true;
		try {
			const res = await apiClient.post('/api/schueler', {
				art,
				vorname: newVorname.trim(),
				nachname: newNachname.trim(),
				klasse: kollege ? '' : newKlasse.trim(),
				barcode_id: newBarcode.trim(),
				geburtsdatum: kollege ? null : newGeburtsdatum.trim()
			});
			if (res.ok) {
				onsuccess?.();
				return;
			}
			const rohtext = await res.text();
			let meldung = '';
			try {
				meldung = JSON.parse(rohtext).error || '';
			} catch {
				meldung = rohtext;
			}
			// Der Server erklärt den Doppeleintrag — seine Meldung nennt den Fall
			// (Namensgleichheit beim Kollegium, Name + Geburtsdatum beim Schüler). Bis zum
			// 16.09.2026 stand hier ein fester Satz über Schüler; bei einer Lehrkraft war
			// er schlicht falsch.
			if (res.status === 409) duplicateConflict = meldung || 'Diese Person gibt es bereits.';
			else createError = meldung || `Fehler beim Anlegen (${leserArtText(art)}).`;
		} catch (err) {
			createError = 'Netzwerkfehler beim Anlegen.';
			console.error(err);
		} finally {
			isSaving = false;
		}
	}
</script>

<Modal {open} onclose={() => onclose?.()} size="md">
	{#snippet header()}
		<h3 class="text-base font-bold text-slate-800">Neuen Leser anlegen</h3>
	{/snippet}
	<div class="p-6 space-y-4">
		<LeserArtWahl bind:art disabled={isSaving} />

		{#if duplicateConflict}
			<div
				class="p-4 bg-amber-50 border border-amber-200 rounded-xl flex items-start gap-3 text-sm font-semibold text-amber-800"
			>
				<TriangleAlert class="h-5 w-5 text-amber-500 shrink-0 mt-0.5" aria-hidden="true" />
				<p>{duplicateConflict}</p>
			</div>
		{/if}

		{#if createError}
			<div
				class="p-3 bg-rose-50 border border-rose-100 rounded-xl text-xs font-semibold text-rose-600"
			>
				{createError}
			</div>
		{/if}

		{#if kollege}
			<KollegiumFormFelder
				bind:vorname={newVorname}
				bind:nachname={newNachname}
				bind:barcode={newBarcode}
			/>
		{:else}
			<StudentFormFelder
				bind:vorname={newVorname}
				bind:nachname={newNachname}
				bind:geburtsdatum={newGeburtsdatum}
				bind:klasse={newKlasse}
				bind:barcode={newBarcode}
				bind:freieKlasse={customKlasseInput}
				{klassen}
			/>
		{/if}

		<div class="flex justify-end gap-3 pt-2 border-t border-slate-100">
			<Button variant="secondary" onclick={() => onclose?.()} disabled={isSaving}>Abbrechen</Button>
			<Button onclick={legeAn} disabled={isSaving}>
				{isSaving ? 'Speichern...' : 'Speichern'}
			</Button>
		</div>
	</div>
</Modal>
