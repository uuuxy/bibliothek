<script>
	import { mahnwesenStore } from '../../stores/mahnwesen.svelte.js';
	import Tabelle from '../ui/Tabelle.svelte';
	import Ladekreis from '../ui/Ladekreis.svelte';
	import MahnlisteMailDialog from './MahnlisteMailDialog.svelte';
	import { uiStore } from '../../stores/uiStore.svelte.js';
	import { Mail } from '@lucide/svelte';
	import Kaestchen from '../ui/Kaestchen.svelte';

	/** Öffnet das Profil des überfälligen Schülers in der Schülerdatei (zentraler Request). */
	function openProfile(schuelerId) {
		uiStore.requestedStudentId = schuelerId;
		uiStore.activeTab = 'students_dir';
	}

	// Derived state for 'Select All' checkbox
	let allSelected = $derived(
		mahnwesenStore.filteredSchueler.length > 0 &&
			mahnwesenStore.selectedIds.size === mahnwesenStore.filteredSchueler.length
	);

	let indeterminate = $derived(
		mahnwesenStore.selectedIds.size > 0 &&
			mahnwesenStore.selectedIds.size < mahnwesenStore.filteredSchueler.length
	);

	// Toggle all
	function toggleAll() {
		if (allSelected) mahnwesenStore.deselectAllSchueler();
		else mahnwesenStore.selectAllSchueler();
	}
</script>

{#if mahnwesenStore.loading}
	<div class="flex justify-center py-20">
		<Ladekreis size="lg" />
	</div>
{:else if mahnwesenStore.error}
	<div
		class="bg-rose-50 border border-rose-200 rounded-2xl p-6 text-center text-rose-600 text-sm font-medium"
	>
		{mahnwesenStore.error}
	</div>
{:else if !mahnwesenStore.data || mahnwesenStore.klassen.length === 0}
	<div class="bg-emerald-50 border border-emerald-200 rounded-2xl p-10 text-center">
		<p class="text-emerald-700 font-semibold">Keine überfälligen Ausleihen vorhanden.</p>
	</div>
{:else}
	<!-- Kein Kartenrahmen. Die Kontur kam mit 160e298 dazu, weil der Arbeitsbereich
	     damals grau war und die Tabelle sich sonst nicht abgehoben hätte. a4133a2 hat
	     die getönte Leinwand wieder zurückgenommen — damit ist der Grund entfallen, der
	     Rahmen aber stehen geblieben. Er war zuletzt die einzige Tabelle im Haus mit
	     Kasten: Schülerdatei, Abgänger, Medienkatalog und Inventur stehen alle
	     edge-to-edge. Getrennt wird über die Kopfzeile, nicht über eine Umrandung. -->
	<div class="w-full pb-6">
		<div class="overflow-x-auto w-full">
			<Tabelle class="whitespace-nowrap">
				<thead class="font-medium">
					<tr>
						<th class="w-12 text-center">
							<Kaestchen
								checked={allSelected}
								{indeterminate}
								onclick={toggleAll}
								aria-label="Alle überfälligen Schüler auswählen"
							/>
						</th>
						<th>Schüler/in</th>
						<th>Klasse</th>
						<th>Medien</th>
						<th>Status</th>
					</tr>
				</thead>
				<tbody>
					{#each mahnwesenStore.filteredSchueler as schueler, _i (_i)}
						<tr aria-selected={mahnwesenStore.selectedIds.has(schueler.schueler_id)}>
							<td class="w-12 text-center">
								<Kaestchen
									checked={mahnwesenStore.selectedIds.has(schueler.schueler_id)}
									onclick={() => mahnwesenStore.toggleSelect(schueler.schueler_id)}
									aria-label="{schueler.name} auswählen"
								/>
							</td>
							<td>
								<div class="flex items-center gap-1.5">
									<button
										type="button"
										onclick={() => openProfile(schueler.schueler_id)}
										class="font-semibold text-slate-800 text-left hover:text-blue-700 hover:underline cursor-pointer rounded focus-visible:outline-2 focus-visible:outline-blue-600"
										aria-label="Profil von {schueler.name} anzeigen"
									>
										{schueler.name}
									</button>
									{#if !schueler.eltern_email}
										<!-- Dezentes „keine Eltern-E-Mail"-Icon statt lautem Dauer-Label auf jeder Zeile.
										     role="img": Auf einem nackten <span> ist aria-label verboten
										     (aria-prohibited-attr, 174 Treffer am 09.09.2026) — als Bild mit
										     Namen liest der Screenreader es vor, statt es zu ignorieren. -->
										<span
											role="img"
											class="text-slate-400 shrink-0 flex items-center"
											title="Keine Eltern-E-Mail hinterlegt"
											aria-label="Keine Eltern-E-Mail hinterlegt"
										>
											<Mail class="h-3.5 w-3.5" aria-hidden="true" />
										</span>
									{/if}
								</div>
							</td>
							<td>
								<span class="text-sm text-slate-600">
									{schueler.klasse}
								</span>
							</td>
							<td>
								<!-- Text statt Cover-Stapel: Ab drei, vier überfälligen Büchern wurde die
								     Reihe aus Miniaturen plus "+N"-Badge selbst zur Ratearbeit — welches
								     Cover zu welchem Titel gehört, war ohnehin nicht zu erkennen. Wie bei
								     Abgänger zählt hier nur, WIE VIELE es sind; der Titeltext im title-
								     Attribut bleibt für den Hover-Fall erhalten. -->
								<span
									class="text-sm text-slate-600"
									title={schueler.medien.map((m) => m.titel).join(', ')}
								>
									{schueler.medien.length}
									{schueler.medien.length === 1 ? 'Buch' : 'Bücher'}
								</span>
							</td>
							<!-- Ein Farbträger je Zeile, und Farbe nur für die AUSNAHME. Auf diesem
							     Bildschirm ist alles überfällig — die erste Erinnerung ist der Normalfall
							     und braucht keine Warnfarbe. Rot bekommt nur die Eskalation. Vorher trug
							     jede Zeile Pille UND roten Text: bei 422 Zeilen eine Farbwand, in der die
							     wirklich dringenden Fälle untergehen. -->
							<td>
								<div class="flex flex-col items-start">
									<span
										class="text-sm font-medium {schueler.mahnstufe === 'Mahnung'
											? 'text-rose-600'
											: 'text-slate-700'}"
									>
										{schueler.mahnstufe}
									</span>
									<span class="text-sm text-slate-500">
										{schueler.maxTage === 0
											? 'heute fällig'
											: `${schueler.maxTage} ${schueler.maxTage === 1 ? 'Tag' : 'Tage'} überfällig`}
									</span>
								</div>
							</td>
						</tr>
					{/each}
					{#if mahnwesenStore.filteredSchueler.length === 0}
						<tr>
							<td colspan="5" class="text-center">
								Keine Treffer für die aktuelle Auswahl (Tab, Klasse oder Suche).
							</td>
						</tr>
					{/if}
				</tbody>
			</Tabelle>
		</div>
	</div>
{/if}

<!-- Die Auswahl-Aktion („Mahnbriefe drucken") lebt jetzt in der kontextuellen Toolbar oben
     (MahnwesenFilters) — kein separater Schwebe-Balken mehr. -->

<MahnlisteMailDialog />
