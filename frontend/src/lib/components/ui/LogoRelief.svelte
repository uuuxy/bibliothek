<script>
	// Dezentes Marken-Relief für den freien Bereich unter einer Suchleiste.
	//
	// Bewusst reine Dekoration: alt="" plus aria-hidden halten es aus dem Screenreader
	// heraus, pointer-events-none aus jeder Klickfläche. Die Deckkraft bleibt bei 5–8 %,
	// damit der Textkontrast darüber unangetastet bleibt — das Relief darf den Inhalt
	// nie tragen, nur den Leerraum füllen.
	//
	// Gemeinsame Komponente, weil beide Suchbildschirme (öffentlicher Katalog und
	// Kiosk-Omnibox) dasselbe Bild zeigen sollen. Zweimal derselbe Block im Markup wäre
	// beim nächsten Feinschliff an genau einer Stelle nachgezogen worden.
	import logoUrl from '../../../assets/logo.png';
</script>

<!-- Die Bildmarke bleibt, der Schriftzug wird abgeschnitten.
     logo.png ist 97×90 px und trägt unter dem Sechseck den Schriftzug „Bibliosys". Beim
     Anschneiden am Bildschirmrand blieb davon „bliosys" bzw. „Biblios" stehen — ein
     angeschnittenes Zeichen wirkt gewollt, ein angeschnittenes Wort wirkt wie ein Fehler.
     Der Rahmen im Seitenverhältnis 97:66 zeigt deshalb nur das Sechseck; der Schriftzug
     liegt außerhalb und kann gar nicht erst halb im Bild landen. -->
<!-- EINE Marke, nicht zwei (04.08.2026). Vorher standen hier zwei Reliefs in
     unterschiedlicher Größe, Deckkraft und Drehung (8 % oben links, 5 % gedreht rechts).
     Zwei Marken auf derselben Fläche lesen sich nicht als Signatur, sondern als Textur,
     die sich nicht entschieden hat — und die Drehung machte aus der Dekoration ein zweites
     Motiv, das um Aufmerksamkeit konkurriert. Eine große, gerade gesetzte Marke ist
     ruhiger und wirkt gewollt.
     Unten rechts verankert: Dort bleibt sie aus dem Weg: Suchfeld und Trefferliste
     wachsen von oben links nach unten. -->
<div class="absolute inset-0 z-0 pointer-events-none no-print overflow-hidden" aria-hidden="true">
	<div class="absolute -bottom-16 -right-20 w-96 md:w-150 aspect-97/66 overflow-hidden">
		<img
			src={logoUrl}
			alt=""
			aria-hidden="true"
			class="logo-relief w-full opacity-[0.06] select-none"
		/>
	</div>
</div>

<style>
	/* Graustufen plus zwei gegenläufige Schatten: heller nach oben links, dunkler nach
	   unten rechts — das ergibt den Prägeeindruck, ohne eine zweite Bilddatei. */
	.logo-relief {
		filter: grayscale(100%) drop-shadow(-2px -2px 2px rgba(255, 255, 255, 0.9))
			drop-shadow(2px 2px 3px rgba(0, 0, 0, 0.12));
	}
</style>
