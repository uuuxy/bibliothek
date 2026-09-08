<!-- @component Ladekreis — DER Ladeindikator der Anwendung (M3 circular progress, unbestimmt).

     Bis zum 08.09.2026 gab es 50 handgebaute Kreise in 45 Dateien: zehn Größen von
     12 bis 48 px, Strichstärken 2 und 4, Farben von blue-600 über slate-800, rose-600,
     emerald-600 bis zinc-950 — jede Datei hatte ihre eigene Meinung, wie „lädt"
     aussieht. M3 kennt eine: ein Bogen in `primary` auf einer schwach getönten Spur,
     auf gefüllten Flächen in der Farbe des Textes (`on-primary`).

     Drei Größen, nach Ort:
       sm 16 px  im Knopf neben der Beschriftung (Strich 2)
       md 24 px  in einer Zeile, neben einem Feld (Strich 3)
       lg 40 px  Seite oder Bereich lädt (Strich 4; M3 nennt 48, unsere Dichte 40)

     `farbe="aktuell"` nimmt currentColor — auf einem primary-Knopf wird der Kreis
     damit von selbst weiß, ohne dass der Aufrufer eine Farbe nennt. Zugänglich als
     role="progressbar" ohne Wert (= unbestimmt) mit sprechendem Namen. -->
<script>
	/** @type {{ size?: 'sm' | 'md' | 'lg', farbe?: 'primary' | 'aktuell', label?: string, class?: string }} */
	let { size = 'md', farbe = 'primary', label = 'Lädt', class: klasse = '' } = $props();
	const masse = { sm: 'h-4 w-4 border-2', md: 'h-6 w-6 border-[3px]', lg: 'h-10 w-10 border-4' };
</script>

<span
	role="progressbar"
	aria-label={label}
	class="inline-block shrink-0 animate-spin rounded-full border-current/25 border-t-current {masse[
		size
	] ?? masse.md} {farbe === 'primary' ? 'text-primary' : ''} {klasse}"
></span>
