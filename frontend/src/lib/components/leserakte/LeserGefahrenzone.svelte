<script>
	import { TriangleAlert } from '@lucide/svelte';
	import Button from '../ui/Button.svelte';
	import { istKollegium } from '../leser/leserArt.js';

	/**
	 * @component LeserGefahrenzone
	 * Admin-only "Gefahrenzone" zum Archivieren/Löschen eines Leser-Profils.
	 * Wird ausschließlich am unteren Ende des Reiters "Stammdaten & Adresse" gerendert.
	 *
	 * Sie steht auch beim Kollegium, und mit einem Kollegen fällt sein Zugang. Das muss hier
	 * stehen, bevor jemand klickt: Nach dem Wiederherstellen ist der Zugang nicht von allein
	 * zurück, er wird über die Schul-E-Mail in der Akte neu angelegt.
	 *
	 * Vom Zugang spricht sie nur, wenn es einen gibt: Praktikum und Fachbereich haben nie
	 * einen (Migration 153), und eine Lehrkraft verliert ihn, wenn die Benutzerverwaltung das
	 * Konto löscht. `email` im Profil ist die Adresse am Konto, leer heißt kein Konto.
	 *
	 * @prop {() => void} onDelete - Callback, der den Lösch-/Archivierungsdialog öffnet.
	 * @prop {any} leser - die geladene Akte (art, email).
	 */

	/** @type {{ onDelete: () => void, leser: any }} */
	let { onDelete, leser } = $props();

	const kollege = $derived(istKollegium(leser));
	const mitZugang = $derived(kollege && !!leser?.email);
</script>

<section
	class="mt-8 border border-error-container bg-error-container/20 rounded-xl p-6 flex flex-col md:flex-row gap-6 items-start md:items-center justify-between"
>
	<div>
		<h3 class="text-error font-bold text-base flex items-center gap-2">
			<TriangleAlert class="w-5 h-5" />
			Gefahrenzone
		</h3>
		<p class="text-error/80 text-sm mt-1 max-w-xl">
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
		class="shrink-0 px-6 bg-surface-container-lowest"
	>
		{kollege ? 'Kollegen archivieren / löschen' : 'Schüler archivieren / löschen'}
	</Button>
</section>
