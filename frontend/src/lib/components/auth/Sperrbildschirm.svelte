<script>
	import { fokusFalle } from '../ui/fokusFalle.js';
	import { Lock } from '@lucide/svelte';
	import Ladekreis from '../ui/Ladekreis.svelte';
	import { authStore } from '../../stores/authStore.svelte.js';
	import { idleLock } from '../../stores/idleLock.svelte.js';
	import Button from '../ui/Button.svelte';
	import Feld from '../ui/Feld.svelte';
	import { scanSchutz } from '../../scanErkennung.js';

	// Sperrbildschirm nach Inaktivität (A4 in docs/datenschutz_offene_punkte.md).
	// Ersetzt die ganze Anwendung (App.svelte rendert Sidebar und Inhalt im gesperrten
	// Zustand NICHT — vorher lag dieser Bildschirm nur darüber, und Strg+P zeigte in der
	// Druckvorschau die Seite dahinter, Tab verließ die Sperre, Screenreader lasen weiter;
	// Prüfung 22.08.2026, A6). Weiter geht es nur mit dem Passwort der angemeldeten Person
	// oder per Abmelden. Der Server hält die Anmeldung so lange gesperrt (stores/idleLock).

	let passwort = $state('');
	// Ein Scan am Sperrbildschirm ist kein Passwort: Er geht nicht zum Server, wo er als
	// Fehlversuch zählte, und er drückt keinen Knopf.
	let scanErkannt = $state(false);

	function beiScan() {
		idleLock.entsperrFehler = null;
		scanErkannt = true;
	}

	$effect(() => {
		setTimeout(() => document.getElementById('sperre-passwort')?.focus(), 50);
	});

	/** @param {SubmitEvent} e */
	async function entsperren(e) {
		e.preventDefault();
		if (idleLock.entsperreLaeuft) return;
		scanErkannt = false;
		const ok = await idleLock.entsperren(passwort);
		passwort = '';
		if (!ok) setTimeout(() => document.getElementById('sperre-passwort')?.focus(), 50);
	}

	function abmelden() {
		idleLock.stop();
		authStore.handleLogout();
	}
</script>

<div
	class="fixed inset-0 z-60 flex items-center justify-center bg-surface p-6"
	role="dialog"
	aria-modal="true"
	aria-labelledby="sperre-titel"
	data-testid="sperrbildschirm"
	use:fokusFalle
	use:scanSchutz={beiScan}
>
	<form
		onsubmit={entsperren}
		class="w-full max-w-sm rounded-3xl bg-surface-container-lowest border border-outline-variant p-8 flex flex-col items-center space-y-5 animate-fade-in"
	>
		<div class="w-12 h-12 rounded-full bg-secondary-container flex items-center justify-center">
			<Lock class="w-6 h-6 text-on-secondary-container" aria-hidden="true" />
		</div>
		<div class="text-center space-y-1">
			<h2 id="sperre-titel" class="text-base font-bold text-on-surface">
				Gesperrt wegen Inaktivität
			</h2>
			<p class="text-xs text-on-surface-variant">
				Angemeldet als <span class="font-semibold text-on-surface"
					>{authStore.currentUser?.email}</span
				>
			</p>
		</div>
		<Feld
			id="sperre-passwort"
			type="password"
			autocomplete="current-password"
			aria-label="Passwort"
			bind:value={passwort}
			disabled={idleLock.entsperreLaeuft}
			placeholder="Passwort"
		/>
		<Button type="submit" size="lg" disabled={idleLock.entsperreLaeuft} class="w-full">
			{#if idleLock.entsperreLaeuft}
				<Ladekreis size="sm" farbe="aktuell" />
				Prüfe…
			{:else}
				Entsperren
			{/if}
		</Button>
		{#if scanErkannt}
			<p class="text-xs text-error font-semibold text-center animate-slide-up" role="alert">
				Scan erkannt: Die Anwendung ist gesperrt. Bitte erst das Passwort eintippen, dann scannen.
			</p>
		{:else if idleLock.entsperrFehler}
			<p class="text-xs text-error font-semibold animate-slide-up" role="alert">
				{idleLock.entsperrFehler}
			</p>
		{/if}
		<button
			type="button"
			onclick={abmelden}
			class="text-xs font-semibold text-on-surface-variant underline-offset-2 hover:underline cursor-pointer"
		>
			Abmelden und als andere Person anmelden
		</button>
	</form>
</div>
