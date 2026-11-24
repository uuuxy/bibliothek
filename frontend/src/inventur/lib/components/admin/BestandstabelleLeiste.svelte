<script>
	import { appState } from '$lib/store.svelte.js';
	import Button from '../../../../lib/components/ui/Button.svelte';
	import Suchpille from '../../../../lib/components/ui/Suchpille.svelte';
	import Segmente from '../../../../lib/components/ui/Segmente.svelte';
	import Ladekreis from '../../../../lib/components/ui/Ladekreis.svelte';
	import { anzahlText } from '../../../../lib/utils/format.js';
	import Icon from '../../../../lib/components/ui/Icon.svelte';

	/**
	 * @type {{
	 *   titelAnzahl: number,
	 *   sichtLaedt?: boolean,
	 *   anzahlGewaehlt: number,
	 *   loescht?: boolean,
	 *   onDelete: () => void,
	 *   onAssignClass: () => void,
	 *   onCreateNew: () => void,
	 *   onRetryCovers: () => void
	 * }}
	 */
	let {
		titelAnzahl,
		sichtLaedt = false,
		anzahlGewaehlt,
		loescht = false,
		onDelete,
		onAssignClass,
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
		etikett="Titel durchsuchen"
	/>

	<div class="flex flex-wrap items-center gap-2 sm:gap-3">
		<!-- Solange die gewählte Sicht lädt, zählt die Zahl noch die alte Liste. Sie bleibt
		     unsichtbar stehen, damit die Segmente daneben nicht springen. -->
		<h2 class="shrink-0 text-lg text-on-surface {sichtLaedt ? 'invisible' : ''}">
			{anzahlText(titelAnzahl, 'Titel', 'Titel')}
		</h2>
		<!-- Zwei Sichten derselben Liste (M3: „Segmented buttons help people select options,
		     switch views, or sort elements"): der Katalog und die Aufräumsicht mit den Titeln,
		     die er nicht zeigt. -->
		<Segmente
			etikett="Sicht der Titelliste"
			optionen={[
				{ wert: 'mit', text: 'Mit Exemplaren' },
				{ wert: 'ohne', text: 'Ohne Exemplare' }
			]}
			wert={appState.bestandsAnsicht}
			onwahl={(wert) => (appState.bestandsAnsicht = wert === 'ohne' ? 'ohne' : 'mit')}
		/>
		<!-- Anlegen steht vorn bei der Liste wie in Leserdatei, Klassensätzen und Benutzern (M3,
		     Spacing: „buttons should be close to the content they're affecting“). Die Aktionen für
		     markierte Titel kommen zuletzt, damit beim Markieren nichts verrutscht. -->
		<Button onclick={onCreateNew} class="mt-2 w-full sm:mt-0 sm:w-auto">
			<Icon name="add" />
			Titel anlegen
		</Button>
		<!-- Kein eigener Knopf für die Kamera: Die Suchpille sucht damit, die Maske „Titel anlegen"
		     scannt die ISBN im Feld (M3, Buttons: „Don't clutter your UI with too many buttons"). -->
		<Button variant="secondary" onclick={onRetryCovers} class="flex-1 sm:flex-none">
			<Icon name="refresh" />
			Cover herunterladen
		</Button>
		{#if anzahlGewaehlt > 0}
			<Button variant="secondary" onclick={onAssignClass}>
				<Icon name="menu_book" />
				{anzahlText(anzahlGewaehlt, 'Titel', 'Titel')} zum Klassensatz hinzufügen
			</Button>
			<Button variant="danger" onclick={onDelete} disabled={loescht}>
				{#if loescht}
					<Ladekreis size="sm" farbe="aktuell" />
				{:else}
					<Icon name="delete" />
				{/if}
				{anzahlText(anzahlGewaehlt, 'Titel', 'Titel')} löschen
			</Button>
		{/if}
	</div>
</div>
