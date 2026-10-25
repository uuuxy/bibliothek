<script>
	/**
	 * @component PortalLernmittel
	 * Die Nur-Lese-Sicht des Kollegiums auf die Lernmittel: Bestand und Mengen je Jahrgang
	 * plus die Klassensatz-Zuordnung, damit Lehrkräfte sehen, was da ist und wie viel.
	 *
	 * Hinter der Anmeldung statt im öffentlichen OPAC: Der OPAC schließt
	 * Schulbücher aus (Suchtreffer-Flut, api/portal/opac.go), und die Bestandszahlen der
	 * Schule gehören nicht offen ins Netz. Die Daten kommen über die Portal-Türen
	 * /api/portal/klassensaetze (Anmeldung genügt, kein
	 * view_books — das Recht würde der Rolle den ganzen Medienkatalog öffnen).
	 */
	import { onMount } from 'svelte';
	import Ladekreis from '../ui/Ladekreis.svelte';
	import { apiFetch } from '../../apiFetch.js';
	import KlassenKarte from '../../../inventur/lib/components/admin/KlassenKarte.svelte';

	/** @type {{ className: string, books: any[] }[]} */
	let klassensaetze = $state.raw([]);
	/** Immer nur eine Klasse offen — wie unter Bibliothek → Klassensätze. @type {string|null} */
	let offeneKlasse = $state(null);
	let laedt = $state(true);
	let ladefehler = $state(false);

	onMount(async () => {
		try {
			const k = await apiFetch('/api/portal/klassensaetze');
			if (k.ok) klassensaetze = await k.json();
			else ladefehler = true;
		} catch {
			// Sonst sagte der leere Zustand unten „Noch keine Klassensätze zugeordnet", auch wenn
			// der Abruf gescheitert ist: keine fehlende Meldung, sondern eine falsche Auskunft.
			ladefehler = true;
		} finally {
			laedt = false;
		}
	});
</script>

{#if laedt}
	<div class="flex justify-center py-16">
		<Ladekreis size="lg" />
	</div>
{:else}
	<!-- Ein Reiter = eine Liste. Der Reiter ist die Überschrift; Beitexte wären eine dritte
	     Ebene unter Reiter und Abschnitt. -->
	<div class="pt-2">
		{#if ladefehler}
			<p class="py-4 text-sm font-semibold text-error" role="alert">
				Die Klassensätze konnten nicht geladen werden. Bitte die Seite neu laden — das hier bedeutet
				NICHT, dass keine zugeordnet sind.
			</p>
		{:else if klassensaetze.length === 0}
			<p class="py-4 text-sm text-on-surface-variant">Noch keine Klassensätze zugeordnet.</p>
		{:else}
			<!-- Dieselbe Karte wie unter Bibliothek → Klassensätze, nur lesend:
			     /api/portal/klassensaetze ist derselbe Handler wie /api/class-books, also auch
			     dieselbe Darstellung. -->
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
