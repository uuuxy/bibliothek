<!-- @component BuchakteKopf — der Kopf der Buchakte wie eine Play-Store-Detailseite: Titel
     groß links, darunter in ruhigem Grau die Einordnung (Fach · Jahrgang · Zweig ·
     Medienart), dann ISBN, Signatur und Standorte als Textzeilen, eine Zahlenreihe
     (verfügbar, Ausleiher, bestellt) und die Aktionen als Knopfreihe. Das Cover steht
     rechts mit Luft.

     Der Standort steht am Exemplar: Die Zeile nennt die Standorte der Exemplare im Bestand
     mit ihrer Zahl, gezählt aus den Karten des Reiters „Exemplare". -->
<script>
	import { Copy, MapPin, SquarePen } from '@lucide/svelte';
	import Button from '../ui/Button.svelte';
	import TitelLoeschen from './TitelLoeschen.svelte';
	import BuchCover from '../ui/BuchCover.svelte';
	import { standorteAusExemplaren, standortZeile } from '../../utils/standorte.js';
	import { exemplarZahlen } from './exemplarStatus.js';
	import { jahrgangSpanne } from '../../utils/format.js';

	/**
	 * @typedef {Object} Props
	 * @property {any} titel
	 * @property {any[]} ausleiher
	 * @property {any[]} exemplare
	 * @property {(() => void) | undefined} [onEdit] nur mit edit_books
	 * @property {(() => Promise<unknown> | void) | undefined} [onDelete] nur mit delete_books
	 */

	/** @type {Props} */
	let { titel, ausleiher, exemplare, onEdit = undefined, onDelete = undefined } = $props();

	let copied = $state(false);

	const jahrgang = $derived(jahrgangSpanne(titel.jahrgangVon, titel.jahrgangBis));

	/** Einordnung in einer Zeile, wie „Sept. 2026 · Buch 3 · Verlag" im Play Store. */
	const einordnung = $derived(
		[
			titel.subject,
			jahrgang ? `Jahrgang ${jahrgang}` : '',
			titel.istLernmittel ? 'Lernmittel' : '',
			titel.auflage || '',
			titel.medientyp && titel.medientyp !== 'Buch' ? titel.medientyp : ''
		]
			.filter(Boolean)
			.join(' · ')
	);

	const nummerArt = $derived(
		titel.medientyp === 'CD' || titel.medientyp === 'DVD' ? 'EAN' : 'ISBN'
	);
	const signatur = $derived(titel.signatur || titel.erweiterte_eigenschaften?.signatur || '');
	const standort = $derived(standortZeile(standorteAusExemplaren(exemplare)));
	// Den Bestand nennt „1 von 2 verfügbar"; daneben steht nur, was diese Zahl nicht sagt.
	const bestellt = $derived(exemplarZahlen(exemplare).bestellt);

	function kopieren() {
		if (!titel.isbn) return;
		navigator.clipboard.writeText(titel.isbn);
		copied = true;
		setTimeout(() => (copied = false), 2000);
	}
</script>

<div class="flex flex-col-reverse gap-8 sm:flex-row sm:items-start">
	<div class="min-w-0 flex-1">
		<h1 class="text-3xl leading-tight font-semibold wrap-break-word text-on-surface">
			{titel.title}
		</h1>
		{#if titel.untertitel}
			<p class="mt-1 text-lg text-on-surface-variant">{titel.untertitel}</p>
		{/if}
		{#if einordnung}
			<p class="mt-2 text-sm text-on-surface-variant">{einordnung}</p>
		{/if}

		<div class="mt-4 flex flex-col gap-1 text-sm text-on-surface-variant">
			<button
				class="flex w-fit cursor-pointer items-center gap-1.5 font-mono transition-colors hover:text-primary"
				onclick={kopieren}
				title="{nummerArt} kopieren"
			>
				<span>{nummerArt}: {titel.isbn || '–'}</span>
				{#if copied}
					<span class="font-sans text-xs font-medium text-primary">Kopiert!</span>
				{:else if titel.isbn}
					<Copy class="h-3.5 w-3.5" aria-hidden="true" />
				{/if}
			</button>
			{#if signatur}
				<span class="flex items-center gap-1.5">
					<MapPin class="h-3.5 w-3.5 shrink-0" aria-hidden="true" />
					<span class="font-mono">{signatur}</span>
					{#if standort}<span>· {standort}</span>{/if}
				</span>
			{:else if standort}
				<span class="flex items-center gap-1.5">
					<MapPin class="h-3.5 w-3.5 shrink-0" aria-hidden="true" />
					<span>{standort}</span>
				</span>
			{/if}
		</div>

		<!-- Zahlenreihe wie „E-Book | 544 Seiten": Wert oben, Bedeutung darunter, Trennlinien. -->
		<dl class="mt-6 flex divide-x divide-outline-variant">
			<div class="pr-6">
				<dd class="text-lg font-semibold tabular-nums text-on-surface">
					{titel.verfuegbar ?? 0} von {titel.gesamt ?? 0}
				</dd>
				<dt class="text-xs text-on-surface-variant">verfügbar</dt>
			</div>
			<div class="px-6">
				<dd class="text-lg font-semibold tabular-nums text-on-surface">{ausleiher.length}</dd>
				<dt class="text-xs text-on-surface-variant">Ausleiher</dt>
			</div>
			{#if bestellt > 0}
				<div class="pl-6">
					<dd class="text-lg font-semibold tabular-nums text-on-surface">{bestellt}</dd>
					<dt class="text-xs text-on-surface-variant">bestellt</dt>
				</div>
			{/if}
		</dl>

		{#if onEdit || onDelete}
			<div class="mt-6 flex flex-wrap items-center gap-3">
				{#if onEdit}
					<Button onclick={onEdit}>
						<SquarePen class="h-4 w-4" aria-hidden="true" />
						Titel bearbeiten
					</Button>
				{/if}
				{#if onDelete}
					<TitelLoeschen {onDelete} />
				{/if}
			</div>
		{/if}
	</div>

	<div class="w-40 shrink-0 sm:w-48">
		<!-- Der Schatten hebt das Cover als Gegenstand von der weißen Fläche ab — wie auf der
		     Play-Store-Detailseite. Nur hier am großen Einzel-Cover; die
		     Kacheln im Raster bleiben flach, zwölf Schatten nebeneinander wären Rauschen. -->
		<BuchCover
			coverUrl={titel.coverUrl}
			isbn={titel.isbn}
			titel={titel.title}
			fach={titel.subject}
			groesse="kachel"
			klasse="shadow-2xl"
		/>
	</div>
</div>
