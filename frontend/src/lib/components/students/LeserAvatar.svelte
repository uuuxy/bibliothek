<!--
  @component
  LeserAvatar — das runde Passbild einer Listenzeile, sonst die Initialen.

  Eigene Datei wegen der 200-Zeilen-Regel (docs/arc42/05-bausteinsicht.md, 5.3): ActiveStudentList hat
  mit der Spalte „Art" die Grenze erreicht, und das Bild ist das Stück, das nichts mit
  dem Führen der Liste zu tun hat.
-->
<script>
	/** @type {{ leser: any }} */
	let { leser } = $props();
</script>

<div
	class="relative w-8 h-8 rounded-full overflow-hidden border border-slate-100/80 bg-slate-50 flex items-center justify-center shrink-0"
>
	{#if leser.foto_url}
		<img
			src={leser.foto_url}
			alt="Passbild von {leser.vorname} {leser.nachname}"
			class="w-full h-full object-cover"
		/>
	{:else}
		<div
			class="w-full h-full flex items-center justify-center bg-slate-100 text-slate-500 font-medium text-sm"
			aria-hidden="true"
		>
			{leser.vorname.charAt(0)}{leser.nachname.charAt(0)}
		</div>
	{/if}
</div>
