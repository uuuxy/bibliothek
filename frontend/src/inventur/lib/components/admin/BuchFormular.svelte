<script>
	import { apiFetch } from '../../../../lib/apiFetch.js';
	import { fade } from 'svelte/transition';
	import StrichcodeScannerOverlay from '$lib/components/scanner/StrichcodeScannerOverlay.svelte';
	import BuchCoverUpload from './BuchCoverUpload.svelte';
	import BuchEingabefelder from './BuchEingabefelder.svelte';
	import BuchExemplareListe from './BuchExemplareListe.svelte';
	import BuchAuflagen from './BuchAuflagen.svelte';
	import Button from '../../../../lib/components/ui/Button.svelte';
	import { erzeugeDnbSchlagwortVorschlag } from '../../../../lib/utils/dnbSchlagwortVorschlag.svelte.js';
	import { frageWennVergeben } from '../../buch_speichern.js';
	import { BookOpen, Printer, Trash2, X } from '@lucide/svelte';

	/**
	 * onDelete kommt nur mit dem Recht delete_books (admin/+page) — ohne Recht gibt es den Knopf nicht.
	 * wirdGescannt: Die Kamera der Maske; der Knopf „Scanner" der Titelliste öffnet sie eingeschaltet.
	 */
	let {
		formular = $bindable(),
		wirdGescannt = $bindable(false),
		onClose,
		onSave,
		onCoverUpload,
		onCoverNeuHolen,
		onAssignClass,
		onDelete = undefined
	} = $props();

	/** ISBN-Abfrage gescheitert — „nichts gefunden" und „Dienst weg" sehen sonst gleich aus. */
	let lookupFehler = $state(false);
	// Der Schlagwort-Vorschlag der DNB (entschieden am 30.09.2026): Beide ISBN-Abfragen — der
	// Scan hier und das Feld in IsbnFeld — holen ihn nach einem Treffer dazu; bei einem Titel,
	// den es schon gibt, der Knopf unter den Schlagworten.
	const dnbVorschlag = erzeugeDnbSchlagwortVorschlag();

	// Neuanlage eines Bibliotheksbuchs ohne Signatur ist gesperrt — die Signatur
	// muss aufs Rücken-Etikett. Lernmittel tragen keins (Migration 093). Die DNB
	// liefert Titel, Autor, Verlag, Jahr, Cover, Fach und Klasse als Vorschlag,
	// die Entscheidung bleibt beim Menschen. Altbestand (formular.id) bleibt
	// speicherbar, damit leere Littera-Importe pflegbar sind.
	const speichernGesperrt = $derived(
		!formular.id && !formular.istLernmittel && !(formular.signatur ?? '').trim()
	);

	/** @param {string} code */
	async function handleScan(code) {
		formular.isbn = code;
		lookupFehler = false;
		// Neue Maske: Trägt die gescannte ISBN schon ein Titel, führt die Frage zu ihm.
		if (!formular.id && (await frageWennVergeben(code))) return;
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
					if (data.title) dnbVorschlag.lade(code);
				} else {
					// Sweep „verschluckte Fehlantwort" (06.09.2026): Vorher blieb das Formular
					// nach dem Scan einfach leer — nicht zu unterscheiden von „zu dieser ISBN
					// ist nichts bekannt". Die Bibliothekarin tippt dann alles ab, obwohl der
					// Dienst nur kurz weg war.
					lookupFehler = true;
				}
			} catch (e) {
				lookupFehler = true;
				console.error('Lookup failed', e);
			}
		}
	}
</script>

<StrichcodeScannerOverlay bind:isScanning={wirdGescannt} onScan={handleScan} />

