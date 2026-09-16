<!--
  @component
  LeserPersoenlicheDaten — der Stammdaten-Reiter in der Akte eines Kollegen.

  Ein Kollege ist kein Schüler mit fehlenden Angaben: Geburtsdatum, LUSD-Kennung,
  Postanschrift und Elternadresse gehören ihm nicht. Stünden sie hier als „Keine Angabe",
  behauptete die Akte, sie FEHLTEN — und jemand trüge sie irgendwann nach.

  Was am Konto hängt (E-Mail, Rolle, Freischaltung), bleibt in „Benutzer & Rechte": Die
  Anmeldung erkennt eine Person an ihrer E-Mail, und zwei Türen zu demselben Zustand sind
  keine Bequemlichkeit — die falsche geht irgendwann auf.
-->
<!-- Farben aus den M3-Rollen (styles/rollen.css), nicht aus der Tailwind-Palette:
     Neues gehört dorthin, und die Farb-Ratsche darf nur sinken. -->
<script>
	import { Folder, Info } from '@lucide/svelte';
	import { leserArtText } from '../../leserArt.js';

	/** @type {{ profile: any }} */
	let { profile } = $props();
</script>

<div class="w-full pt-2 animate-fade-in space-y-8">
	<div class="flex justify-between items-center border-b border-outline-variant pb-4">
		<h3 class="text-xl font-bold text-on-surface flex items-center gap-2">
			<Folder class="w-6 h-6 text-primary" aria-hidden="true" />
			Persönliche Daten
		</h3>
	</div>

	<div class="grid grid-cols-1 md:grid-cols-2 gap-8">
		<div class="space-y-6">
			<div>
				<p class="text-xs font-medium text-on-surface-variant mb-1">Name</p>
				<p class="text-on-surface font-semibold">{profile.vorname} {profile.nachname}</p>
			</div>
			<div>
				<p class="text-xs font-medium text-on-surface-variant mb-1">Art</p>
				<p class="text-on-surface font-semibold">{leserArtText(profile.art)}</p>
			</div>
		</div>

		<div class="space-y-6">
			<div>
				<p class="text-xs font-medium text-on-surface-variant mb-1">Ausweisnummer</p>
				{#if profile.barcode_id}
					<p class="text-on-surface font-semibold font-mono tracking-widest">
						{profile.barcode_id}
					</p>
				{:else}
					<p class="text-on-surface-variant italic text-sm">
						Noch keine — ohne Nummer lässt sich kein Ausweis drucken.
					</p>
				{/if}
			</div>
		</div>
	</div>

	<div
		class="flex items-start gap-3 rounded-xl border border-outline-variant bg-surface-container-low px-4 py-3 text-sm text-on-surface-variant"
	>
		<Info class="h-5 w-5 shrink-0 text-outline" aria-hidden="true" />
		<p>
			Name, Ausweisnummer, E-Mail-Adresse, Rolle und Freischaltung stehen in
			<span class="font-semibold">Benutzer &amp; Rechte</span>. Die Anmeldung erkennt eine Person an
			ihrer E-Mail-Adresse — darum wird sie an genau einer Stelle gepflegt.
		</p>
	</div>
</div>
