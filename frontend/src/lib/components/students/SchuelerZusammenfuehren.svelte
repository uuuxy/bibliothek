<!-- @component SchuelerZusammenfuehren — der Abschnitt „Doppelter Datensatz?" am Ende des
     Reiters „Stammdaten & Adresse", über der Gefahrenzone: sichtbar nur mit dem Recht
     merge_students (schuelerRechte.zusammenfuehren; eigenes Recht seit 03.09.2026). Trägt den Dialog gleich mit,
     damit StudentProfile.svelte (an der 200-Zeilen-Ratsche) nichts davon halten muss. -->
<script>
	import { Merge } from '@lucide/svelte';
	import Button from '../ui/Button.svelte';
	import SchuelerZusammenfuehrenDialog from './SchuelerZusammenfuehrenDialog.svelte';

	/** @type {{ profile: any, onMerged: (zielId: string) => void }} */
	let { profile, onMerged } = $props();

	let offen = $state(false);
</script>

<section
	class="mt-8 rounded-2xl border border-outline-variant bg-surface-container-low p-6 flex flex-col md:flex-row gap-6 items-start md:items-center justify-between"
>
	<div>
		<h3 class="text-on-surface font-bold text-base flex items-center gap-2">
			<Merge class="w-5 h-5 text-primary" aria-hidden="true" />
			Doppelter Datensatz?
		</h3>
		<p class="text-on-surface-variant text-sm mt-1 max-w-xl">
			Steht dieselbe Person zweimal in der Kartei — etwa nach einer Namensänderung in der LUSD, die
			der Export ohne Schüler-ID nicht wiedererkannt hat —, lassen sich beide Datensätze zu einem
			zusammenführen. Ausweis, Ausleihen und Historie bleiben erhalten.
		</p>
	</div>
	<Button variant="secondary" size="lg" onclick={() => (offen = true)} class="shrink-0 px-6">
		Mit anderem Datensatz zusammenführen
	</Button>
</section>

<SchuelerZusammenfuehrenDialog bind:open={offen} {profile} {onMerged} />
