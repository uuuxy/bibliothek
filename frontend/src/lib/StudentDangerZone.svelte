<script>
	import { AlertTriangle } from '@lucide/svelte';
	import Button from './components/ui/Button.svelte';
	import { istKollegium } from './leserArt.js';

	/**
	 * @component StudentDangerZone
	 * Admin-only "Gefahrenzone" zum Archivieren/Löschen eines Leser-Profils.
	 * Wird ausschließlich am unteren Ende des Reiters "Stammdaten & Adresse" gerendert.
	 *
	 * Seit dem 16.09.2026 steht sie auch beim Kollegium. Vorher war sie dort ausgeblendet,
	 * weil der ganze Löschweg gegen die Sicht `schueler` schrieb und bei einem Kollegen
	 * „nicht gefunden" antwortete. Mit ihm fällt jetzt sein ZUGANG (entschieden am
	 * 16.09.2026: „wenn ein kollege gelöscht wird dann wird alles gelöscht"). Das muss
	 * hier stehen, bevor jemand
	 * klickt: Nach dem Wiederherstellen ist der Zugang nicht von allein zurück, er wird
	 * über die Schul-E-Mail in der Akte neu angelegt.
	 *
	 * Vom Zugang spricht sie nur, wenn es einen gibt: Praktikum und Fachbereich haben nie
	 * einen (Migration 153), und eine Lehrkraft verliert ihn, wenn die Benutzerverwaltung das
	 * Konto löscht. `email` im Profil ist die Adresse am Konto, leer heißt kein Konto.
	 *
	 * @prop {() => void} onDelete - Callback, der den Lösch-/Archivierungsdialog öffnet.
	 * @prop {any} profile - die geladene Akte (art, email).
	 */

	/** @type {{ onDelete: () => void, profile: any }} */
	let { onDelete, profile } = $props();

	const kollege = $derived(istKollegium(profile));
	const mitZugang = $derived(kollege && !!profile?.email);
</script>

<section
	class="mt-8 border border-rose-100 bg-rose-50/50 rounded-2xl p-6 flex flex-col md:flex-row gap-6 items-start md:items-center justify-between"
>
	<div>
		<h3 class="text-rose-700 font-bold text-base flex items-center gap-2">
			<AlertTriangle class="w-5 h-5" />
			Gefahrenzone
		</h3>
		<p class="text-rose-600/80 text-sm mt-1 max-w-xl">
			{#if mitZugang}
				Das Löschen entfernt die Person aus dem regulären System — und mit ihr den Zugang zu „Mein
				Portal“. Offene Ausleihen oder Forderungen müssen vorher beglichen werden.
			{:else if kollege}
				Das Löschen entfernt den Eintrag aus dem regulären System. Offene Ausleihen oder Forderungen
				müssen vorher beglichen werden.
			{:else}
				Das Löschen dieses Schülerprofils entfernt die Person aus dem regulären System. Offene
				Ausleihen oder Forderungen müssen vorher beglichen werden.
			{/if}
		</p>
	</div>
	<Button
		variant="danger"
		size="lg"
		onclick={onDelete}
		class="shrink-0 px-6 bg-white hover:bg-rose-600 hover:text-white"
	>
		{kollege ? 'Kollegen archivieren / löschen' : 'Schüler archivieren / löschen'}
	</Button>
</section>
