<script>
	import { onMount } from 'svelte';
	import { apiFetch } from './apiFetch.js';
	import { CircleCheck, TriangleAlert, OctagonAlert, RefreshCw } from '@lucide/svelte';
	import Button from './components/ui/Button.svelte';

	let bericht = $state(/** @type {{ gesamt: string, befunde: any[] } | null} */ (null));
	let laedt = $state(true);
	let fehler = $state('');

	// Die Reihenfolge der Bereiche kommt vom Server. Sortiert wird nur nach Dringlichkeit:
	// Wer diese Seite öffnet, will wissen, was ihn aufhält — nicht, was alles gut ist.
	const RANG = { kritisch: 0, warnung: 1, ok: 2 };
	const sortiert = $derived(
		[...(bericht?.befunde ?? [])].sort((a, b) => RANG[a.stufe] - RANG[b.stufe])
	);
	const offen = $derived(sortiert.filter((b) => b.stufe !== 'ok').length);

	// Farben aus den M3-Rollen (styles/rollen.css), nicht aus der Tailwind-Palette — die
	// Ratsche in frontend-hygiene-farben.test.js laesst nur noch Rollen zu, und sie hat
	// beim ersten Anlauf dieser Datei zu Recht angeschlagen.
	//
	// Fuer „Warnung" gibt es keine eigene Rolle: M3 kennt error, aber kein warning. Sie
	// traegt deshalb dieselbe Farbe wie „kritisch" und unterscheidet sich am SYMBOL —
	// ein zweiter Kanal neben der Farbe ist ohnehin die Vorgabe (WCAG 1.4.1).
	const AUSSEHEN = {
		kritisch: { symbol: OctagonAlert, farbe: 'text-error', flaeche: 'bg-error-container' },
		warnung: { symbol: TriangleAlert, farbe: 'text-error', flaeche: 'bg-surface-container-low' },
		ok: { symbol: CircleCheck, farbe: 'text-primary', flaeche: 'bg-surface' }
	};

	async function laden() {
		laedt = true;
		fehler = '';
		try {
			const res = await apiFetch('/api/admin/system/betriebsbereitschaft');
			if (!res.ok) throw new Error(`Abruf fehlgeschlagen (HTTP ${res.status})`);
			bericht = await res.json();
		} catch (e) {
			fehler = String(e instanceof Error ? e.message : e);
		} finally {
			laedt = false;
		}
	}

	onMount(laden);
</script>

<!-- Seit 16.08.2026 ein Tab der Einstellungen (Betreiber-Entscheidung: schlankeres
     Menue), kein eigener Bildschirm mehr — deshalb KEIN PageShell; die Huelle stellt
     SystemSettings. Der Einleitungssatz erklaert, was diese Ansicht NICHT ist — keine
     Fehlerliste. Sie zeigt, was eingerichtet, aber nicht in Betrieb ist; solche
     Luecken melden sich nie von selbst. -->
<div class="space-y-6">
	<div class="flex items-start justify-between gap-4">
		<p class="max-w-2xl text-sm text-on-surface-variant">
			Was ist eingerichtet, aber nicht in Betrieb? Funktionen, die fertig sind und nichts tun, weil
			eine Einstellung fehlt, melden sich nirgends — hier stehen sie.
		</p>
		<Button variant="secondary" onclick={laden} disabled={laedt}>
			<RefreshCw class="h-4 w-4 {laedt ? 'animate-spin' : ''}" aria-hidden="true" />
			Neu prüfen
		</Button>
	</div>

	{#if fehler}
		<p class="rounded-xl bg-error-container px-4 py-3 text-sm text-on-error-container">{fehler}</p>
	{:else if laedt && !bericht}
		<p class="py-16 text-center text-sm text-on-surface-variant">Wird geprüft …</p>
	{:else if bericht}
		<p class="text-sm font-medium text-on-surface-variant">
			{#if offen === 0}
				Alles eingerichtet — nichts liegt still.
			{:else}
				{offen}
				{offen === 1 ? 'Punkt' : 'Punkte'} offen.
			{/if}
		</p>

		<div class="divide-y divide-outline-variant border-y border-outline-variant">
			{#each sortiert as b (b.bereich)}
				{@const stil = AUSSEHEN[b.stufe] ?? AUSSEHEN.ok}
				{@const Symbol = stil.symbol}
				<div class="flex gap-4 py-4 {stil.flaeche}">
					<Symbol class="mt-0.5 h-5 w-5 shrink-0 {stil.farbe}" aria-hidden="true" />
					<div class="min-w-0">
						<p class="font-semibold text-on-surface">{b.bereich}</p>
						<p class="mt-0.5 text-sm text-on-surface">{b.befund}</p>
						{#if b.folge}
							<p class="mt-1 text-sm text-on-surface-variant">{b.folge}</p>
						{/if}
						{#if b.abhilfe}
							<p class="mt-2 text-sm font-medium text-primary">{b.abhilfe}</p>
						{/if}
					</div>
				</div>
			{/each}
		</div>
	{/if}
</div>
