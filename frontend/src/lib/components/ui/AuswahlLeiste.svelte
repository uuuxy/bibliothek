<!-- @component AuswahlLeiste — die Aktionen für markierte Zeilen einer Liste (Material 3
     „floating toolbar").

     M3 Selection: „To exit a selection mode, tap each selected item until they're unselected,
     or tap an action on the toolbar." M3 Toolbars: Die schwebende Form „is best used for
     contextual actions relevant to the body content". Maße nach material-components-android
     (FloatingToolbar.md): Fläche `surface-container` (Farbschema „standard"), Form „50 %
     rounded", Abstand 16 dp zum Rand; darüber Erhebung wie ein Menü (shadow-xl).

     Die Leiste liegt unten über dem Inhalt statt oben in der Werkzeugleiste: Wer markiert,
     scrollt dabei durch die Liste, und ein Knopf oben wäre nach der dritten Zeile aus dem Bild
     (derselbe Grund wie bei der Leiste des Ausweisdrucks, students/AuswahlAktionsleiste).
     `sticky`, nicht `fixed`, und als letztes Kind der Spalte, die die Liste trägt: M3
     „Floating toolbars shouldn't exceed the edge of the window or pane" — die Leiste gehört
     mittig unter ihre Liste. `fixed` zentrierte sie über dem Fenster; auf der Pflegeseite der
     Schlagworte lag sie dann über der Kategorienliste (gemessen am 23.09.2026).

     Die Aktionen kommen als children; das X hebt die Markierung auf. -->
<script>
	import { X } from '@lucide/svelte';
	import Button from './Button.svelte';

	/** @type {{
	 *   satz: string,
	 *   beschriftung: string,
	 *   onleeren: () => void,
	 *   children: import('svelte').Snippet
	 * }} */
	let { satz, beschriftung, onleeren, children } = $props();
</script>

<div
	class="no-print pointer-events-none sticky bottom-4 z-40 flex justify-center"
	role="region"
	aria-label={beschriftung}
>
	<div
		class="pointer-events-auto flex h-16 max-w-full items-center gap-2 rounded-full bg-surface-container pr-3 pl-6 text-on-surface shadow-xl"
	>
		<span class="text-sm whitespace-nowrap" aria-live="polite">{satz}</span>
		<div class="ml-4 flex items-center gap-2">
			{@render children()}
			<Button
				variant="ghost"
				size="sm"
				aria-label="Markierung aufheben"
				data-tip="Markierung aufheben"
				onclick={onleeren}
			>
				<X class="h-4 w-4" aria-hidden="true" />
			</Button>
		</div>
	</div>
</div>
