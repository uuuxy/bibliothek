<script>
	import { melde } from '../inventur/lib/store.svelte.js';
	import Ladekreis from './components/ui/Ladekreis.svelte';
	import BuchakteAusleiher from './BuchakteAusleiher.svelte';
	import BuchakteExemplare from './BuchakteExemplare.svelte';
	import BuchakteHistorie from './BuchakteHistorie.svelte';
	import BuchakteVormerkungen from './BuchakteVormerkungen.svelte';
	import BuchakteKopf from './BuchakteKopf.svelte';
	import { useBuchakte } from './useBuchakte.svelte.js';
	import Seitenrahmen from './components/layout/Seitenrahmen.svelte';
	import LadeFehler from './components/ui/LadeFehler.svelte';
	import Reiter from './components/ui/Reiter.svelte';
	import { authStore } from './stores/authStore.svelte.js';
	import { hatRecht } from './menue.js';
	import { exemplarZahlen } from './components/exemplarStatus.js';
	import { ChevronLeft, FaceSlightlyFrowning } from '@lucide/svelte';
	import { untrack } from 'svelte';

	/** @type {{ titelId: string | null, onBack: () => void }} */
	let { titelId, onBack } = $props();

	const akte = useBuchakte();

	// Enges Theken-Recht (db/rechte_vorgabe.go): Ohne manage_vormerkungen antwortet die API
	// mit 403 — dann den Reiter gar nicht erst anbieten statt leer scheitern lassen.
	const darfVormerken = $derived(hatRecht(authStore.currentUser, 'manage_vormerkungen'));
	// Die Aktionen der Akte hängen am Recht — vorher sah jeder „Titel bearbeiten" und
	// „Gesamten Titel löschen", auch wer beides nicht durfte (Knopf → 403).
	const darfBearbeiten = $derived(hatRecht(authStore.currentUser, 'edit_books'));
	const darfLoeschen = $derived(hatRecht(authStore.currentUser, 'delete_books'));
	// Eine Zahl, die niemand kennt, ist ein Fragezeichen, kein „(0)": Scheitert der Abruf einer
	// Liste, wäre „Ausleiher (0)" eine Aussage über den Bestand, die niemand geprüft hat.
	/** @param {string} name @param {number} anzahl */
	const zaehler = (name, anzahl) =>
		`${name} (${akte.fehlendeListen.includes(name) ? '?' : anzahl})`;
	// „Exemplare" zählt den Bestand wie „1 von 2 verfügbar" im Kopf; bestellte und
	// ausgesonderte stehen im Reiter mit ihrem Wort an der Karte.
	const reiter = $derived([
		{ id: 'ausleiher', label: zaehler('Ausleiher', akte.ausleiher.length) },
		{ id: 'exemplare', label: zaehler('Exemplare', exemplarZahlen(akte.exemplare).bestand) },
		...(darfVormerken
			? [{ id: 'vormerkungen', label: zaehler('Vormerkungen', akte.vormerkungen.length) }]
			: []),
		{ id: 'historie', label: 'Historie' }
	]);

	// Der Effekt hängt an genau einer Sache, der Titel-ID. untrack sorgt dafür, dass nichts,
	// was ladeAlles unterwegs liest (appState.selectedBook, sein eigener Zustand), zum Auslöser
	// dieses Effekts wird; sonst läuft die Akte in einer Endlosschleife, bis Svelte sie
	// abbricht (siehe useBuchakte).
	$effect(() => {
		const id = titelId;
		if (id) untrack(() => akte.ladeAlles(id));
	});
</script>

<Seitenrahmen>
	<!-- Back Button + Breadcrumb -->
	<div class="flex items-center justify-between">
		<div class="flex items-center gap-3">
			<button
				onclick={onBack}
				class="flex items-center gap-2 px-3 py-2 rounded-xl text-on-surface-variant hover:text-on-surface transition-colors text-sm font-semibold cursor-pointer"
			>
				<ChevronLeft class="w-4 h-4" aria-hidden="true" />
				Zurück zum Katalog
			</button>
			<span class="text-on-surface-variant">/</span>
			<span class="text-on-surface-variant text-sm truncate max-w-xs"
				>{akte.buch?.title ?? 'Lade...'}</span
			>
		</div>
	</div>

	{#if akte.laedt}
		<div class="flex justify-center items-center py-32">
			<Ladekreis size="lg" />
		</div>
	{:else if akte.buch}
		<BuchakteKopf
			titel={akte.buch}
			ausleiher={akte.ausleiher}
			exemplare={akte.exemplare}
			onEdit={darfBearbeiten ? akte.bearbeiteTitel : undefined}
			onDelete={darfLoeschen ? () => akte.loescheTitel(melde, onBack) : undefined}
		/>

		{#if akte.fehlendeListen.length > 0}
			<p class="text-sm font-semibold text-error" role="alert">
				Nicht geladen: {akte.fehlendeListen.join(', ')}. Diese Reiter sind leer, weil der Abruf
				gescheitert ist — nicht, weil nichts da wäre.
			</p>
		{/if}

		<Reiter
			etikett="Bereiche der Buchakte"
			{reiter}
			aktiv={akte.aktiverReiter}
			onwahl={(id) => (akte.aktiverReiter = id)}
		/>

		<!-- Tab Content -->
		<div class="w-full">
			{#if akte.aktiverReiter === 'ausleiher'}
				<BuchakteAusleiher ausleiher={akte.ausleiher} titel={akte.buch} {onBack} />
			{:else if akte.aktiverReiter === 'exemplare'}
				<BuchakteExemplare
					bind:exemplare={akte.exemplare}
					titel={akte.buch}
					ladeAlles={akte.ladeAlles}
				/>
			{:else if akte.aktiverReiter === 'historie'}
				<BuchakteHistorie verlauf={akte.verlauf} />
			{:else if akte.aktiverReiter === 'vormerkungen'}
				<BuchakteVormerkungen bind:vormerkungen={akte.vormerkungen} titel={akte.buch} />
			{/if}
		</div>
	{:else if akte.kopfFehler}
		<!-- „Nicht gefunden" wäre hier eine Aussage über den Bestand. Der Abruf ist
		     gescheitert — den Titel gibt es womöglich, es kam nur nichts an. -->
		<LadeFehler
			onerneut={() => titelId && akte.ladeAlles(titelId)}
			titel="Titel nicht geladen"
			text={akte.kopfFehler}
		/>
	{:else}
		<div class="py-24 flex flex-col items-center text-on-surface-variant gap-3">
			<FaceSlightlyFrowning class="w-12 h-12" aria-hidden="true" />
			<p class="font-semibold">Buch nicht gefunden.</p>
			<button
				onclick={onBack}
				class="text-primary text-sm font-semibold hover:underline cursor-pointer"
				>Zurück zum Katalog</button
			>
		</div>
	{/if}
</Seitenrahmen>
