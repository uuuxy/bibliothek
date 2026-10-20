<!-- @component Abschnitt — die Überschrift eines Formular-Abschnitts.

     Die Hierarchie baut M3 über Schrift und Farbrolle, nicht über Schmuck: Der Titel steht in
     `on-surface` (der vollen Textfarbe) über den Beschriftungen der Felder, die in
     `on-surface-variant` stehen. Ein farbiger Punkt vor dem Titel ist kein M3-Bauteil und
     behauptete eine Bedeutung, die es nicht gibt. -->
<script>
	/** @type {{ titel: string, class?: string }} */
	let { titel, class: className = '' } = $props();
</script>

<h3 class="mb-4 text-base font-medium text-on-surface {className}">{titel}</h3>
