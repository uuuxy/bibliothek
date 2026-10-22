<script>
	import { fade } from 'svelte/transition';
	import BuchCoverUpload from './BuchCoverUpload.svelte';
	import BuchEingabefelder from './BuchEingabefelder.svelte';
	import BuchExemplareListe from './BuchExemplareListe.svelte';
	import { signaturFehlt } from './buchFormularRegeln.js';
	import Button from '../../../../lib/components/ui/Button.svelte';
	import { erzeugeDnbSchlagwortVorschlag } from '../../../../lib/utils/dnbSchlagwortVorschlag.svelte.js';
	import { erzeugeIsbnAbfrage } from './isbnAbfrage.svelte.js';
	import { enterSpeichert } from '../../../../lib/scanErkennung.js';
	import { etikettenStore } from '../../../../lib/stores/etiketten.svelte.js';
	import { uiStore } from '../../../../lib/stores/uiStore.svelte.js';
	import { authStore } from '../../../../lib/stores/authStore.svelte.js';
	import { hatRecht } from '../../../../lib/menue.js';
	import { rechteDesReiters } from '../../../../lib/druckCenterReiter.js';
	import { BookOpen, Printer, Trash, X } from '@lucide/svelte';

	/**
	 * onDelete kommt nur mit dem Recht delete_books (TitelVerwaltung) — ohne Recht gibt es den Knopf nicht.
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

	// Der Schlagwort-Vorschlag der DNB: Die ISBN-Abfrage holt ihn nach einem Treffer dazu; bei
	// einem Titel, den es schon gibt, der Knopf unter den Schlagworten.
	const dnbVorschlag = erzeugeDnbSchlagwortVorschlag();
	const abfrage = erzeugeIsbnAbfrage(
		() => formular,
		() => dnbVorschlag
	);

	// Die Etiketten des Titels entstehen im Druck-Center wie die eines Exemplars in der Buchakte.
	// Wer den Reiter „Buch-Etiketten" nicht öffnen darf, landete auf einem anderen.
	const darfEtiketten = $derived(
		rechteDesReiters('labels').every((recht) => hatRecht(authStore.currentUser, recht))
	);

	/** Öffnet das Druck-Center mit diesem Titel; seine Exemplare sind dort angehakt. */
	function etikettenDrucken() {
		etikettenStore.leereDruckauftrag();
		etikettenStore.waehleTitel({ id: formular.id, titel: formular.title, autor: formular.author });
		uiStore.requestedDruckCenterTab = 'labels';
		uiStore.activeTab = 'druck-center';
	}

	// Pflicht sind der Titel und am neuen Bibliotheksbuch die Signatur, die ISBN nicht:
	// Zeitschriften, Spiele und alte Bücher tragen keine. Ein Pflichtfeld zeigt seinen Fehler
	// erst, wenn ein Klick auf „Speichern" es leer vorgefunden hat.
	let verlangt = $state(false);

	/** Die Mitte des Fensters, weil der Kopf der Maske oben stehen bleibt. @param {string} id */
	function fuehreZu(id) {
		const feld = document.getElementById(id);
		feld?.scrollIntoView({ block: 'center', inline: 'nearest' });
		feld?.focus({ preventScroll: true });
	}

	// Ein Klick auf „Speichern" verlässt das ISBN-Feld und stößt dessen Abfrage an. Gespeichert
	// wird erst mit ihren Angaben: Der Server trägt nichts nach. Hat sie nach einem
	// vorhandenen Titel gefragt, entscheidet die Antwort darauf und nicht dieser Klick.
	async function speichern() {
		if (await abfrage.ruht()) return;
		// Der Knopf bleibt bedienbar: Fehlt ein Pflichtfeld, führt der Klick zum Feld, das den
		// Grund nennt.
		verlangt = true;
		if (!(formular.title ?? '').trim()) return fuehreZu('buch-titel');
		if (signaturFehlt(formular)) return fuehreZu('buch-signatur');
		onSave();
	}
</script>

<!-- Enter speichert wie „Speichern", außer im ISBN-Feld, im Feld der Schlagworte und am Ende
     eines Scans (enterSpeichert). -->
<div
	class="flex flex-col w-full my-4"
	transition:fade={{ duration: 200 }}
	use:enterSpeichert={speichern}
>
	<!-- Der Kopf ist das einzige stehende Band der Maske: Schließen, Überschrift und die eine
	     Aktion, die die ganze Seite betrifft. Alles Weitere steht in der rechten Spalte, damit
	     die Höhe den Feldern bleibt. „Speichern" steht am rechten Rand (Material 3, Dialogs:
	     „Buttons are aligned to the trailing edge of the dialog"); die Meldungen erscheinen
	     unten und liegen nicht über ihm. -->
	<div
		class="h-14 px-4 border-b border-outline-variant flex items-center gap-2 bg-surface-container-lowest sticky top-0 z-10"
	>
		<!-- M3 Icon button, Standard: Symbol in on-surface-variant, die Rückmeldung beim
		     Zeigen kommt aus dem State-Layer aller Knöpfe (komponenten.css). -->
		<button
			onclick={onClose}
			class="icon-btn p-2 text-on-surface-variant"
			aria-label="Schließen"
			data-tip="Schließen"
		>
			<X class="w-6 h-6" aria-hidden="true" />
		</button>
		<h2 class="min-w-0 truncate text-xl font-bold text-on-surface">
			{formular.id ? 'Buch bearbeiten' : 'Neues Buch'}
		</h2>
		<Button onclick={speichern} class="ml-auto shrink-0 px-5">Speichern</Button>
	</div>

	<!-- Zwei Spalten wie eine Play-Store-Detailseite: Felder und Exemplare links, Cover und die
	     Aktionen zum Titel rechts und beim Scrollen stehend. Die Spalte steht im Quelltext vor
	     den Exemplaren: In einem schmalen Fenster folgt sie so auf die Felder und nicht erst
	     auf das letzte Exemplar. -->
	<div class="flex-1 p-6 lg:grid lg:grid-cols-[minmax(0,1fr)_16rem] lg:gap-x-10">
		<!-- Die Felder enden an der Breite der Masken (max-w-maske): In einem breiten Fenster
		     laufen Textfelder sonst über die ganze Spalte. Die Exemplare darunter nutzen die Breite. -->
		<div class="max-w-maske space-y-8 lg:col-start-1">
			<BuchEingabefelder bind:formular bind:wirdGescannt {dnbVorschlag} {abfrage} {verlangt} />
		</div>
		<aside
			class="mx-auto mt-8 w-full max-w-64 lg:col-start-2 lg:row-span-2 lg:row-start-1 lg:mt-0 lg:sticky lg:top-20 lg:self-start"
		>
			<BuchCoverUpload bind:formular {onCoverUpload} {onCoverNeuHolen} />
			{#if formular.id}
				<div class="mt-4 flex flex-col gap-2 border-t border-outline-variant pt-4">
					{#if darfEtiketten}
						<Button
							variant="secondary"
							onclick={etikettenDrucken}
							class="w-full"
							title="Etiketten dieses Titels im Druck-Center drucken"
						>
							<Printer class="w-4 h-4" aria-hidden="true" />
							Barcodes drucken
						</Button>
					{/if}
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
							<Trash class="w-4 h-4" aria-hidden="true" />
							Titel löschen
						</Button>
					{/if}
				</div>
			{/if}
		</aside>
		<!-- Auch am neuen Titel: Unter „Exemplare" steht die Zahl, mit der er angelegt wird. -->
		<div class="lg:col-start-1">
			<BuchExemplareListe bind:formular />
		</div>
	</div>
</div>
