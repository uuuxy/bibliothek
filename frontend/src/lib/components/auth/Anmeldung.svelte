<script>
	import { authStore } from '../../stores/authStore.svelte.js';
	import Ladekreis from '../ui/Ladekreis.svelte';
	import Button from '../ui/Button.svelte';
	import Feld from '../ui/Feld.svelte';
	import logoUrl from '../../../assets/logo.png';
	import { scanSchutz } from '../../utils/scanErkennung.js';

	// Ein Scan im Passwortfeld ginge als Anmeldeversuch zum Server und zählte als Fehlversuch.
	function beiScan() {
		authStore.loginError = 'Scan erkannt: Bitte erst anmelden, dann scannen.';
	}

	// „Anmelden" bleibt bei leerem Feld bedienbar: Der Klick nennt am Feld, was fehlt, und führt
	// dorthin. handleLogin prüft dasselbe noch einmal.
	let verlangt = $state(false);
	const emailFehlt = $derived(verlangt && !authStore.loginEmail.trim());
	const passwortFehlt = $derived(verlangt && !authStore.loginPassword);

	// Ist die Domain der Schule gesetzt, genügt der Name vor dem @; den Rest ergänzt der Server.
	// Ohne Antwort bleibt das Feld bei der vollen Adresse.
	let domain = $state('');
	async function ladeDomain() {
		try {
			const res = await fetch('/api/auth/domain');
			if (!res.ok) {
				domain = '';
				return;
			}
			const wert = await res.json();
			domain = typeof wert === 'string' ? wert : '';
		} catch {
			domain = '';
		}
	}
	// Ein Text unter dem Feld: Der Fehler ersetzt den Hinweis, statt unter ihm zu stehen.
	const emailHinweis = $derived.by(() => {
		if (emailFehlt)
			return domain
				? 'Bitte deinen Namen oder deine Schul-E-Mail eintragen.'
				: 'Bitte deine Schul-E-Mail eintragen.';
		return domain ? `Der Name vor dem @ genügt – @${domain} ergänzt das Programm.` : '';
	});

	/** @param {SubmitEvent} e */
	function absenden(e) {
		e.preventDefault();
		verlangt = true;
		if (!authStore.loginEmail.trim()) return document.getElementById('login-email')?.focus();
		if (!authStore.loginPassword) return document.getElementById('login-password')?.focus();
		authStore.handleLogin(e, undefined);
	}

	$effect(() => {
		setTimeout(() => document.getElementById('login-email')?.focus(), 50);
		ladeDomain();
	});
</script>

<main class="min-h-screen flex items-center justify-center p-6 bg-surface">
	<form
		onsubmit={absenden}
		use:scanSchutz={beiScan}
		class="w-full max-w-md p-8 rounded-3xl bg-surface-container-lowest shadow-xl flex flex-col items-center space-y-6 animate-fade-in no-print"
	>
		<img src={logoUrl} alt="Bibliosys Logo" class="w-24 h-24 object-contain" />
		<div class="text-center space-y-1.5">
			<h2 class="text-base font-bold text-on-surface">Anmelden</h2>
			<p class="text-sm text-on-surface-variant">Mit den Zugangsdaten deiner Schul-E-Mail.</p>
		</div>
		<!-- Die Beschriftung steht über dem Feld wie in jeder Maske: Ein Platzhalter verschwände beim
		     Tippen, und M3 verlangt „Label text should always be visible". -->
		<div class="w-full space-y-4">
			<Feld
				id="login-email"
				etikett={domain ? 'Name oder E-Mail' : 'E-Mail'}
				type={domain ? 'text' : 'email'}
				autocomplete={domain ? 'username' : 'email'}
				autocapitalize="none"
				spellcheck="false"
				bind:value={authStore.loginEmail}
				ungueltig={emailFehlt}
				hint={emailHinweis}
			/>
			<Feld
				id="login-password"
				etikett="Passwort"
				type="password"
				autocomplete="current-password"
				bind:value={authStore.loginPassword}
				ungueltig={passwortFehlt}
				hint={passwortFehlt ? 'Bitte dein Passwort eintragen.' : ''}
			/>
		</div>
		<Button type="submit" size="lg" disabled={authStore.isLoggingIn} class="w-full">
			{#if authStore.isLoggingIn}
				<Ladekreis size="sm" farbe="aktuell" />
				Anmelden...
			{:else}
				Anmelden
			{/if}
		</Button>
		{#if authStore.loginError}
			<p role="alert" class="text-xs text-error font-semibold animate-slide-up">
				{authStore.loginError}
			</p>
		{/if}
		{#if authStore.abmeldeHinweis}
			<!-- Bleibt stehen bis zur nächsten Anmeldung: Ein Toast wäre nach Sekunden weg, und
			     der Hinweis gilt, solange die Sitzung am Server noch nicht gesperrt ist. -->
			<p role="status" class="text-xs text-error font-semibold text-center">
				{authStore.abmeldeHinweis}
			</p>
		{/if}
	</form>
</main>
