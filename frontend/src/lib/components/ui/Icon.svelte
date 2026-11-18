<!--
  @component
  Ein Icon aus Googles Material Symbols (Outlined), die eine Stelle, über die jedes Icon der
  Oberfläche läuft. Schrift und Regeln stehen in styles/icons.css, die Namen in iconNamen.js.

  Ohne `vorlesen` ist das Icon Schmuck neben einem Text und für den Screenreader unsichtbar; mit
  `vorlesen` steht es für sich und trägt den Namen als Bild (M3, Icons, Accessibility: „Icon meaning
  should always be unambiguous and accessible for all users“).

  `groesse` nur, wo kein Bauteil die Größe vorgibt und 20 px nicht passen: 18 in einem kleinen
  Bedienelement oder neben kleiner Schrift (die Größe der Chips), 24 neben einer Überschrift, 40
  oder 48 für eine hervorgehobene Stelle, M3 (Icons, Optical sizes): „Use larger
  size 40dp–48dp symbols when primary actions need to be highlighted“.

  `imText` für ein Icon mitten in einer Textzeile: so groß wie der Text und etwas tiefer, M3 (Icons,
  Using Material Symbols with typography): „Use the same size for your Material Symbols and text“
  und „Shift down the baseline of symbols to approximately 11.5% of the text size“.
-->
<script>
	/**
	 * @type {{
	 *   name: import('./iconNamen.js').IconName,
	 *   groesse?: 18 | 24 | 40 | 48,
	 *   gefuellt?: boolean,
	 *   imText?: boolean,
	 *   vorlesen?: string,
	 *   class?: string
	 * }}
	 */
	let { name, groesse, gefuellt = false, imText = false, vorlesen, class: klasse = '' } = $props();
</script>

<span
	class="icon {klasse}"
	data-icon={name}
	data-gefuellt={gefuellt ? '' : undefined}
	data-im-text={imText ? '' : undefined}
	style:--icon-groesse={groesse ? `${groesse}px` : undefined}
	role={vorlesen ? 'img' : undefined}
	aria-label={vorlesen}
	aria-hidden={vorlesen ? undefined : 'true'}
></span>
