<script>
	import { onMount } from 'svelte';
	import { apiFetch, leseApiFehler } from '../../apiFetch.js';
	import { CircleCheck, TriangleAlert, OctagonAlert, RefreshCw } from '@lucide/svelte';
	import Button from '../ui/Button.svelte';
	import Ladekreis from '../ui/Ladekreis.svelte';
	import LadeFehler from '../ui/LadeFehler.svelte';
	import { fehlertext } from '../../utils/fehlertext.js';

	/** @typedef {{ text: string, zusatz?: string }} FehlerZeile */
	/** @typedef {{ stufe: string, anzahl: number, befund: string, meldungen: FehlerZeile[], folge?: string, abhilfe?: string }} FehlerImLog */

	let stand = $state(/** @type {FehlerImLog | null} */ (null));
	let laedt = $state(true);
	let fehler = $state('');

	// Dieselben Zeichen wie in der Betriebsbereitschaft: Farben aus den M3-Rollen, und weil M3 kein
	// warning kennt, unterscheidet sich die Warnung am Symbol (WCAG 1.4.1).
	const ZEICHEN = {
		kritisch: { symbol: OctagonAlert, farbe: 'text-error' },
		warnung: { symbol: TriangleAlert, farbe: 'text-error' },
		ok: { symbol: CircleCheck, farbe: 'text-primary' }
	};

	async function laden() {
		laedt = true;
		fehler = '';
		try {
			const res = await apiFetch('/api/admin/system/fehler-im-log');
			if (!res.ok)
				throw new Error(await leseApiFehler(res, `Abruf fehlgeschlagen (HTTP ${res.status})`));
			stand = await res.json();
		} catch (e) {
			fehler = fehlertext(e);
		} finally {
			laedt = false;
		}
	}

	onMount(laden);
</script>

<!-- Eine Kategorie der Einstellungen direkt unter der Betriebsbereitschaft, kein eigener
     Bildschirm; Überschrift und Satz stellt KategorieRahmen. Die Betriebsbereitschaft nennt
     Zustände, diese Seite Ereignisse. -->
<div class="space-y-6">
	<div class="flex items-start justify-between gap-4">
		{#if stand}
			{@const zeichen = ZEICHEN[stand.stufe] ?? ZEICHEN.ok}
			{@const Symbol = zeichen.symbol}
			<p class="flex items-start gap-3 text-sm font-medium text-on-surface">
				<Symbol class="h-5 w-5 shrink-0 {zeichen.farbe}" aria-hidden="true" />
				{stand.befund}
			</p>
		{:else}
			<span></span>
		{/if}
		<Button variant="secondary" onclick={laden} disabled={laedt} class="shrink-0">
			<RefreshCw class="h-4 w-4 {laedt ? 'animate-spin' : ''}" aria-hidden="true" />
			Neu laden
		</Button>
	</div>

	{#if fehler}
		<LadeFehler onerneut={laden} titel="Fehler im Log nicht geladen" text={fehler} />
	{:else if laedt && !stand}
		<div class="flex justify-center py-20">
			<Ladekreis size="lg" etikett="Fehler im Log lädt" />
		</div>
	{:else if stand}
		{#if stand.meldungen.length}
			<!-- Eine Liste nach M3: je Meldung die Beschriftung und darunter der Zusatztext mit
			     Anzahl, letzter Uhrzeit und Ursache. Den Wortlaut baut der Server, die Mail zeigt
			     denselben. -->
			<ul class="divide-y divide-outline-variant border-y border-outline-variant">
				{#each stand.meldungen as m (m.text)}
					<li class="py-3">
						<p class="text-base text-on-surface">{m.text}</p>
						{#if m.zusatz}
							<p class="mt-0.5 text-sm wrap-break-word text-on-surface-variant">{m.zusatz}</p>
						{/if}
					</li>
				{/each}
			</ul>
		{/if}
		{#if stand.folge}
			<p class="text-sm text-on-surface-variant">{stand.folge}</p>
		{/if}
		{#if stand.abhilfe}
			<p class="text-sm font-medium text-primary">{stand.abhilfe}</p>
		{/if}
	{/if}
</div>
