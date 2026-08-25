<script>
	/**
	 * @component PortalLernmittel
	 * Die Nur-Lese-Sicht des Kollegiums auf die Lernmittel (Betreiber-Entscheidung
	 * 24.08.2026): Bestand und Mengen je Jahrgang plus die Klassensatz-Zuordnung —
	 * das ursprüngliche Ziel „Lehrkräfte sehen von außen, was da ist und wie viel".
	 *
	 * Bewusst hinter der Anmeldung statt im öffentlichen OPAC: Der OPAC schließt
	 * Schulbücher aus (Suchtreffer-Flut, api/opac.go), und die Bestandszahlen der
	 * Schule gehören nicht offen ins Netz. Die Daten kommen über die Portal-Türen
	 * /api/portal/lernmittel und /api/portal/klassensaetze (Anmeldung genügt, kein
	 * view_books — das Recht würde der Rolle den ganzen Medienkatalog öffnen).
	 */
	import { onMount } from 'svelte';
	import { apiFetch } from '../../apiFetch.js';

	import KlassenUebersichtStartseite from '../../../inventur/lib/components/KlassenUebersichtStartseite.svelte';
	import KlassenKarte from '../../../inventur/lib/components/admin/KlassenKarte.svelte';
	import {
		buecherNachKlassenGruppieren,
		bestandsFarbe
	} from '../../../inventur/lib/startseiten_api.js';

	/** @type {{ bereich: 'klassensaetze' | 'jahrgang' }} */
	let { bereich } = $props();

	/** @type {any[]} */
	let buecher = $state.raw([]);
	/** @type {{ className: string, books: any[] }[]} */
	let klassensaetze = $state.raw([]);
	/** Immer nur eine Klasse offen — wie unter Bibliothek → Klassensätze. @type {string|null} */
	let offeneKlasse = $state(null);
	let laedt = $state(true);

	const gruppen = $derived(buecherNachKlassenGruppieren(buecher));

	onMount(async () => {
		try {
			const [b, k] = await Promise.all([
				apiFetch('/api/portal/lernmittel'),
				apiFetch('/api/portal/klassensaetze')
			]);
			if (b.ok) buecher = (await b.json()).data ?? [];
			if (k.ok) klassensaetze = (await k.json()).data ?? [];
		} catch {
			/* Die leeren Zustände unten sagen, dass nichts da ist. */
		} finally {
			laedt = false;
		}
	});
</script>

{#if laedt}
	<div class="flex justify-center py-16">
		<div
			class="h-8 w-8 animate-spin rounded-full border-4 border-primary border-t-transparent"
		></div>
	</div>
{:else}
	<!-- Ein Reiter = eine Liste. Der Reiter ist die Überschrift; Beitexte standen hier bis
	     25.08.2026 als dritte Ebene unter Reiter und Abschnitt. -->
	<div class="pt-2">
		{#if bereich === 'klassensaetze'}
			{#if klassensaetze.length === 0}
				<p class="py-4 text-sm text-on-surface-variant">Noch keine Klassensätze zugeordnet.</p>
			{:else}
				<!-- Dieselbe Karte wie unter Bibliothek → Klassensätze, nur lesend (Peter,
				     25.08.2026: „warum zeigen wir hier nicht einfach die Übersicht der
				     Klassensätze?"). /api/portal/klassensaetze ist derselbe Handler wie
				     /api/class-books — die Daten waren schon gleich, nur die Darstellung nicht. -->
				<div>
					{#each klassensaetze as gruppe (gruppe.className)}
						<KlassenKarte
							group={gruppe}
							kompakt
							offen={offeneKlasse === gruppe.className}
							onToggle={() =>
								(offeneKlasse = offeneKlasse === gruppe.className ? null : gruppe.className)}
						/>
					{/each}
				</div>
			{/if}
		{:else if gruppen.length === 0}
			<p class="py-4 text-sm text-on-surface-variant">Noch keine Bücher im Bestand.</p>
		{:else}
			<!-- Dieselbe Aufklapp-Liste wie im Medienkatalog, aber kompakt: Dort ist der
			     Jahrgang eine Seitenüberschrift (24 px), hier eine Listenzeile unter dem Reiter. -->
			<KlassenUebersichtStartseite
				filteredClasses={gruppen}
				getStockColor={bestandsFarbe}
				kompakt
			/>
		{/if}
	</div>
{/if}
