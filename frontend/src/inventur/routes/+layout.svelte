<script>
	import "../app.css";
	import { appState, logout } from "$lib/store.svelte.js";
	import Toast from "$lib/components/Toast.svelte";

	let { children } = $props();

	// Fallback pathname for browser-only execution
	let pathname = $derived(typeof window !== "undefined" ? window.location.pathname : "/");
</script>

<div class="min-h-screen bg-slate-50">
	<header class="flex justify-between items-center px-6 py-4 bg-white/80 backdrop-blur-md border-b border-slate-200 sticky top-0 z-10 shadow-xs">
		<a
			href={pathname.startsWith("/admin") && appState.adminAuthenticated ? "/admin" : "/"}
			class="no-underline font-bold text-slate-800 text-lg"
		>
			{#if pathname.startsWith("/admin") && appState.adminAuthenticated}
				Admin Dashboard
			{:else}
				📚 Schulbuch-Inventar
			{/if}
		</a>
		<nav class="flex gap-4 items-center">
			<a href="/" class="no-underline text-slate-600 hover:text-blue-600 font-semibold transition-colors">Startseite</a>
			<a href="/admin" class="no-underline text-slate-600 hover:text-blue-600 font-semibold transition-colors">Admin</a>
			{#if (pathname.startsWith("/admin") && appState.adminAuthenticated) || (!pathname.startsWith("/admin") && appState.guestAuthenticated)}
				<button onclick={logout} class="bg-red-50 hover:bg-red-100 text-red-650 border border-red-200 px-4 py-2 rounded-lg cursor-pointer text-sm font-medium transition-colors"> Abmelden </button>
			{/if}
		</nav>
	</header>

	<main class="p-5">{@render children?.()}</main>
	<Toast />
</div>
