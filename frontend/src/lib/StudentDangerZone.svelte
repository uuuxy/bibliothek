<script>
	import { AlertTriangle } from '@lucide/svelte';
	import Button from './components/ui/Button.svelte';

	/**
	 * @component StudentDangerZone
	 * Admin-only "Gefahrenzone" zum Archivieren/Löschen eines Schülerprofils.
	 * Wird ausschließlich am unteren Ende des Reiters "Stammdaten & Adresse" gerendert.
	 *
	 * @prop {() => void} onDelete - Callback, der den Lösch-/Archivierungsdialog öffnet.
	 */

	/** @type {{ onDelete: () => void }} */
	let { onDelete } = $props();
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
			Das Löschen dieses Schülerprofils entfernt die Person aus dem regulären System. Offene
			Ausleihen oder Forderungen müssen vorher beglichen werden.
		</p>
	</div>
	<Button
		variant="danger"
		size="lg"
		onclick={onDelete}
		class="shrink-0 px-6 bg-white hover:bg-rose-600 hover:text-white"
	>
		Schüler archivieren / löschen
	</Button>
</section>
