<script>
	import { tick } from 'svelte';
	import { fokusFalle } from '../ui/fokusFalle.js';
	import { Lock } from '@lucide/svelte';
	import Ladekreis from '../ui/Ladekreis.svelte';
	import { authStore } from '../../stores/authStore.svelte.js';
	import { leerlaufSperre } from '../../stores/leerlaufSperre.svelte.js';
	import Button from '../ui/Button.svelte';
	import Feld from '../ui/Feld.svelte';
	import BestaetigungsDialog from '../ui/BestaetigungsDialog.svelte';
	import { verwerfenFrage } from '../../stores/bestaetigung.svelte.js';
	import { uiStore } from '../../stores/uiStore.svelte.js';
	import { scanSchutz } from '../../scanErkennung.js';

	// Sperrbildschirm nach Inaktivität. Die
	// Anwendung dahinter bleibt stehen, damit Ungespeichertes die Sperre überlebt, ist aber
	// ausgeblendet und träge (App.svelte): Die Druckvorschau zeigt sie nicht, Tab erreicht
	// sie nicht, Screenreader lesen sie nicht (Prüfung 22.08.2026, A6). Tasten und Zeiger
	// hält sperrSchild.js von ihr fern; data-sperrbildschirm ist sein Kennzeichen. Weiter
	// geht es nur mit dem Passwort der angemeldeten Person oder per Abmelden. Der Server
	// hält die Anmeldung so lange gesperrt (stores/leerlaufSperre).

	let passwort = $state('');
	// Ein Scan am Sperrbildschirm ist kein Passwort: Er geht nicht zum Server, wo er als
	// Fehlversuch zählte, und er drückt keinen Knopf.
	let scanErkannt = $state(false);

	function beiScan() {
		leerlaufSperre.entsperrFehler = null;
		scanErkannt = true;
	}

	/** @param {SubmitEvent} e */
	async function entsperren(e) {
		e.preventDefault();
		if (leerlaufSperre.entsperreLaeuft) return;
		scanErkannt = false;
		const ok = await leerlaufSperre.entsperren(passwort);
		passwort = '';
		// Das Feld war während der Prüfung gesperrt und nimmt den Fokus zurück, sobald es wieder
		// bedienbar ist; eine inzwischen offene Rückfrage behält ihn.
		if (ok || verwerfenOffen) return;
		await tick();
		document.getElementById('sperre-passwort')?.focus();
	}

	// Hält eine Maske hinter der Sperre Ungespeichertes, fragt der Bildschirm vor dem Abmelden
	// selbst: Die Rückfragen der Anwendung sind hinter der Sperre nicht zu sehen (App.svelte).
	let verwerfenOffen = $state(false);
	function abmelden() {
		if (uiStore.verlassenSperre?.()) verwerfenOffen = true;
		else meldeAb();
	}
	function meldeAb() {
		leerlaufSperre.stop();
		authStore.handleLogout();
	}
</script>

<div
	class="fixed inset-0 z-60 flex items-center justify-center bg-surface p-6"
	role="dialog"
	aria-modal="true"
	aria-labelledby="sperre-titel"
	data-testid="sperrbildschirm"
	data-sperrbildschirm
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
			disabled={leerlaufSperre.entsperreLaeuft}
			placeholder="Passwort"
		/>
		<Button type="submit" size="lg" disabled={leerlaufSperre.entsperreLaeuft} class="w-full">
			{#if leerlaufSperre.entsperreLaeuft}
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
		{:else if leerlaufSperre.entsperrFehler}
			<p class="text-xs text-error font-semibold animate-slide-up" role="alert">
				{leerlaufSperre.entsperrFehler}
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
	<BestaetigungsDialog
		frage={verwerfenOffen ? verwerfenFrage('Verwerfen und abmelden') : null}
		onantwort={(verwerfen) => {
			verwerfenOffen = false;
			if (verwerfen) meldeAb();
		}}
	/>
</div>
