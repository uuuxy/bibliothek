<script>
	import { appState } from '$lib/store.svelte.js';
	import Button from '../../../../lib/components/ui/Button.svelte';
	import Suchpille from '../../../../lib/components/ui/Suchpille.svelte';
	import Segmente from '../../../../lib/components/ui/Segmente.svelte';
	import Ladekreis from '../../../../lib/components/ui/Ladekreis.svelte';
	import { BookOpen, Camera, Plus, RefreshCw, Trash } from '@lucide/svelte';

	/**
	 * @type {{
	 *   titelAnzahl: number,
	 *   anzahlGewaehlt: number,
	 *   loescht?: boolean,
	 *   onDelete: () => void,
	 *   onAssignClass: () => void,
	 *   onScan: () => void,
	 *   onCreateNew: () => void,
	 *   onRetryCovers: () => void
	 * }}
	 */
	let {
		titelAnzahl,
		anzahlGewaehlt,
		loescht = false,
		onDelete,
		onAssignClass,
		onScan,
		onCreateNew,
		onRetryCovers
	} = $props();
</script>

<div class="flex flex-col gap-4 border-b border-outline-variant px-4 py-4 md:px-6">
	<Suchpille
		kamera
		autofokus
		id="titelverwaltung-suchfeld"
		bind:wert={appState.searchQuery}
		platzhalter="Titel, Autor oder ISBN eingeben …"
		etikett="Bücher durchsuchen"
	/>

	<div class="flex flex-wrap items-center gap-2 sm:gap-3">
		<h2 class="shrink-0 text-lg font-bold text-on-surface">
			Bücher ({titelAnzahl})
		</h2>
		<!-- Zwei Sichten derselben Liste (M3: „Segmented buttons help people select options,
		     switch views, or sort elements"): der Katalog und die Aufräumsicht mit den Titeln,
		     die er nicht zeigt. -->
		<Segmente
			klasse="mr-auto"
			etikett="Sicht der Titelliste"
			optionen={[
				{ wert: 'mit', text: 'Mit Exemplaren' },
				{ wert: 'ohne', text: 'Ohne Exemplare' }
			]}
			wert={appState.bestandsAnsicht}
			onwahl={(wert) => (appState.bestandsAnsicht = wert === 'ohne' ? 'ohne' : 'mit')}
		/>
		{#if anzahlGewaehlt > 0}
			<Button variant="secondary" onclick={onAssignClass}>
				<BookOpen class="w-4 h-4" aria-hidden="true" />
				Zum Klassensatz hinzufügen ({anzahlGewaehlt})
			</Button>
			<Button variant="danger" onclick={onDelete} disabled={loescht}>
				{#if loescht}
					<Ladekreis size="sm" farbe="aktuell" />
				{:else}
					<Trash class="w-4 h-4" aria-hidden="true" />
				{/if}
				Löschen ({anzahlGewaehlt})
			</Button>
		{/if}

		<Button variant="secondary" onclick={onRetryCovers} class="flex-1 sm:flex-none">
			<RefreshCw class="w-4 h-4" aria-hidden="true" />
			Cover lokalisieren
		</Button>

		<!-- Öffnet die Maske „Neues Buch" mit eingeschalteter Kamera; das Symbol ist dasselbe wie
		     an der Kamera im ISBN-Feld und in der Suchpille. -->
		<Button variant="secondary" onclick={onScan} class="flex-1 sm:flex-none">
			<Camera class="w-4 h-4" aria-hidden="true" />
			Scanner
		</Button>

		<Button onclick={onCreateNew} class="mt-2 w-full sm:mt-0 sm:w-auto">
			<Plus class="w-4 h-4" aria-hidden="true" />
			Neues Buch
		</Button>
	</div>
</div>
