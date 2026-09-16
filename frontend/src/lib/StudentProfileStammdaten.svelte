<script>
	/**
	 * @type {{
	 *   profile: any,
	 *   darfBearbeiten?: boolean,
	 *   darfZusammenfuehren?: boolean,
	 *   onEdit: () => void,
	 *   onMerged?: (zielId: string) => void
	 * }}
	 */
	let {
		profile,
		darfBearbeiten = false,
		darfZusammenfuehren = false,
		onEdit,
		onMerged = () => {}
	} = $props();

	import Button from './components/ui/Button.svelte';
	import SchuelerZusammenfuehren from './components/students/SchuelerZusammenfuehren.svelte';
	import { istKollegium, leserArtText } from './leserArt.js';
	import { Folder, Info, SquarePen } from '@lucide/svelte';

	const kollege = $derived(istKollegium(profile));

	function formatDate(dateString) {
		if (!dateString) return 'Keine Angabe';
		try {
			const d = new Date(dateString);
			return d.toLocaleDateString('de-DE', { day: '2-digit', month: '2-digit', year: 'numeric' });
		} catch {
			return dateString;
		}
	}
</script>

<!-- EINE Ansicht für jeden Leser — dieselbe Regel wie im Formular (Peter, 16.09.2026:
     „bitte nicht verkomplizieren").

     Bis zum 16.09.2026 stand hier eine Weiche: Ein Kollege bekam eine eigene Ansicht ohne
     Geburtsdatum, LUSD-Kennung und Elternadresse, mit der Begründung, „Keine Angabe"
     behaupte, sie FEHLTEN. Seit das Formular jedem alle Felder anbietet, ist das nicht
     mehr haltbar: Ein Kollege konnte ein Geburtsdatum eintragen, das seine Akte danach
     nicht zeigte — ein Feld, das man füllen, aber nicht lesen kann, ist schlechter als
     eines, das leer dasteht.

     Die Ausweisnummer steht bewusst NICHT hier: Sie steht schon auf der Karte links
     (StudentProfileCard.svelte), samt Hinweis, wenn noch keine vergeben ist. Zweimal
     dieselbe Zahl ist keine Gründlichkeit.

     Zusammenführen bleibt vorerst eine Schülersache — nicht aus Absicht, sondern weil der
     Zusammenführ-Code gegen die Sicht `schueler` schreibt und bei einem Kollegen null
     Zeilen träfe. Steht in docs/OFFEN.md 5.16. -->
<div class="w-full pt-2 animate-fade-in space-y-8">
	<div class="flex justify-between items-center border-b border-outline-variant pb-4">
		<h3 class="text-xl font-bold text-on-surface flex items-center gap-2">
			<Folder class="w-6 h-6 text-primary" aria-hidden="true" />
			Stammdaten & Adresse
		</h3>
		<div class="flex items-center gap-2">
			{#if darfBearbeiten}
				<!-- Keine Palette-Übersteuerung: Das M3-Bauteil trägt seine Farben selbst
				     (styles/rollen.css). Die vier Tailwind-Klassen hier waren eine zweite
				     Farbquelle neben den Rollen — genau das, was die Farb-Ratsche abbaut. -->
				<Button variant="secondary" size="lg" onclick={onEdit} class="px-5">
					<SquarePen class="w-4 h-4" aria-hidden="true" />
					Bearbeiten
				</Button>
			{/if}
		</div>
	</div>

	<div class="grid grid-cols-1 md:grid-cols-2 gap-8">
		<div class="space-y-6">
			<!-- Die Art stand bis zum 16.09.2026 NUR in der Akte eines Kollegen. In der
			     Schülerakte fehlte sie ganz — „hier steht nirgends ob jemand ein Schüler,
			     LiV, oder lehrer ist" (Peter). Seit die Leserdatei alle in einer Tabelle
			     führt, ist sie bei jedem die erste Auskunft, nicht nur bei den anderen. -->
			<div>
				<p class="text-xs font-medium text-on-surface-variant mb-1">Art</p>
				<p class="text-on-surface font-semibold">{leserArtText(profile.art)}</p>
			</div>
			<div>
				<p class="text-xs font-medium text-on-surface-variant mb-1">Geburtsdatum</p>
				<p class="text-on-surface font-semibold">{formatDate(profile.geburtsdatum)}</p>
			</div>
			<div>
				<p class="text-xs font-medium text-on-surface-variant mb-1">LUSD ID</p>
				<p class="text-on-surface font-semibold">{profile.lusd_id || 'Keine Angabe'}</p>
			</div>
		</div>

		<div class="space-y-6">
			<div>
				<p class="text-xs font-medium text-on-surface-variant mb-1">Postanschrift</p>
				{#if profile.strasse}
					<p class="text-on-surface font-semibold">{profile.strasse} {profile.hausnummer}</p>
					<p class="text-on-surface font-semibold">{profile.plz} {profile.ort}</p>
				{:else}
					<p class="text-on-surface-variant italic text-sm">Keine Adresse hinterlegt</p>
				{/if}
			</div>
			<div>
				<p class="text-xs font-medium text-on-surface-variant mb-1">Eltern E-Mail</p>
				{#if profile.eltern_email}
					<a href="mailto:{profile.eltern_email}" class="text-primary hover:underline font-semibold"
						>{profile.eltern_email}</a
					>
				{:else}
					<p class="text-on-surface-variant italic text-sm">Keine E-Mail hinterlegt</p>
				{/if}
			</div>
		</div>
	</div>

	<!-- Was am KONTO hängt, steht nicht hier: Die Anmeldung erkennt eine Person an ihrer
	     E-Mail-Adresse, und die wird an genau einer Stelle gepflegt. Der Hinweis gilt nur
	     dem Kollegium — ein Schüler hat kein Konto, ihm sagte der Satz nichts. -->
	{#if kollege}
		<div
			class="flex items-start gap-3 rounded-xl border border-outline-variant bg-surface-container-low px-4 py-3 text-sm text-on-surface-variant"
		>
			<Info class="h-5 w-5 shrink-0 text-outline" aria-hidden="true" />
			<p>
				<span class="font-semibold">E-Mail-Adresse, Rolle und Freischaltung</span> stehen nicht
				hier, sondern in <span class="font-semibold">Benutzer &amp; Rechte</span>. Die Anmeldung
				erkennt eine Person an ihrer E-Mail-Adresse — darum wird sie an genau einer Stelle gepflegt.
			</p>
		</div>
	{/if}

	<!-- Zusammenführen: Admin-Recht, unumkehrbar — deshalb hier unten bei den Stammdaten,
	     nicht zwischen den Dokument-Knöpfen. Dialog und Suche bringt der Abschnitt mit.

	     Seit dem 16.09.2026 auch beim Kollegium: Ein Kollege, der von Hand eingetragen wurde
	     und sich später selbst anmeldet, steht zweimal da — und war bis dahin von NIEMANDEM
	     zu reparieren, weil das Zusammenführen gegen die Sicht `schueler` schrieb. Das Recht
	     bleibt, wie es war (merge_students: Admin und Leitung ab Werk). -->
	{#if darfZusammenfuehren}
		<SchuelerZusammenfuehren {profile} {onMerged} />
	{/if}
</div>
