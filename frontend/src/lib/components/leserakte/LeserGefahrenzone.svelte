<script>
	import { TriangleAlert } from '@lucide/svelte';
	import Button from '../ui/Button.svelte';
	import { istKollegium } from '../leser/leserArt.js';
	import { schuelerRechte } from '../leser/schuelerRechte.js';
	import { authStore } from '../../stores/authStore.svelte.js';

	/**
	 * @component LeserGefahrenzone
	 * "Gefahrenzone" zum Archivieren/Löschen eines Leser-Profils, für jeden mit dem Recht
	 * delete_students. Wird ausschließlich am unteren Ende des Reiters "Stammdaten & Adresse"
	 * gerendert.
	 *
	 * Sie steht auch beim Kollegium, und mit einem Kollegen fällt sein Zugang. Das muss hier
	 * stehen, bevor jemand klickt: Nach dem Wiederherstellen ist der Zugang nicht von allein
	 * zurück, er wird über die Schul-E-Mail in der Akte neu angelegt.
	 *
	 * Vom Zugang spricht sie nur, wenn es einen gibt (`hat_zugang` der Akte): Praktikum und
	 * Fachbereich haben nie einen (Migration 153), eine aus Littera übernommene Lehrkraft nur ein
	 * Konto mit Ersatzadresse, und eine Lehrkraft verliert ihn, wenn die Benutzerverwaltung das
	 * Konto löscht.
	 *
	 * Einen Zugang löscht nur, wer auch Konten verwaltet, wie in der Benutzerverwaltung. Wer das
	 * nicht darf, sieht den Knopf abgeschaltet und den Grund daneben, statt nach ihm zu suchen.
	 *
	 * @prop {() => void} onDelete - Callback, der den Lösch-/Archivierungsdialog öffnet.
	 * @prop {any} leser - die geladene Akte (art, hat_zugang).
	 */

	/** @type {{ onDelete: () => void, leser: any }} */
	let { onDelete, leser } = $props();

	const kollege = $derived(istKollegium(leser));
	const mitZugang = $derived(!!leser?.hat_zugang);
	// Das Recht nach derselben Zuordnung wie die Akte (schuelerRechte), nicht nach der Rolle.
	const zugangGesperrt = $derived(
		mitZugang && !schuelerRechte(authStore.currentUser).loeschenMitZugang
	);
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
			{#if zugangGesperrt}
				Diese Person hat einen Zugang zu „Mein Portal“. Löschen kann sie nur, wer „Benutzer &amp;
				Rechte verwalten“ darf.
			{:else if mitZugang}
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
		disabled={zugangGesperrt}
		class="shrink-0 px-6 bg-surface-container-lowest"
	>
		{kollege ? 'Kollegen archivieren / löschen' : 'Schüler archivieren / löschen'}
	</Button>
</section>
