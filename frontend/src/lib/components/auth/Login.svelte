<script>
	import { authStore } from '../../stores/authStore.svelte.js';
	import Button from '../ui/Button.svelte';
	import logoUrl from '../../../assets/logo.png';

	$effect(() => {
		setTimeout(() => document.getElementById('login-email')?.focus(), 50);
	});
</script>

<div class="min-h-screen flex items-center justify-center p-6 bg-slate-50">
	<form
		onsubmit={(e) => authStore.handleLogin(e, undefined)}
		class="w-full max-w-md p-8 rounded-3xl bg-white border border-slate-100 shadow-xl flex flex-col items-center space-y-6 animate-fade-in no-print"
	>
		<img
			src={logoUrl}
			alt="Bibliosys Logo"
			class="w-24 h-24 object-contain"
		/>
		<div class="text-center space-y-1.5">
			<h2 class="text-base font-bold text-slate-800">Webmail-Login erforderlich</h2>
			<p class="text-xs text-slate-400 font-medium">
				Bitte logge dich mit deiner Schul-E-Mail ein.
			</p>
		</div>
		<div class="w-full space-y-3">
			<input
				id="login-email"
				type="email"
				autocomplete="email"
				bind:value={authStore.loginEmail}
				class="w-full bg-slate-50 border border-slate-200 rounded-xl py-3 px-4 text-slate-800 focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-300 transition-all"
				placeholder="name@philipp-reis-schule.de"
			/>
			<input
				id="login-password"
				type="password"
				autocomplete="current-password"
				bind:value={authStore.loginPassword}
				class="w-full bg-slate-50 border border-slate-200 rounded-xl py-3 px-4 text-slate-800 focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-300 transition-all"
				placeholder="Passwort"
			/>
		</div>
		<Button type="submit" size="lg" disabled={authStore.isLoggingIn} class="w-full">
			{#if authStore.isLoggingIn}
				<div
					class="w-4 h-4 border-2 border-white/40 border-t-white rounded-full animate-spin"
				></div>
				Anmelden...
			{:else}
				Anmelden
			{/if}
		</Button>
		{#if authStore.loginError}
			<p class="text-xs text-rose-500 font-semibold animate-slide-up">{authStore.loginError}</p>
		{/if}
	</form>
</div>
