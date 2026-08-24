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
	import {
		buecherNachKlassenGruppieren,
		bestandsFarbe
	} from '../../../inventur/lib/startseiten_api.js';

	/** @type {any[]} */
	let buecher = $state.raw([]);
	/** @type {{ className: string, books: any[] }[]} */
	let klassensaetze = $state.raw([]);
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
	<div class="space-y-12">
		<section>
			<h3 class="text-base font-bold text-on-surface">Klassensätze</h3>
			<p class="mt-1 max-w-2xl text-xs leading-relaxed text-on-surface-variant">
				Welche Klasse hat welche Bücher — die Zuordnung pflegt die Bibliothek.
			</p>
			{#if klassensaetze.length === 0}
				<p class="py-4 text-sm text-on-surface-variant">Noch keine Klassensätze zugeordnet.</p>
			{:else}
				<div class="mt-2 divide-y divide-outline-variant">
					{#each klassensaetze as gruppe (gruppe.className)}
						<details class="group py-2">
							<summary
								class="flex cursor-pointer list-none items-center gap-3 rounded-lg py-2 text-left"
							>
								<span
									class="bg-secondary-container text-on-secondary-container flex h-6 min-w-6 shrink-0 items-center justify-center rounded-full px-2 text-xs font-bold tabular-nums"
									>{gruppe.books.length}</span
								>
								<span class="text-sm font-semibold text-on-surface">Klasse {gruppe.className}</span>
							</summary>
							<ul class="divide-y divide-outline-variant pl-9">
								{#each gruppe.books as buch (buch.id)}
									<li class="flex items-baseline justify-between gap-4 py-2">
										<span class="min-w-0">
											<span class="text-sm text-on-surface">{buch.title}</span>
											{#if buch.subject}
												<span class="text-xs text-on-surface-variant"> · {buch.subject}</span>
											{/if}
										</span>
										<span
											class="shrink-0 text-xs font-semibold tabular-nums text-on-surface-variant"
											>{buch.verfuegbar}/{buch.gesamt} verfügbar</span
										>
									</li>
								{/each}
							</ul>
						</details>
					{/each}
				</div>
			{/if}
		</section>

		<section>
			<h3 class="text-base font-bold text-on-surface">Bestand nach Jahrgang</h3>
			<p class="mt-1 max-w-2xl text-xs leading-relaxed text-on-surface-variant">
				Jede Kachel zeigt verfügbar/gesamt. Titel ohne genaue Jahrgangs-Zuordnung sammeln sich in
				der letzten Gruppe.
			</p>
			{#if gruppen.length === 0}
				<p class="py-4 text-sm text-on-surface-variant">Noch keine Bücher im Bestand.</p>
			{:else}
				<div class="mt-2">
					<KlassenUebersichtStartseite filteredClasses={gruppen} getStockColor={bestandsFarbe} />
				</div>
			{/if}
		</section>
	</div>
{/if}
