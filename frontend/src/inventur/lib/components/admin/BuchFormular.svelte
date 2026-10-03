<script>
	import { fade } from 'svelte/transition';
	import BuchCoverUpload from './BuchCoverUpload.svelte';
	import BuchEingabefelder from './BuchEingabefelder.svelte';
	import BuchExemplareListe from './BuchExemplareListe.svelte';
	import { signaturFehlt } from './buch_form_optionen.js';
	import Button from '../../../../lib/components/ui/Button.svelte';
	import { erzeugeDnbSchlagwortVorschlag } from '../../../../lib/utils/dnbSchlagwortVorschlag.svelte.js';
	import { erzeugeIsbnAbfrage } from './isbnAbfrage.svelte.js';
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

	// Der Schlagwort-Vorschlag der DNB: Die ISBN-Abfrage holt ihn nach einem Treffer dazu; bei
	// einem Titel, den es schon gibt, der Knopf unter den Schlagworten.
	const dnbVorschlag = erzeugeDnbSchlagwortVorschlag();
	const abfrage = erzeugeIsbnAbfrage(
		() => formular,
		() => dnbVorschlag
	);

	// Ein Klick auf „Speichern" verlässt das ISBN-Feld und stößt dessen Abfrage an. Gespeichert
	// wird erst mit ihren Angaben: Der Server trägt nichts nach. Hat sie nach einem
	// vorhandenen Titel gefragt, entscheidet die Antwort darauf und nicht dieser Klick.
	async function speichern() {
		if (await abfrage.ruht()) return;
		// Der Knopf bleibt bedienbar: Fehlt die Pflicht-Signatur, führt der Klick zum Feld, das
		// den Grund nennt. Die Mitte des Fensters, weil der Kopf der Maske oben stehen bleibt.
		if (signaturFehlt(formular)) {
			const feld = document.getElementById('buch-signatur');
			feld?.scrollIntoView({ block: 'center', inline: 'nearest' });
			feld?.focus({ preventScroll: true });
			return;
		}
		onSave();
	}
</script>

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
		<Button onclick={speichern} class="ml-4 shrink-0 px-5">Speichern</Button>
	</div>

	<!-- Zwei Spalten wie eine Play-Store-Detailseite: Felder und Exemplare links, Cover und die
	     Aktionen zum Titel rechts und beim Scrollen stehend. Die Spalte steht im Quelltext vor
	     den Exemplaren: In einem schmalen Fenster folgt sie so auf die Felder und nicht erst
	     auf das letzte Exemplar. -->
	<div class="flex-1 p-6 lg:grid lg:grid-cols-[minmax(0,1fr)_16rem] lg:gap-x-10">
		<div class="space-y-8 lg:col-start-1">
			<BuchEingabefelder bind:formular bind:wirdGescannt {dnbVorschlag} {abfrage} />
		</div>
		<aside
			class="mx-auto mt-8 w-full max-w-64 lg:col-start-2 lg:row-span-2 lg:row-start-1 lg:mt-0 lg:sticky lg:top-20 lg:self-start"
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
		<!-- Auch am neuen Titel: Unter „Exemplare" steht die Zahl, mit der er angelegt wird. -->
		<div class="lg:col-start-1">
			<BuchExemplareListe bind:formular />
		</div>
	</div>
</div>
