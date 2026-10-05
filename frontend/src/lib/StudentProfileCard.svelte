<script>
	import { Camera, Lock, X } from '@lucide/svelte';
	import { ausleiheGesperrt } from './sperrStatus.js';
	import StatusChip from './components/ui/StatusChip.svelte';
	import StudentKontoStatus from './components/students/StudentKontoStatus.svelte';
	import AbgangsjahrFeld from './components/students/AbgangsjahrFeld.svelte';
	import { initialen, avatarVerlauf } from './avatarKachel.js';
	import { leserArtText, istKollegium } from './leserArt.js';

	/** @type {{ profile: any, rechte?: { bearbeiten: boolean, foto: boolean }, timestamp: number, showWebcam: boolean, showDeleteConfirm: boolean, onDeselect: () => void, leftActions?: import('svelte').Snippet, onLock?: () => void, offen?: { anzahl: number, summe: number } }} */
	let {
		profile = $bindable(),
		rechte = { bearbeiten: false, foto: false },
		timestamp,
		showWebcam = $bindable(),
		showDeleteConfirm = $bindable(),
		onDeselect,
		leftActions,
		onLock,
		offen = undefined
	} = $props();

	const initials = $derived(initialen(profile));
	const avatarGradient = $derived(avatarVerlauf(profile));

	let imageFailed = $state(false);
	// Ein Kollege ist kein Schüler mit fehlenden Angaben: Klasse und Abgangsjahr gibt es
	// bei ihm nicht, und „Klasse " mit nichts dahinter sähe aus wie ein Datenfehler.
	const kollege = $derived(istKollegium(profile));
</script>

<div
	class="lg:col-span-1 relative bg-surface/60 border-r border-outline-variant px-7 pt-8 pb-6 flex flex-col items-start text-left gap-6"
>
	<!-- Schließen -->
	<button
		onclick={onDeselect}
		class="icon-btn absolute top-4 right-4 p-2 text-on-surface-variant"
		title="Akte schließen (ESC)"
	>
		<X class="w-5 h-5" aria-hidden="true" />
	</button>

	<!-- Foto -->
	<div class="relative group">
		{#if profile.foto_url && !imageFailed}
			<img
				src={profile.foto_url.startsWith('data:')
					? profile.foto_url
					: profile.foto_url + '?t=' + timestamp}
				alt="Passbild"
				class="w-28 h-28 object-cover rounded-2xl border border-outline-variant"
				onerror={() => (imageFailed = true)}
			/>
		{:else}
			<div
				class="w-28 h-28 rounded-2xl border border-black/5 shadow-inner flex items-center justify-center text-white font-bold text-4xl tracking-tight select-none bg-linear-to-br {avatarGradient}"
				aria-hidden="true"
			>
				{initials}
			</div>
		{/if}
		<button
			hidden={!rechte.foto}
			onclick={() => (showWebcam = true)}
			aria-label="Passbild mit Webcam aufnehmen"
			class="absolute bottom-1 right-1 p-2 rounded-full bg-scrim/60 text-white backdrop-blur-md cursor-pointer border border-white/20"
			title="Passbild aufnehmen"
		>
			<Camera class="h-4 w-4" aria-hidden="true" />
		</button>
	</div>

	<!-- Name & Metadaten -->
	<div class="w-full space-y-2">
		{#if ausleiheGesperrt(profile)}
			<StatusChip ton="fehler" text="Ausleihe gesperrt" icon={Lock} />
		{/if}

		<h3 class="text-3xl font-bold text-on-surface leading-tight">
			{profile.vorname}
			{profile.nachname}
		</h3>
		<p class="text-lg font-bold text-on-surface-variant">
			{kollege ? leserArtText(profile.art) : `Klasse ${profile.klasse}`}
		</p>

		{#if !kollege}
			<AbgangsjahrFeld bind:profile darfBearbeiten={rechte.bearbeiten} />
		{/if}

		<!-- Ohne Ausweisnummer ist ein Kollege nur ohne aktiven Zugang (offene Anfrage,
		     deaktiviert, keine Schul-Adresse): Ein aktiver bekommt und behält eine
		     (Migrationen 136, 145). Eine leere Zeile sähe nach einem Anzeigefehler aus;
		     ohne Nummer gibt es auch keinen Ausweis zu drucken. -->
		{#if profile.barcode_id}
			<p class="text-sm text-on-surface-variant font-mono tracking-widest">{profile.barcode_id}</p>
		{:else}
			<p class="text-sm text-on-surface-variant italic">Noch keine Ausweisnummer</p>
		{/if}
	</div>

	<StudentKontoStatus {profile} {onLock} {offen} />

	<!-- Linke Aktionen (z. B. "Sitzung beenden" im Kiosk). Ausweis-Druck & DSGVO-
	     Auskunft leben bewusst rechts unter „Dokumente & Aktionen" — die Identitäts-
	     spalte bleibt rein Identität. -->
	{#if leftActions}
		<div class="w-full mt-auto pt-4">
			{@render leftActions()}
		</div>
	{/if}
</div>
