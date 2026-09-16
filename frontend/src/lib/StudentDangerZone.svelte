<script>
	import { AlertTriangle } from '@lucide/svelte';
	import Button from './components/ui/Button.svelte';

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
	 * @prop {() => void} onDelete - Callback, der den Lösch-/Archivierungsdialog öffnet.
	 * @prop {boolean} [kollege] - Lehrkraft oder LiV (hat einen Zugang).
	 */

	/** @type {{ onDelete: () => void, kollege?: boolean }} */
	let { onDelete, kollege = false } = $props();
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
			{#if kollege}
				Das Löschen entfernt die Person aus dem regulären System — und mit ihr den Zugang zu „Mein
				Portal“. Offene Ausleihen oder Forderungen müssen vorher beglichen werden.
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
