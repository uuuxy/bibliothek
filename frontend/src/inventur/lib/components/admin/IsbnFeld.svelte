<!--
  IsbnFeld.svelte
  Die ISBN einer Buchmaske: getippt, mit dem Handscanner oder über die Kamera gelesen. Alle
  Wege fragen über dieselbe Abfrage (isbnAbfrage.svelte.js).
-->
<script>
	import { onMount } from 'svelte';
	import Ladekreis from '../../../../lib/components/ui/Ladekreis.svelte';
	import StrichcodeScannerOverlay from '$lib/components/scanner/StrichcodeScannerOverlay.svelte';
	import { showToast } from '$lib/store.svelte.js';
	import { Camera, RefreshCw } from '@lucide/svelte';
	import Feld from '../../../../lib/components/ui/Feld.svelte';
	/** abfrage: die ISBN-Abfrage der Maske (erzeugeIsbnAbfrage); die Maske wartet vor dem
	 *  Speichern auf sie. */
	let { formular = $bindable(), wirdGescannt = $bindable(), abfrage } = $props();

	/** @type {HTMLInputElement | undefined} */
	let eingabe = $state();
	// Die ISBN, zu der die Eingabetaste gefragt hat: Das Verlassen des Feldes danach fragt
	// mit ihr nicht noch einmal.
	let mitEingabetasteGefragt = '';

	// Ein Handscanner tippt blind in das Feld mit dem Fokus. In einer neuen Maske steht er
	// deshalb im ISBN-Feld — nicht hinter dem Kamera-Fenster und nicht bei einem vorhandenen
	// Titel, dessen ISBN der Scan überschriebe.
	onMount(() => {
		if (!formular.id && !wirdGescannt) eingabe?.focus();
	});
	// Führt die Frage nach der vergebenen ISBN zu ihrem Titel, zeigt dieselbe Maske ihn:
	// Der Fokus verlässt das Feld, sonst träfe der nächste Scan dessen ISBN.
	$effect(() => {
		if (formular.id && document.activeElement === eingabe) eingabe?.blur();
	});

	function aufKnopf() {
		if (!formular.isbn) {
			showToast('Bitte zuerst eine ISBN eingeben.', 'error');
			return;
		}
		return abfrage.nachschlagen(true);
	}

	function beiVerlassen() {
		const schonGefragt = mitEingabetasteGefragt === formular.isbn;
		mitEingabetasteGefragt = '';
		if (formular.isbn && !schonGefragt) return abfrage.nachschlagen(false);
	}

	/** Die Eingabetaste beendet einen Scan: Sie fragt wie das Verlassen des Feldes. Die ISBN
	 * ist danach markiert, der nächste Scan ersetzt sie.
	 * @param {KeyboardEvent} ereignis */
	function beiTaste(ereignis) {
		if (ereignis.key !== 'Enter' || !formular.isbn) return;
		ereignis.preventDefault();
		mitEingabetasteGefragt = formular.isbn;
		eingabe?.select();
		return abfrage.nachschlagen(false);
	}

	/** Die Kamera trägt die ISBN ein, ohne dass jemand das Feld verlässt. @param {string} code */
	function beiKameraScan(code) {
		formular.isbn = code;
		return abfrage.nachschlagen(false);
	}
</script>

<StrichcodeScannerOverlay bind:isScanning={wirdGescannt} onScan={beiKameraScan} />

<!-- Die zwei Symbole sind nachlaufende Icon-Buttons des Feldes: on-surface-variant wie die
     Kamera im Suchfeld (ui/Suchfeld), beim Zeigen primary. Die Rückmeldung als Fläche kommt
     aus dem State-Layer aller Knöpfe. -->
<Feld
	id="buch-isbn"
	label={formular.medientyp === 'CD' || formular.medientyp === 'DVD' ? 'EAN' : 'ISBN'}
	bind:value={formular.isbn}
	bind:element={eingabe}
	onblur={beiVerlassen}
	onkeydown={beiTaste}
	oninput={abfrage.vergissAusgang}
	hint={abfrage.ausgang?.text ?? ''}
	ungueltig={!!abfrage.ausgang?.fehler}
	feld="pr-20"
>
	{#snippet nachlaufend()}
		<button
			type="button"
			onclick={aufKnopf}
			disabled={abfrage.aktiv}
			class="rounded-full p-0.5 text-on-surface-variant transition-colors hover:text-primary disabled:opacity-50"
			title="Daten aus dem Internet aktualisieren"
			aria-label="Daten aus dem Internet aktualisieren"
		>
			{#if abfrage.aktiv}
				<Ladekreis size="md" />
			{:else}
				<RefreshCw class="h-5 w-5" aria-hidden="true" />
			{/if}
		</button>
		<button
			type="button"
			onclick={() => (wirdGescannt = true)}
			aria-pressed={wirdGescannt}
			class="rounded-full p-0.5 text-on-surface-variant transition-colors hover:text-primary"
			title="Scan ISBN"
			aria-label="Scan ISBN"
		>
			<Camera class="h-5 w-5" aria-hidden="true" />
		</button>
	{/snippet}
</Feld>
