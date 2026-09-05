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
	 * /api/portal/klassensaetze (Anmeldung genügt, kein
	 * view_books — das Recht würde der Rolle den ganzen Medienkatalog öffnen).
	 */
	import { onMount } from 'svelte';
	import { apiFetch } from '../../apiFetch.js';
	import KlassenKarte from '../../../inventur/lib/components/admin/KlassenKarte.svelte';

	/** @type {{ className: string, books: any[] }[]} */
	let klassensaetze = $state.raw([]);
	/** Immer nur eine Klasse offen — wie unter Bibliothek → Klassensätze. @type {string|null} */
	let offeneKlasse = $state(null);
	let laedt = $state(true);

	onMount(async () => {
		try {
			const k = await apiFetch('/api/portal/klassensaetze');
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
						offen={offeneKlasse === gruppe.className}
						onToggle={() =>
							(offeneKlasse = offeneKlasse === gruppe.className ? null : gruppe.className)}
					/>
				{/each}
			</div>
		{/if}
	</div>
{/if}
