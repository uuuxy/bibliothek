<script>
	/**
	 * @type {{
	 *   leser: any,
	 *   darfBearbeiten?: boolean,
	 *   darfZusammenfuehren?: boolean,
	 *   onEdit: () => void,
	 *   onMerged?: (zielId: string) => void
	 * }}
	 */
	let {
		leser,
		darfBearbeiten = false,
		darfZusammenfuehren = false,
		onEdit,
		onMerged = () => {}
	} = $props();

	import Button from './components/ui/Button.svelte';
	import SchuelerZusammenfuehren from './components/leser/SchuelerZusammenfuehren.svelte';
	import { istKollegium, leserArtText } from './leserArt.js';
	import { Info, SquarePen } from '@lucide/svelte';

	const kollege = $derived(istKollegium(leser));

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

<!-- Eine Ansicht für jeden Leser, dieselbe Regel wie im Formular: Ein Feld, das man füllen,
     aber nicht lesen kann, ist schlechter als eines, das leer dasteht.

     Die Ausweisnummer steht nicht hier: Sie steht schon auf der Karte links
     (LeserakteKopf.svelte), samt Hinweis, wenn noch keine vergeben ist. -->
<div class="w-full pt-2 animate-fade-in space-y-8">
	<!-- Keine eigene Überschrift: Der Reiter darüber heißt schon „Stammdaten & Adresse“. -->
	{#if darfBearbeiten}
		<div class="flex justify-end border-b border-outline-variant pb-4">
			<!-- Keine Palette-Übersteuerung: Das M3-Bauteil trägt seine Farben selbst
			     (styles/rollen.css). Die vier Tailwind-Klassen hier waren eine zweite
			     Farbquelle neben den Rollen — genau das, was die Farb-Ratsche abbaut. -->
			<Button variant="secondary" size="lg" onclick={onEdit} class="px-5">
				<SquarePen class="w-4 h-4" aria-hidden="true" />
				Bearbeiten
			</Button>
		</div>
	{/if}

	<div class="grid grid-cols-1 md:grid-cols-2 gap-8">
		<div class="space-y-6">
			<!-- Die Art ist bei jedem die erste Auskunft: Die Leserdatei führt Schüler, LiV und
			     Lehrkräfte in einer Tabelle. -->
			<div>
				<p class="text-xs font-medium text-on-surface-variant mb-1">Art</p>
				<p class="text-on-surface font-semibold">{leserArtText(leser.art)}</p>
			</div>
			<div>
				<p class="text-xs font-medium text-on-surface-variant mb-1">Geburtsdatum</p>
				<p class="text-on-surface font-semibold">{formatDate(leser.geburtsdatum)}</p>
			</div>
			<div>
				<p class="text-xs font-medium text-on-surface-variant mb-1">LUSD ID</p>
				<p class="text-on-surface font-semibold">{leser.lusd_id || 'Keine Angabe'}</p>
			</div>
		</div>

		<div class="space-y-6">
			<div>
				<p class="text-xs font-medium text-on-surface-variant mb-1">Postanschrift</p>
				{#if leser.strasse}
					<p class="text-on-surface font-semibold">{leser.strasse} {leser.hausnummer}</p>
					<p class="text-on-surface font-semibold">{leser.plz} {leser.ort}</p>
				{:else}
					<p class="text-on-surface-variant italic text-sm">Keine Adresse hinterlegt</p>
				{/if}
			</div>
			<div>
				<p class="text-xs font-medium text-on-surface-variant mb-1">Eltern E-Mail</p>
				{#if leser.eltern_email}
					<a href="mailto:{leser.eltern_email}" class="text-primary hover:underline font-semibold"
						>{leser.eltern_email}</a
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
			class="flex items-start gap-3 rounded-xl border border-outline-variant px-4 py-3 text-sm text-on-surface-variant"
		>
			<Info class="h-5 w-5 shrink-0 text-outline" aria-hidden="true" />
			<p>
				<span class="font-semibold">E-Mail-Adresse, Rolle und Freischaltung</span> stehen nicht
				hier, sondern in <span class="font-semibold">Benutzer &amp; Rechte</span>. Die Anmeldung
				erkennt eine Person an ihrer E-Mail-Adresse — darum wird sie an genau einer Stelle gepflegt.
			</p>
		</div>
	{/if}

	<!-- Zusammenführen: Admin-Recht, unumkehrbar, deshalb hier unten bei den Stammdaten,
	     nicht zwischen den Dokument-Knöpfen. Dialog und Suche bringt der Abschnitt mit. Auch
	     beim Kollegium: Ein Kollege, der von Hand eingetragen wurde und sich später selbst
	     anmeldet, steht sonst zweimal da (Recht merge_students, ab Werk Admin und Leitung). -->
	{#if darfZusammenfuehren}
		<SchuelerZusammenfuehren {leser} {onMerged} />
	{/if}
</div>
