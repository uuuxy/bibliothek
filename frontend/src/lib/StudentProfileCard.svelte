<script>
	import { Camera, Lock, X } from '@lucide/svelte';
	import { ausleiheGesperrt } from './sperrStatus.js';
	import StudentKontoStatus from './components/students/StudentKontoStatus.svelte';
	import AbgangsjahrFeld from './components/students/AbgangsjahrFeld.svelte';
	import { initialen, avatarVerlauf } from './avatarKachel.js';
	import { leserArtText, istKollegium } from './leserArt.js';

	/** @type {{ profile: any, rechte?: { bearbeiten: boolean, foto: boolean }, timestamp: number, showWebcam: boolean, showDeleteConfirm: boolean, onDeselect: () => void, leftActions?: import('svelte').Snippet, onLock?: () => void }} */
	let {
		profile = $bindable(),
		rechte = { bearbeiten: false, foto: false },
		timestamp,
		showWebcam = $bindable(),
		showDeleteConfirm = $bindable(),
		onDeselect,
		leftActions,
		onLock
	} = $props();

	const initials = $derived(initialen(profile));
	const avatarGradient = $derived(avatarVerlauf(profile));

	let imageFailed = $state(false);
	// Ein Kollege ist kein Schüler mit fehlenden Angaben: Klasse und Abgangsjahr gibt es
	// bei ihm nicht, und „Klasse " mit nichts dahinter sähe aus wie ein Datenfehler.
	const kollege = $derived(istKollegium(profile));
</script>

<div
	class="lg:col-span-1 relative bg-slate-50/60 border-r border-slate-200 px-7 pt-8 pb-6 flex flex-col items-start text-left gap-6"
>
	<!-- Schließen -->
	<button
		onclick={onDeselect}
		class="absolute top-4 right-4 p-2 text-slate-400 hover:text-slate-600 hover:bg-slate-200/60 rounded-full transition-colors cursor-pointer"
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
				class="w-28 h-28 object-cover rounded-2xl border border-slate-200"
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
			class="absolute bottom-1 right-1 p-2 rounded-full bg-slate-900/60 hover:bg-slate-900 text-white backdrop-blur-md transition-all cursor-pointer border border-white/20"
			title="Passbild aufnehmen"
		>
			<Camera class="h-4 w-4" aria-hidden="true" />
		</button>
	</div>

	<!-- Name & Metadaten -->
	<div class="w-full space-y-2">
		{#if ausleiheGesperrt(profile)}
			<span
				class="inline-flex items-center px-2.5 py-1 rounded-md text-xs font-medium bg-rose-100 text-rose-700 border border-rose-200 mb-1"
			>
				<Lock class="w-3 h-3 mr-1" aria-hidden="true" />
				Ausleihe gesperrt
			</span>
		{/if}

		<h3 class="text-3xl font-bold text-slate-900 leading-tight">
			{profile.vorname}
			{profile.nachname}
		</h3>
		<p class="text-lg font-bold text-slate-700">
			{kollege ? leserArtText(profile.art) : `Klasse ${profile.klasse}`}
		</p>

		{#if !kollege}
			<AbgangsjahrFeld bind:profile darfBearbeiten={rechte.bearbeiten} />
		{/if}

		<!-- Ein Kollege aus der Selbstanmeldung hat noch keine Ausweisnummer. Eine leere
		     Zeile sähe nach einem Anzeigefehler aus; ohne Nummer gibt es auch keinen
		     Ausweis zu drucken. Eingetragen wird sie in „Benutzer & Rechte". -->
		{#if profile.barcode_id}
			<p class="text-sm text-slate-400 font-mono tracking-widest">{profile.barcode_id}</p>
		{:else}
			<p class="text-sm text-on-surface-variant italic">Noch keine Ausweisnummer</p>
		{/if}
	</div>

	<StudentKontoStatus {profile} {onLock} />

	<!-- Linke Aktionen (z. B. "Sitzung beenden" im Kiosk). Ausweis-Druck & DSGVO-
	     Auskunft leben bewusst rechts unter „Dokumente & Aktionen" — die Identitäts-
	     spalte bleibt rein Identität. -->
	{#if leftActions}
		<div class="w-full mt-auto pt-4">
			{@render leftActions()}
		</div>
	{/if}
</div>
