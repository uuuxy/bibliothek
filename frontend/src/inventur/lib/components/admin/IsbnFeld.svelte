<!--
  IsbnFeld.svelte
  Die ISBN einer Buchmaske: getippt oder über die Kamera gelesen. Beide Wege fragen auf
  dieselbe Weise, erst den eigenen Katalog, dann die Katalogdienste.
-->
<script>
	import { apiFetch } from '../../../../lib/apiFetch.js';
	import Ladekreis from '../../../../lib/components/ui/Ladekreis.svelte';
	import StrichcodeScannerOverlay from '$lib/components/scanner/StrichcodeScannerOverlay.svelte';
	import { showToast } from '$lib/store.svelte.js';
	import { frageWennVergeben } from '../../buch_speichern.js';
	import { Camera, RefreshCw } from '@lucide/svelte';
	import Feld from '../../../../lib/components/ui/Feld.svelte';
	/** dnbVorschlag: Nach einer ISBN-Abfrage mit Treffer holt er die Schlagworte der DNB dazu. */
	let { formular = $bindable(), wirdGescannt = $bindable(), dnbVorschlag = undefined } = $props();

	const VON_HAND = 'Angaben von Hand eintragen oder später erneut abfragen.';
	const NICHTS_BEKANNT = {
		text: 'Zu dieser ISBN ist bei den Katalogdiensten nichts bekannt.',
		fehler: false
	};
	const GESCHEITERT = { text: `Die ISBN-Abfrage ist fehlgeschlagen. ${VON_HAND}`, fehler: true };
	/** Was eine Abfrage ohne Treffer bedeutet, je Status der Antwort.
	 * @type {Record<number, { text: string, fehler: boolean }>} */
	const AUSGAENGE = {
		400: { text: 'Die Nummer hat nicht die Form einer ISBN (10 oder 13 Stellen).', fehler: true },
		404: NICHTS_BEKANNT,
		502: { text: `Die Katalogdienste sind nicht erreichbar. ${VON_HAND}`, fehler: true }
	};

	let isLookupActive = $state(false);
	// Der Ausgang einer Abfrage ohne Treffer bleibt unter dem Feld stehen, bis neu gefragt
	// oder die ISBN geändert wird: Eine Meldung, die nach Sekunden verschwindet, sähe aus wie
	// „nichts bekannt", auch wenn nur der Dienst kurz fort war.
	/** @type {{ text: string, fehler: boolean } | null} */
	let ausgang = $state(null);
	// Ein Klick auf den Knopf verlässt zugleich das Feld: Der zweite Auslöser schließt sich
	// dem laufenden Ablauf an, statt Katalog und Katalogdienste doppelt zu fragen.
	let laeuft = false;
	let aufWunsch = false;

	/**
	 * Erst der eigene Katalog, dann die Katalogdienste: In einer neuen Maske führt eine ISBN,
	 * die schon ein Titel trägt, zu ihm, statt Angaben für ein zweites Buch zu laden.
	 * @param {boolean} wunsch — der Knopf lädt auch, wenn schon ein Titel dasteht
	 */
	async function nachschlagen(wunsch) {
		aufWunsch ||= wunsch;
		if (laeuft) return;
		laeuft = true;
		try {
			if (!formular.id && (await frageWennVergeben(formular.isbn))) return;
			if (aufWunsch || !formular.title) await holeMetadaten();
		} finally {
			laeuft = false;
			aufWunsch = false;
		}
	}

	function aufKnopf() {
		if (!formular.isbn) {
			showToast('Bitte zuerst eine ISBN eingeben.', 'error');
			return;
		}
		return nachschlagen(true);
	}

	function beiVerlassen() {
		if (formular.isbn) return nachschlagen(false);
	}

	/** Die Kamera trägt die ISBN ein, ohne dass jemand das Feld verlässt. @param {string} code */
	function beiKameraScan(code) {
		formular.isbn = code;
		return nachschlagen(false);
	}

	/** @param {{ text: string, fehler: boolean }} ergebnis */
	function melde(ergebnis) {
		ausgang = ergebnis;
		showToast(ergebnis.text, ergebnis.fehler ? 'error' : 'info');
	}

	async function holeMetadaten() {
		isLookupActive = true;
		ausgang = null;
		try {
			const antwort = await apiFetch(`/api/lookup/${formular.isbn}`);
			if (!antwort.ok) {
				melde(AUSGAENGE[antwort.status] ?? GESCHEITERT);
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
			const klasse = parseInt(daten.grade);
			if (!Number.isNaN(klasse)) formular.gradeLevel = klasse;
			if (!daten.title) {
				melde(NICHTS_BEKANNT);
				return;
			}
			dnbVorschlag?.lade(formular.isbn);
			showToast(`Metadaten übernommen: ${daten.title}`, 'success');
		} catch (fehler) {
			console.error('Fehler beim Nachschlagen der ISBN', fehler);
			melde(GESCHEITERT);
		} finally {
			isLookupActive = false;
		}
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
	onblur={beiVerlassen}
	oninput={() => (ausgang = null)}
	hint={ausgang?.text ?? ''}
	ungueltig={!!ausgang?.fehler}
	feld="pr-20"
>
	{#snippet nachlaufend()}
		<button
			type="button"
			onclick={aufKnopf}
			disabled={isLookupActive}
			class="rounded-full p-0.5 text-on-surface-variant transition-colors hover:text-primary disabled:opacity-50"
			title="Daten aus dem Internet aktualisieren"
			aria-label="Daten aus dem Internet aktualisieren"
		>
			{#if isLookupActive}
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
