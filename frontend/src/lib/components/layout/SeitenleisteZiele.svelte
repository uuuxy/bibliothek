<!-- @component SeitenleisteZiele — die Navigationsziele der Seitenleiste, nach Gruppen.

     Eigene Datei: Die Seitenleiste trägt Breite, Aufklappen und Fuß, diese Datei die Ziele.
     `zu` heißt eingeklappt gewählt, `schmal` heißt zu sehen sind nur die Symbole; über dem
     Inhalt aufgeklappt ist die Leiste zu, aber nicht schmal. `nachWahl` klappt sie dann
     wieder ein: Die Maus steht nach dem Klick noch auf ihr, der Inhalt soll frei sein. -->
<script>
	import { authStore } from '../../stores/authStore.svelte.js';
	import { uiStore } from '../../stores/uiStore.svelte.js';
	import { menueGruppen, darfSehen } from '../../menue.js';
	import { ChevronDown } from '@lucide/svelte';
	import NavIcon from './NavIcon.svelte';
	import Zaehlerpille from '../ui/Zaehlerpille.svelte';

	/** @type {{ zu: boolean, schmal: boolean, nachWahl: () => void }} */
	let { zu, schmal, nachWahl } = $props();

	let systemOpen = $state(false);

	/** @param {string} id */
	function navigiere(id) {
		uiStore.activeTab = id;
		uiStore.selectedBook = null;
		nachWahl();
	}

	/**
	 * Wartende Arbeit je Navigationsziel. In M3 ist genau das die Aufgabe eines Badges: am
	 * Ziel anzeigen, dass dort etwas liegt. Ein Banner, wie M2 es hatte, kennt M3 nicht mehr.
	 * @param {string} id
	 */
	function wartendeArbeit(id) {
		if (id === 'orders') return uiStore.pendingReservierungen + uiStore.offeneAnliegen; // + Anliegen
		if (id === 'druck-center') return uiStore.offeneEtiketten;
		return 0;
	}
</script>

<!-- Ein Navigationsziel. Als Snippet, weil die System-Gruppe und alle anderen Gruppen
     dasselbe Markup brauchen.

     Der ausgewählte Eintrag trägt secondary-container, nicht die Primärfarbe: In M3
     markiert der secondary-container eine Auswahl — Menüeintrag, Navigationsziel,
     Filterchip (styles/rollen.css).

     rounded-full statt rounded-xl aus demselben Grund wie beim Button: In M3 ist das
     Navigationsziel eine Pille. Über dem Inhalt aufgeklappt behält der Eintrag Höhe und
     Symbol der schmalen Leiste, damit unter der Maus nichts verrutscht. -->
{#snippet navItem(/** @type {any} */ item)}
	<button
		onclick={() => navigiere(item.id)}
		class="relative flex w-full items-center rounded-full text-sm font-medium transition-colors
			{schmal ? 'justify-center px-0 py-2.5' : zu ? 'gap-3 pr-3 pl-2.5 py-2.5' : 'gap-3 px-3 py-2'}
			{uiStore.activeTab === item.id
			? 'bg-secondary-container text-on-secondary-container font-semibold'
			: 'text-on-surface-variant hover:bg-surface-container cursor-pointer'}"
		title={item.label}
		aria-current={uiStore.activeTab === item.id ? 'page' : undefined}
	>
		<NavIcon name={item.icon} />
		{#if !schmal}
			<span class="animate-fade-in flex-1 whitespace-nowrap text-left">{item.label}</span>
			<Zaehlerpille anzahl={wartendeArbeit(item.id)} klasse="ml-auto" />
			<!-- Eingeklappt bleibt nur der Punkt: M3 nennt das „small badge" — er sagt „hier
			     liegt etwas", ohne dass eine Zahl in eine 40-px-Pille gequetscht wird. -->
		{:else if wartendeArbeit(item.id) > 0}
			<span class="bg-error absolute top-0.5 right-0.5 h-2.5 w-2.5 rounded-full ring-2 ring-surface"
			></span>
		{/if}
	</button>
{/snippet}

<!-- Gruppenüberschriften und die aufklappbare Gruppe „System" nur bei ausgeklappter Leiste:
     Über dem Inhalt aufgeklappt rückten sie die Ziele unter der Maus nach unten. -->
<nav class="space-y-6 px-3 py-6">
	{#each menueGruppen as group (group.name)}
		{#if group.items.some((item) => darfSehen(item, authStore.currentUser))}
			<div class="space-y-1">
				{#if group.name === 'System'}
					{#if !zu}
						<button
							onclick={() => (systemOpen = !systemOpen)}
							class="group/sys mb-2 flex w-full cursor-pointer items-center justify-between px-3 text-left"
							aria-expanded={systemOpen}
							aria-controls="seitenleiste-system"
						>
							<span
								class="text-on-surface-variant group-hover/sys:text-on-surface animate-fade-in text-sm font-medium transition-colors"
								>{group.name}</span
							>
							<ChevronDown
								class="text-on-surface-variant group-hover/sys:text-on-surface h-3.5 w-3.5 transition-transform duration-200 {systemOpen
									? 'rotate-180'
									: ''}"
								aria-hidden="true"
							/>
						</button>
					{/if}
					{#if systemOpen || zu}
						<div id="seitenleiste-system" class="animate-fade-in space-y-1">
							{#each group.items as item (item.id)}
								{#if darfSehen(item, authStore.currentUser)}
									{@render navItem(item)}
								{/if}
							{/each}
						</div>
					{/if}
				{:else}
					{#if !zu}
						<!-- Sektions-Überschrift in der M3-Form: title-small auf on-surface-variant
						     (Material Components Android, NavigationView: `subheaderTextAppearance` =
						     `textAppearanceTitleSmall`, `subheaderColor` = `colorOnSurfaceVariant`). -->
						<span
							class="text-on-surface-variant animate-fade-in mb-2 block px-3 text-sm font-medium"
							>{group.name}</span
						>
					{/if}
					{#each group.items as item (item.id)}
						{#if darfSehen(item, authStore.currentUser)}
							{@render navItem(item)}
						{/if}
					{/each}
				{/if}
			</div>
		{/if}
	{/each}
</nav>
