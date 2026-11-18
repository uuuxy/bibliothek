<!--
  @component
  LeserAvatar — das runde Passbild einer Listenzeile, sonst die Initialen.

  Eigene Datei wegen der 200-Zeilen-Regel (docs/ARCHITEKTUR.md, 5.3): AktiveLeserListe hat
  mit der Spalte „Art" die Grenze erreicht, und das Bild ist das Stück, das nichts mit
  dem Führen der Liste zu tun hat.
-->
<script>
	/** @type {{ leser: any }} */
	let { leser } = $props();
</script>

<div
	class="relative w-8 h-8 rounded-full overflow-hidden border border-outline-variant bg-surface flex items-center justify-center shrink-0"
>
	{#if leser.foto_url}
		<!-- Leerer Alternativtext: Der Name steht in der Zelle daneben, M3 Alt text: „Don’t repeat
		     the caption in alt text“. -->
		<img src={leser.foto_url} alt="" class="w-full h-full object-cover" />
	{:else}
		<div
			class="w-full h-full flex items-center justify-center bg-surface-container text-on-surface-variant font-medium text-sm"
			aria-hidden="true"
		>
			{leser.vorname.charAt(0)}{leser.nachname.charAt(0)}
		</div>
	{/if}
</div>
