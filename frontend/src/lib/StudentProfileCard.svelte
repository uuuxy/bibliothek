<script>
	import { Camera, Lock, RotateCcw, X } from '@lucide/svelte';
	import { apiClient } from './apiFetch.js';
	import { studentTabExtensions } from './plugins.svelte.js';
	import Button from './components/ui/Button.svelte';
	import StudentKontoStatus from './components/students/StudentKontoStatus.svelte';

	/** @type {{ profile: any, role: string, timestamp: number, showWebcam: boolean, showDeleteConfirm: boolean, onDeselect: () => void, leftActions?: import('svelte').Snippet, onLock?: () => void }} */
	let {
		profile = $bindable(),
		role = '',
		timestamp,
		showWebcam = $bindable(),
		showDeleteConfirm = $bindable(),
		onDeselect,
		leftActions,
		onLock
	} = $props();

	// ── Initialen-Avatar (Passbild-Ersatz) ────────────────────────────────────
	// Eine leere graue Box liest sich wie „kaputt". Fehlt das Foto, zeigen wir –
	// wie Apple Kontakte / Google – eine Initialen-Kachel auf farbigem Verlauf.
	// Die Farbe wird deterministisch aus dem Namen abgeleitet, damit derselbe
	// Schüler immer dieselbe Kachel bekommt (wirkt gewollt statt zufällig).
	const AVATAR_GRADIENTS = [
		'from-blue-500 to-indigo-600',
		'from-emerald-500 to-teal-600',
		'from-fuchsia-500 to-purple-600',
		'from-amber-500 to-orange-600',
		'from-rose-500 to-pink-600',
		'from-sky-500 to-cyan-600',
		'from-violet-500 to-purple-600',
		'from-lime-500 to-emerald-600'
	];
	const initials = $derived(
		((profile.vorname?.[0] ?? '') + (profile.nachname?.[0] ?? '')).toUpperCase() || '?'
	);
	const avatarGradient = $derived.by(() => {
		const key = `${profile.vorname ?? ''} ${profile.nachname ?? ''}`;
		let h = 0;
		for (let i = 0; i < key.length; i++) h = (h * 31 + key.charCodeAt(i)) >>> 0;
		return AVATAR_GRADIENTS[h % AVATAR_GRADIENTS.length];
	});

	// ── Abgangsjahr inline editing ────────────────────────────────────────────
	let editingAbgang = $state(false);
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
		let maxGrade;
		if (suffix.startsWith('h')) maxGrade = 9;
		else if (grade >= 11) maxGrade = 13;
		else maxGrade = 10;
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
		{#if profile.ist_gesperrt}
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

		{#if role === 'admin'}
			{#if editingAbgang}
				<div class="flex items-center gap-2 flex-wrap">
					<input
						type="number"
						min="2000"
						max="2100"
						bind:value={abgangInput}
						class="w-24 h-7 px-2 text-sm border border-blue-400 rounded-lg text-center font-bold focus:outline-none focus:ring-2 focus:ring-blue-200"
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
