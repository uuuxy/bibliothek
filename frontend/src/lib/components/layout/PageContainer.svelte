<!--
  @component PageContainer
  Wiederverwendbarer zentraler Layout-Container für responsive Seiten in Svelte 5.
  Nutzt standardisierte Tailwind-Klassen für maximale Breite und ergonomische Ausrichtung.
-->
<script lang="ts">
	let { children } = $props();
</script>

<div class="w-full max-w-5xl mx-auto px-4 sm:px-6 lg:px-8">
	{@render children()}
</div>
