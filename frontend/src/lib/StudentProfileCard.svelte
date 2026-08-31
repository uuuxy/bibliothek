<script>
	import { Camera, Lock, RotateCcw, X } from '@lucide/svelte';
	import { apiClient } from './apiFetch.js';
	import { ausleiheGesperrt } from './sperrStatus.js';
	import { studentTabExtensions } from './plugins.svelte.js';
	import Button from './components/ui/Button.svelte';
	import Feld from './components/ui/Feld.svelte';
	import StudentKontoStatus from './components/students/StudentKontoStatus.svelte';
	import { initialen, avatarVerlauf } from './avatarKachel.js';

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

	let editingAbgang = $state(false); // Abgangsjahr inline bearbeiten
	let abgangInput = $state(0);
	let abgangSaving = $state(false);
	let abgangError = $state('');
	let imageFailed = $state(false);

	function startEditAbgang() {
		abgangInput = profile.abgaenger_jahr;
		abgangError = '';
		editingAbgang = true;
	}

	/** Calculates the expected graduation year from a class string (mirrors backend logic) */
	function calcAbgangFromKlasse(klasse) {
		const kl = (klasse || '').toLowerCase().trim();
		const m = kl.match(/^(\d+)(.*)/);
		if (!m) return new Date().getFullYear() + 5;
		const grade = parseInt(m[1], 10);
		const suffix = m[2] || '';
		const maxGrade = suffix.startsWith('h') ? 9 : grade >= 11 ? 13 : 10;
		const yearsLeft = Math.max(0, maxGrade - grade);
		const now = new Date();
		const base = now.getMonth() >= 7 ? now.getFullYear() + 1 : now.getFullYear();
		return base + yearsLeft;
	}

	async function saveAbgang() {
		const year = parseInt(String(abgangInput), 10);
		if (isNaN(year) || year < 2000 || year > 2100) {
			abgangError = 'Bitte ein gültiges Jahr eingeben (2000–2100)';
			return;
		}
		abgangSaving = true;
		abgangError = '';
		try {
			const res = await apiClient.patch(`/api/schueler/${profile.id}`, { abgaenger_jahr: year });
			if (res.ok) {
				profile.abgaenger_jahr = year;
				editingAbgang = false;
			} else {
				const d = await res.json().catch(() => ({}));
				abgangError = d.error || 'Fehler beim Speichern';
			}
		} catch {
			abgangError = 'Netzwerkfehler';
		} finally {
			abgangSaving = false;
		}
	}
</script>

<div
	class="lg:col-span-1 relative bg-slate-50/60 border-r border-slate-200 px-7 pt-8 pb-6 flex flex-col items-start text-left gap-6"
>
	<!-- Schließen -->
	<button
		onclick={onDeselect}
		class="absolute top-4 right-4 p-2 text-slate-400 hover:text-slate-600 hover:bg-slate-200/60 rounded-full transition-colors cursor-pointer"
		title="Schüler schließen (ESC)"
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
		<p class="text-lg font-bold text-slate-700">Klasse {profile.klasse}</p>

		{#if rechte.bearbeiten}
			{#if editingAbgang}
				<div class="flex items-center gap-2 flex-wrap">
					<Feld
						type="number"
						min="2000"
						max="2100"
						bind:value={abgangInput}
						aria-label="Abgangsjahr"
						feld="w-24 text-center font-bold"
					/>
					<Button
						variant="secondary"
						size="sm"
						onclick={() => {
							abgangInput = calcAbgangFromKlasse(profile.klasse);
						}}
						title="Automatisch aus Klasse berechnen"
						><RotateCcw class="h-3.5 w-3.5" aria-hidden="true" /> Neu berechnen</Button
					>
					<Button size="sm" onclick={saveAbgang} disabled={abgangSaving}>
						{abgangSaving ? '…' : 'Speichern'}
					</Button>
					<Button variant="ghost" size="sm" onclick={() => (editingAbgang = false)}>✕</Button>
				</div>
				{#if abgangError}<p class="text-xs text-rose-500 mt-1">{abgangError}</p>{/if}
			{:else}
				<button
					onclick={startEditAbgang}
					class="text-base text-slate-500 font-semibold hover:text-blue-600 hover:underline cursor-pointer transition-colors"
					title="Abgangsjahr bearbeiten"
				>
					Abgang {profile.abgaenger_jahr} ✎
				</button>
			{/if}
		{:else}
			<p class="text-base text-slate-500 font-semibold">Abgang {profile.abgaenger_jahr}</p>
		{/if}

		<p class="text-sm text-slate-400 font-mono tracking-widest">{profile.barcode_id}</p>
	</div>

	<StudentKontoStatus {profile} {onLock} />

	<!-- Plugin-Erweiterungen -->
	{#if studentTabExtensions.length > 0}
		<div class="w-full flex flex-col gap-3">
			{#each studentTabExtensions as ext, _i (_i)}
				{@const Component = ext.component}
				<div class="w-full">
					<span class="block text-xs font-medium text-slate-400 mb-2">{ext.name}</span>
					<Component student={profile} {...ext.props} />
				</div>
			{/each}
		</div>
	{/if}

	<!-- Linke Aktionen (z. B. "Sitzung beenden" im Kiosk). Ausweis-Druck & DSGVO-
	     Auskunft leben bewusst rechts unter „Dokumente & Aktionen" — die Identitäts-
	     spalte bleibt rein Identität. -->
	{#if leftActions}
		<div class="w-full mt-auto pt-4">
			{@render leftActions()}
		</div>
	{/if}
</div>
