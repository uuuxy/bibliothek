<!--
  @component Sheet
  Die weisse Karte, auf der Inhalt liegt — das Gegenstueck zur getoenten Leinwand
  (`surface` in App.svelte). In Material 3 ist genau das die Flaechenhierarchie:
  die App-Flaeche ist getoent, Inhalt liegt als hellerer Container darauf.

  Das Rezept stand vorher nur an einzelnen Stellen im Baum (MahnwesenTable machte es
  richtig, die Schuelerdatei gar nicht und taeuschte es stattdessen mit einem getoenten
  Seitenhintergrund vor). Hier steht es EINMAL.

  Radius 12 px (`rounded-xl`): In der M3-Shape-Skala ist das die KARTEN-Rolle. 16 px
  (`rounded-2xl`) gehoert Navigationsflaechen, 28 px den Dialogen — die Zuordnung steht
  in styles/theme-mass.css. Diese Komponente stand zuerst selbst auf 16 px.

  `class` ist fuer Layout gedacht (Abstaende, Ueberlauf) — keine Flaechenfarben:
  Utilities gleicher Spezifitaet entscheidet die Stylesheet-Reihenfolge, nicht das
  class-Attribut, ein `bg-*` von aussen wirkt hier also nicht zuverlaessig.
-->
<script>
	let { class: klasse = '', children } = $props();
</script>

<div
	class="bg-surface-container-lowest border-outline-variant overflow-hidden rounded-xl border shadow-sm {klasse}"
>
	{@render children()}
</div>
