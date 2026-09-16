<!-- @component Abschnitt — die Überschrift eines Formular-Abschnitts.

     Bis zum 16.09.2026 trug jeder Abschnitt der Leserakte einen farbigen Punkt vor dem
     Titel: zweimal grau, einmal blau. Das ist kein Material-3-Bauteil, es stand so in
     keiner zweiten Maske der Anwendung, und die Farbe behauptete eine Bedeutung, die es
     nicht gibt — warum wäre die Anschrift wichtiger als der Name?

     Geblieben ist die Hierarchie, die M3 über Schrift und Farbrolle baut: Der Titel steht
     in `on-surface` (der vollen Textfarbe) über den Beschriftungen der Felder, die in
     `on-surface-variant` stehen. Damit kommen die Farben aus den Rollen (styles/rollen.css)
     statt aus der Tailwind-Palette — der Punkt war eine der wenigen Stellen, an denen eine
     NEUE Datei die Farb-Ratsche wieder anhob (frontend-hygiene-farben.test.js). -->
<script>
	/** @type {{ titel: string, class?: string }} */
	let { titel, class: className = '' } = $props();
</script>

<h3 class="mb-4 text-base font-medium text-on-surface {className}">{titel}</h3>
