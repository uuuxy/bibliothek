<script>
	import "../app.css";
	import { appState, logout } from "$lib/store.svelte.js";
	import Toast from "$lib/components/Toast.svelte";

	let { children } = $props();

	// Fallback pathname for browser-only execution
	let pathname = $derived(typeof window !== "undefined" ? window.location.pathname : "/");
</script>

<div class="min-h-screen">
	<header class="flex justify-between items-center px-6 py-4 bg-slate-900/85 backdrop-blur-md border-b border-zinc-800 sticky top-0 z-10">
		<a
			href={pathname.startsWith("/admin") && appState.adminAuthenticated ? "/admin" : "/"}
			class="no-underline font-bold text-zinc-100 text-lg"
		>
			{#if pathname.startsWith("/admin") && appState.adminAuthenticated}
				Admin Dashboard
			{:else}
				📚 Schulbuch-Inventar
			{/if}
		</a>
		<nav class="flex gap-4 items-center">
			<a href="/" class="no-underline text-zinc-300 hover:text-emerald-400 font-medium transition-colors">Startseite</a>
			<a href="/admin" class="no-underline text-zinc-300 hover:text-emerald-400 font-medium transition-colors">Admin</a>
			{#if (pathname.startsWith("/admin") && appState.adminAuthenticated) || (!pathname.startsWith("/admin") && appState.guestAuthenticated)}
				<button onclick={logout} class="bg-red-500/10 hover:bg-red-500/20 text-red-400 border border-red-500/20 px-4 py-2 rounded-lg cursor-pointer text-sm font-medium transition-colors"> Abmelden </button>
			{/if}
		</nav>
	</header>

	<main class="p-5">{@render children?.()}</main>
	<Toast />
</div>
