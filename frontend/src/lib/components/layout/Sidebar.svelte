<script>
	import { authStore } from '../../stores/authStore.svelte.js';
	import { uiStore } from '../../stores/uiStore.svelte.js';
	import { menuGroups, canSeeItem } from '../../menu.js';
	import { sidebarExtensions } from '../../plugins.svelte.js';
	import BackupStatusBadge from '../system/BackupStatusBadge.svelte';
	import { ChevronsLeft, ChevronDown } from '@lucide/svelte';
	import NavIcon from './NavIcon.svelte';
	import SidebarFooter from './SidebarFooter.svelte';
	import logoUrl from '../../../assets/logo.png';

	let systemOpen = $state(false);
	const zu = $derived(uiStore.isSidebarCollapsed);

	function handleLogout() {
		authStore.handleLogout(() => {
			uiStore.activeTab = 'kiosk';
		});
	}

	/** @param {string} id */
	function handleNavigate(id) {
		uiStore.activeTab = id;
		uiStore.selectedBook = null;
	}
</script>

<!-- Ein Navigationsziel. Als Snippet, weil es vorher ZWEIMAL im Markup stand — einmal
     für die System-Gruppe, einmal für alle anderen. Beide Kopien trugen die vollständige
     Icon-Kette; die Datei war dadurch 476 Zeilen lang, und wer eine Farbe änderte, änderte
     sie an einer von zwei Stellen.

     Der ausgewählte Eintrag trägt secondary-container, nicht die Primärfarbe. Das ist
     nicht Geschmack, sondern steht seit dem 04.08.2026 als Regel in styles/rollen.css:
     "In M3 markiert NICHT die Primärfarbe eine Auswahl, sondern der secondary-container —
     Menüeintrag, Navigationsziel, Filterchip." Bis zum 07.08. widersprach ausgerechnet die
     Navigation dieser Regel: Gemessen kam bg-blue-50 als #f7f9ff heraus — von Weiß kaum zu
     unterscheiden. Welcher Punkt aktiv war, sah man praktisch nicht.

     rounded-full statt rounded-xl aus demselben Grund wie beim Button: In M3 ist das
     Navigationsziel eine Pille. -->
{#snippet navItem(/** @type {any} */ item)}
	<button
		onclick={() => handleNavigate(item.id)}
		class="relative flex w-full items-center rounded-full text-sm font-medium transition-colors
			{zu ? 'justify-center px-0 py-2.5' : 'gap-3 px-3 py-2'}
			{uiStore.activeTab === item.id
			? 'bg-secondary-container text-on-secondary-container font-semibold'
			: 'text-on-surface-variant hover:bg-surface-container cursor-pointer'}"
		title={item.label}
		aria-current={uiStore.activeTab === item.id ? 'page' : undefined}
	>
		<NavIcon name={item.icon} />
		{#if !zu}
			<span class="animate-fade-in flex-1 text-left">{item.label}</span>
			{#if item.id === 'orders' && uiStore.pendingReservierungen > 0}
				<span
					class="bg-error text-on-error text-label-small ml-auto flex h-5 min-w-5 items-center justify-center rounded-full px-1 font-bold"
					>{uiStore.pendingReservierungen}</span
				>
			{/if}
		{:else if item.id === 'orders' && uiStore.pendingReservierungen > 0}
			<span class="bg-error absolute top-0.5 right-0.5 h-2.5 w-2.5 rounded-full ring-2 ring-white"
			></span>
		{/if}
	</button>
{/snippet}

<!-- Doppelpfeil zum Ein-/Ausklappen. Ein Snippet, weil er sich nur in Richtung und
     Beschriftung unterscheidet. -->
{#snippet klappPfeil(/** @type {boolean} */ einklappen)}
	<button
		onclick={() => (uiStore.isSidebarCollapsed = einklappen)}
		class="icon-btn text-on-surface-variant hover:bg-surface-container"
		aria-label={einklappen ? 'Navigation einklappen' : 'Navigation ausklappen'}
		data-tip={einklappen ? 'Navigation einklappen' : 'Navigation ausklappen'}
	>
		<ChevronsLeft class="h-4.5 w-4.5 {einklappen ? '' : 'rotate-180'}" aria-hidden="true" />
	</button>
{/snippet}

<aside
	class="border-outline-variant flex shrink-0 flex-col justify-between border-r bg-surface
		no-print h-screen transition-all duration-300 {zu ? 'w-16' : 'w-64'}"
>
	<div class="flex h-full flex-col justify-between overflow-y-auto">
		<div>
			<div
				class="border-outline-variant/60 flex h-16 shrink-0 items-center border-b px-4 {zu
					? 'justify-center'
					: 'justify-between'}"
			>
				{#if !zu}
					<div class="flex items-center gap-3 overflow-hidden">
						<img
							src={logoUrl}
							alt="Logo"
							class="animate-fade-in h-12 w-12 shrink-0 object-contain"
						/>
						<span class="text-on-surface animate-fade-in text-xl font-bold tracking-tight"
							>Bibliosys</span
						>
					</div>
				{/if}
				{@render klappPfeil(!zu)}
			</div>

			<nav class="space-y-6 px-3 py-6">
				{#each menuGroups as group (group.name)}
					{#if group.items.some((item) => canSeeItem(item, authStore.currentUser))}
						<div class="space-y-1">
							{#if group.name === 'System'}
								{#if !zu}
									<button
										onclick={() => (systemOpen = !systemOpen)}
										class="group/sys mb-2 flex w-full cursor-pointer items-center justify-between px-3 text-left"
										aria-expanded={systemOpen}
									>
										<span
											class="text-on-surface-variant/70 group-hover/sys:text-on-surface-variant animate-fade-in text-xs font-medium transition-colors"
											>{group.name}</span
										>
										<ChevronDown
											class="text-on-surface-variant/70 group-hover/sys:text-on-surface-variant h-3.5 w-3.5 transition-transform duration-200 {systemOpen
												? 'rotate-180'
												: ''}"
											aria-hidden="true"
										/>
									</button>
								{/if}
								{#if systemOpen || zu}
									<div class="animate-fade-in space-y-1">
										{#each group.items as item (item.id)}
											{#if canSeeItem(item, authStore.currentUser)}
												{@render navItem(item)}
											{/if}
										{/each}
									</div>
								{/if}
							{:else}
								{#if !zu}
									<span
										class="text-on-surface-variant/70 animate-fade-in mb-2 block px-3 text-xs font-medium"
										>{group.name}</span
									>
								{/if}
								{#each group.items as item (item.id)}
									{#if canSeeItem(item, authStore.currentUser)}
										{@render navItem(item)}
									{/if}
								{/each}
							{/if}
						</div>
					{/if}
				{/each}

				{#if sidebarExtensions.length > 0}
					<div class="border-outline-variant/60 space-y-1 border-t pt-4">
						{#if !zu}
							<span class="text-on-surface-variant/70 mb-2 block px-3 text-xs font-medium"
								>Erweiterungen</span
							>
						{/if}
						{#each sidebarExtensions as ext, _i (_i)}
							{@const Component = ext.component}
							<Component {...ext.props} collapsed={zu} />
						{/each}
					</div>
				{/if}
			</nav>
		</div>

		<div class="border-outline-variant/60 mt-auto border-t">
			<!-- Backup-Wächter: nur Admins können das Problem beheben -->
			{#if authStore.currentUser?.rolle === 'admin'}
				<BackupStatusBadge collapsed={zu} />
			{/if}

			<SidebarFooter collapsed={zu} benutzer={authStore.currentUser} onLogout={handleLogout} />
		</div>
	</div>
</aside>
