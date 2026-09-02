<!-- @component ThekenUebersicht — der Ruhezustand der Theke: Marken-Relief plus vier
     Zahlen, die den Tag beschreiben (überfällig, Abholfach, wartende Klassensätze,
     offene Anliegen). Bis zum 02.09.2026 zeigte der Startbildschirm vor dem ersten
     Scan nur Logo und Feld.

     Nur ZAHLEN: Der Bildschirm steht am Tresen und ist für Schüler einsehbar — wer
     säumig ist, steht im Mahnwesen hinter dem Klick (thekenUebersicht.js).
     Jede Kachel hängt am Recht ihrer Route; Klassensätze und Anliegen kommen aus dem
     uiStore (dieselben Zähler wie die Badges, alle 30 s frisch), Überfällige und
     Abholfach werden hier geholt — und nach jedem Theken-Vorgang ('action' auf der
     gemeinsamen SSE-Leitung) entprellt nachgezogen.

     Absolut unter dem Suchfeld platziert, weil die Theke ihr Feld nie verschiebt
     (Omnibox.svelte) und dieser Block im DOM VOR dem Feld steht. -->
<script>
	import { onMount } from 'svelte';
	import { AlarmClock, PackageCheck, BookCopy, MessageSquare } from '@lucide/svelte';
	import LogoRelief from './ui/LogoRelief.svelte';
	import { apiFetch } from '../apiFetch.js';
	import { abonniere } from '../liveEvents.js';
	import { hatRecht } from '../menu.js';
	import { authStore } from '../stores/authStore.svelte.js';
	import { uiStore } from '../stores/uiStore.svelte.js';
	import { sichtbareKacheln, zaehleAbholbereit, zaehleUeberfaellig } from '../thekenUebersicht.js';

	const SYMBOLE = /** @type {Record<string, any>} */ ({
		ueberfaellig: AlarmClock,
		abholbereit: PackageCheck,
		klassensaetze: BookCopy,
		anliegen: MessageSquare
	});

	let ueberfaellig = $state(0);
	let abholbereit = $state(0);

	const kacheln = $derived(sichtbareKacheln((r) => hatRecht(authStore.currentUser, r)));

	/** @param {string} id */
	function wertVon(id) {
		if (id === 'ueberfaellig') return ueberfaellig;
		if (id === 'abholbereit') return abholbereit;
		if (id === 'klassensaetze') return uiStore.pendingReservierungen;
		return uiStore.offeneAnliegen;
	}

	async function laden() {
		const darf = (/** @type {string} */ r) => hatRecht(authStore.currentUser, r);
		try {
			if (darf('view_students')) {
				const res = await apiFetch('/api/dashboard/summary');
				if (res.ok) ueberfaellig = zaehleUeberfaellig(await res.json());
			}
			if (darf('manage_vormerkungen')) {
				const res = await apiFetch('/api/vormerkungen');
				if (res.ok) abholbereit = zaehleAbholbereit(await res.json());
			}
		} catch {
			/* Eine fehlende Zahl darf die Theke nicht aufhalten. */
		}
	}

	onMount(() => {
		laden();
		/** @type {ReturnType<typeof setTimeout> | null} */
		let timer = null;
		const abmelden = abonniere('action', () => {
			if (timer) clearTimeout(timer);
			timer = setTimeout(laden, 1500);
		});
		return () => {
			abmelden();
			if (timer) clearTimeout(timer);
		};
	});

	/** @param {string | null} ziel */
	function oeffne(ziel) {
		if (ziel) uiStore.activeTab = ziel;
	}

	const KACHEL =
		'flex w-60 items-center gap-3 rounded-2xl bg-surface-container-low px-4 py-3 text-left';
</script>

{#snippet inhalt(/** @type {import('../thekenUebersicht.js').Kachel} */ k)}
	{@const Symbol = SYMBOLE[k.id]}
	{@const wert = wertVon(k.id)}
	{@const warnend = k.id === 'ueberfaellig' && wert > 0}
	<Symbol
		class="h-5 w-5 shrink-0 {warnend ? 'text-error' : 'text-on-surface-variant'}"
		aria-hidden="true"
	/>
	<span class="flex flex-col leading-tight">
		<span class="text-2xl font-medium {warnend ? 'text-error' : 'text-on-surface'}">{wert}</span>
		<span class="text-sm text-on-surface-variant">{k.label}</span>
	</span>
{/snippet}

<LogoRelief />

{#if kacheln.length > 0}
	<div
		class="absolute inset-x-0 top-28 z-20 flex justify-center px-4"
		aria-label="Heute an der Theke"
	>
		<div class="flex flex-wrap justify-center gap-3">
			{#each kacheln as k (k.id)}
				{#if k.ziel}
					<button
						type="button"
						onclick={() => oeffne(k.ziel)}
						class="{KACHEL} m3-state cursor-pointer"
						title="Öffnen"
					>
						{@render inhalt(k)}
					</button>
				{:else}
					<div class={KACHEL}>{@render inhalt(k)}</div>
				{/if}
			{/each}
		</div>
	</div>
{/if}
