<script>
	import { Lock } from '@lucide/svelte';
	import { authStore } from '../../stores/authStore.svelte.js';
	import { idleLock } from '../../stores/idleLock.svelte.js';
	import Button from '../ui/Button.svelte';

	// Sperrbildschirm nach Inaktivität (A4 in docs/datenschutz_offene_punkte.md).
	// Verdeckt die ganze Anwendung; weiter geht es nur mit dem Passwort der angemeldeten
	// Person (echte Wiederanmeldung gegen den Mailserver) oder per Abmelden. Der Inhalt
	// dahinter bleibt im DOM — die Sperre ist eine Sichtschutz-, keine Sicherheitsgrenze
	// gegen jemanden mit Entwicklerwerkzeugen; dafür ist die Abmeldung da.

	let passwort = $state('');

	$effect(() => {
		setTimeout(() => document.getElementById('sperre-passwort')?.focus(), 50);
	});

	/** @param {SubmitEvent} e */
	async function entsperren(e) {
		e.preventDefault();
		if (idleLock.entsperreLaeuft) return;
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
	class="fixed inset-0 z-[60] flex items-center justify-center bg-surface p-6 no-print"
	role="dialog"
	aria-modal="true"
	aria-labelledby="sperre-titel"
	data-testid="sperrbildschirm"
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
		<input
			id="sperre-passwort"
			type="password"
			autocomplete="current-password"
			aria-label="Passwort"
			bind:value={passwort}
			disabled={idleLock.entsperreLaeuft}
			class="w-full bg-surface-container-low border border-outline-variant rounded-xl px-4 text-on-surface focus:outline-none focus:border-primary transition-colors"
			placeholder="Passwort"
		/>
		<Button type="submit" size="lg" disabled={idleLock.entsperreLaeuft} class="w-full">
			{#if idleLock.entsperreLaeuft}
				<div
					class="w-4 h-4 border-2 border-on-primary/40 border-t-on-primary rounded-full animate-spin"
				></div>
				Prüfe…
			{:else}
				Entsperren
			{/if}
		</Button>
		{#if idleLock.entsperrFehler}
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