<div class="flex flex-col w-full my-4" transition:fade={{ duration: 200 }}>
	<!-- Der Kopf ist das einzige stehende Band der Maske: Schließen, Überschrift und die eine
	     Aktion, die die ganze Seite betrifft. Alles Weitere steht in der rechten Spalte, damit
	     die Höhe den Feldern bleibt. „Speichern" folgt auf die Überschrift und steht nicht am
	     rechten Rand: Dort erscheinen die Meldungen (ToastContainer) und lägen über dem Knopf. -->
	<div
		class="h-14 px-4 border-b border-outline-variant flex items-center gap-2 bg-surface-container-lowest sticky top-0 z-10"
	>
		<!-- M3 Icon button, Standard: Symbol in on-surface-variant, die Rückmeldung beim
		     Zeigen kommt aus dem State-Layer aller Knöpfe (komponenten.css). -->
		<button onclick={onClose} class="icon-btn p-2 text-on-surface-variant" aria-label="Schließen">
			<X class="w-6 h-6" aria-hidden="true" />
		</button>
		<h2 class="min-w-0 truncate text-xl font-bold text-on-surface">
			{formular.id ? 'Buch bearbeiten' : 'Neues Buch'}
		</h2>
		<Button
			onclick={onSave}
			disabled={speichernGesperrt}
			title={speichernGesperrt ? 'Signatur eintragen, um zu speichern' : undefined}
			class="ml-4 shrink-0 px-5"
		>
			Speichern
		</Button>
	</div>

	<!-- Zwei Spalten wie eine Play-Store-Detailseite: Felder und Listen links, Cover und die
	     Aktionen zum Titel rechts und beim Scrollen stehend. Die Spalte steht im Quelltext vor
	     den Listen: In einem schmalen Fenster folgt sie so auf die Felder und nicht erst auf
	     das letzte Exemplar. -->
	<div class="flex-1 p-6 lg:grid lg:grid-cols-[minmax(0,1fr)_16rem] lg:gap-x-10">
		<div class="space-y-8 lg:col-start-1">
			{#if lookupFehler}
				<p class="mb-3 text-sm font-semibold text-error" role="alert">
					Die ISBN-Abfrage ist fehlgeschlagen — die Felder bleiben leer. Das heißt NICHT, dass zu
					dieser ISBN nichts bekannt ist.
				</p>
			{/if}

			<BuchEingabefelder bind:formular bind:wirdGescannt {dnbVorschlag} />
		</div>
		<aside
			class="mx-auto mt-8 w-full max-w-64 lg:col-start-2 lg:row-start-1 lg:mt-0 lg:sticky lg:top-20 lg:self-start {formular.id
				? 'lg:row-span-2'
				: ''}"
		>
			<BuchCoverUpload bind:formular {onCoverUpload} {onCoverNeuHolen} />
			{#if formular.id}
				<div class="mt-4 flex flex-col gap-2 border-t border-outline-variant pt-4">
					<Button
						variant="secondary"
						onclick={() => window.open(`/api/buecher/titel/${formular.id}/etiketten`, '_blank')}
						class="w-full"
						title="A4 Zweckform Etikettenbogen für dieses Buch generieren"
					>
						<Printer class="w-4 h-4" aria-hidden="true" />
						Barcodes drucken
					</Button>
					<Button
						variant="secondary"
						onclick={onAssignClass}
						class="w-full"
						title="Diesen Titel dem Klassensatz einer Schulklasse hinzufügen"
					>
						<BookOpen class="w-4 h-4" aria-hidden="true" />
						Zum Klassensatz hinzufügen
					</Button>
					{#if onDelete}
						<!-- Der Server verweigert das Löschen bei verliehenen Exemplaren. -->
						<Button
							variant="ghost"
							onclick={onDelete}
							class="w-full text-error"
							title="Diesen Titel mit allen Exemplaren löschen"
						>
							<Trash2 class="w-4 h-4" aria-hidden="true" />
							Titel löschen
						</Button>
					{/if}
				</div>
			{/if}
		</aside>
		{#if formular.id}
			<div class="space-y-8 lg:col-start-1">
				<BuchAuflagen {formular} />
				<BuchExemplareListe bind:formular />
			</div>
		{/if}
	</div>
</div>
