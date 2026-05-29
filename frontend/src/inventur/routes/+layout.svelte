<script>
	import "../app.css";
	import { resolve } from "$app/paths";
	import { page } from "$app/stores";
	import { appState, logout } from "$lib/store.svelte.js";
	import Toast from "$lib/components/Toast.svelte";

	let { children } = $props();
</script>

<div class="app-shell">
	<header class="topbar">
		<a
			href={resolve(
				$page.url.pathname.startsWith("/admin") &&
					appState.adminAuthenticated
					? "/admin"
					: "/",
			)}
			class="brand"
		>
			{#if $page.url.pathname.startsWith("/admin") && appState.adminAuthenticated}
				Admin Dashboard
			{:else}
				📚 Schulbuch-Inventar
			{/if}
		</a>
		<nav class="nav">
			<a href={resolve("/")}>Startseite</a>
			<a href={resolve("/admin")}>Admin</a>
			{#if ($page.url.pathname.startsWith("/admin") && appState.adminAuthenticated) || (!$page.url.pathname.startsWith("/admin") && appState.guestAuthenticated)}
				<button onclick={logout} class="logout-btn"> Abmelden </button>
			{/if}
		</nav>
	</header>

	<main>{@render children?.()}</main>
	<Toast />
</div>

<style>
	:global(body) {
		margin: 0;
		font-family:
			Inter,
			system-ui,
			-apple-system,
			Segoe UI,
			sans-serif;
		background: linear-gradient(
			180deg,
			var(--md3-surface),
			var(--md3-surface-container)
		);
		color: var(--md3-on-surface);
	}

	.app-shell {
		min-height: 100vh;
	}

	.topbar {
		display: flex;
		justify-content: space-between;
		align-items: center;
		padding: 1rem 1.5rem;
		background: color-mix(in srgb, var(--md3-surface) 85%, transparent);
		backdrop-filter: blur(8px);
		border-bottom: 1px solid var(--md3-outline);
		position: sticky;
		top: 0;
		z-index: 10;
	}

	.brand {
		text-decoration: none;
		font-weight: 700;
		color: var(--md3-on-surface);
	}

	.nav {
		display: flex;
		gap: 1rem;
		align-items: center;
	}

	.logout-btn {
		background: #fee2e2;
		color: #dc2626;
		border: none;
		padding: 0.5rem 1rem;
		border-radius: 6px;
		cursor: pointer;
		font-size: 0.9rem;
		font-weight: 500;
		transition: background-color 0.2s;
	}

	.logout-btn:hover {
		background: #fecaca;
	}

	.nav a {
		text-decoration: none;
		color: var(--md3-on-surface-variant);
		font-weight: 500;
	}

	.nav a:hover {
		color: var(--md3-primary);
	}

	main {
		padding: 1.25rem;
	}
</style>
