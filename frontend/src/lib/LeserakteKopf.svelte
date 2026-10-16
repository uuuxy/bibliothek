<script>
	import { Camera, Lock, X } from '@lucide/svelte';
	import { ausleiheGesperrt } from './sperrStatus.js';
	import StatusChip from './components/ui/StatusChip.svelte';
	import LeserKontoStatus from './components/leser/LeserKontoStatus.svelte';
	import AbgangsjahrFeld from './components/leser/AbgangsjahrFeld.svelte';
	import { initialen } from './avatarKachel.js';
	import { leserArtText, istKollegium } from './leserArt.js';

	/** @type {{ leser: any, rechte?: { bearbeiten: boolean, foto: boolean }, timestamp: number, showWebcam: boolean, onDeselect: () => void, leftActions?: import('svelte').Snippet, onLock?: () => void, offen?: { anzahl: number, summe: number } }} */
	let {
		leser = $bindable(),
		rechte = { bearbeiten: false, foto: false },
		timestamp,
		showWebcam = $bindable(),
		onDeselect,
		leftActions,
		onLock,
		offen = undefined
	} = $props();

	const initials = $derived(initialen(leser));

	let imageFailed = $state(false);
	// Ein Kollege ist kein Schüler mit fehlenden Angaben: Klasse und Abgangsjahr gibt es
	// bei ihm nicht, und „Klasse " mit nichts dahinter sähe aus wie ein Datenfehler.
	const kollege = $derived(istKollegium(leser));
</script>

<div
	class="lg:col-span-1 relative bg-surface/60 border-r border-outline-variant px-7 pt-8 pb-6 flex flex-col items-start text-left gap-6"
>
	<!-- Schließen -->
	<button
		onclick={onDeselect}
		class="icon-btn absolute top-4 right-4 p-2 text-on-surface-variant"
		title="Akte schließen (ESC)"
		aria-label="Akte schließen"
	>
		<X class="w-5 h-5" aria-hidden="true" />
	</button>

	<!-- Foto -->
	<div class="relative group">
		{#if leser.foto_url && !imageFailed}
			<img
				src={leser.foto_url.startsWith('data:')
					? leser.foto_url
					: leser.foto_url + '?t=' + timestamp}
				alt="Passbild"
				class="w-28 h-28 object-cover rounded-2xl border border-outline-variant"
				onerror={() => (imageFailed = true)}
			/>
		{:else}
			<div
				class="w-28 h-28 rounded-2xl flex items-center justify-center bg-primary-container text-on-primary-container font-bold text-4xl tracking-tight select-none"
				aria-hidden="true"
			>
				{initials}
			</div>
		{/if}
		<button
			hidden={!rechte.foto}
			onclick={() => (showWebcam = true)}
			aria-label="Passbild mit Webcam aufnehmen"
			class="icon-btn absolute bottom-1 right-1 bg-secondary-container text-on-secondary-container"
			data-tip="Passbild aufnehmen"
		>
			<Camera class="h-4 w-4" aria-hidden="true" />
		</button>
	</div>

	<!-- Name & Metadaten -->
	<div class="w-full space-y-2">
		{#if ausleiheGesperrt(leser)}
			<StatusChip ton="fehler" text="Ausleihe gesperrt" icon={Lock} />
		{/if}

		<h3 class="text-3xl leading-tight font-bold wrap-break-word text-on-surface">
			{leser.vorname}
			{leser.nachname}
		</h3>
		<p class="text-lg font-bold text-on-surface-variant">
			{kollege ? leserArtText(leser.art) : `Klasse ${leser.klasse}`}
		</p>

		{#if !kollege}
			<AbgangsjahrFeld bind:leser darfBearbeiten={rechte.bearbeiten} />
		{/if}

		<!-- Ohne Ausweisnummer ist ein Kollege nur ohne aktiven Zugang (offene Anfrage,
		     deaktiviert, keine Schul-Adresse): Ein aktiver bekommt und behält eine
		     (Migrationen 136, 145). Eine leere Zeile sähe nach einem Anzeigefehler aus;
		     ohne Nummer gibt es auch keinen Ausweis zu drucken. -->
		{#if leser.barcode_id}
			<p class="text-sm text-on-surface-variant font-mono tracking-widest">{leser.barcode_id}</p>
		{:else}
			<p class="text-sm text-on-surface-variant italic">Noch keine Ausweisnummer</p>
		{/if}
	</div>

	<LeserKontoStatus {leser} {onLock} {offen} />

	<!-- Linke Aktionen (z. B. "Sitzung beenden" im Kiosk). Ausweis-Druck & DSGVO-
	     Auskunft leben bewusst rechts unter „Dokumente & Aktionen" — die Identitäts-
	     spalte bleibt rein Identität. -->
	{#if leftActions}
		<div class="w-full mt-auto pt-4">
			{@render leftActions()}
		</div>
	{/if}
</div>
