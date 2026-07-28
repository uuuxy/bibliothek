<script>
	/** @type {{
	 *   children?: any,
	 *   variant?: 'primary' | 'secondary' | 'danger' | 'success' | 'danger-solid' | 'ghost',
	 *   size?: 'sm' | 'md' | 'lg',
	 *   class?: string,
	 *   [key: string]: any
	 * }} */
	let { children, variant = 'primary', size = 'md', class: className = '', ...rest } = $props();

	const variants = {
		primary: 'bg-blue-600 text-white hover:bg-blue-700 border-transparent shadow-sm',
		secondary: 'bg-white border-slate-200 text-slate-700 hover:bg-slate-100 shadow-sm',
		danger: 'bg-rose-50 border-rose-200 text-rose-700 hover:bg-rose-100 shadow-sm',
		'danger-solid': 'bg-rose-600 text-white hover:bg-rose-700 border-transparent shadow-sm',
		success: 'bg-emerald-50 border-emerald-200 text-emerald-700 hover:bg-emerald-100 shadow-sm',
		ghost:
			'bg-transparent border-transparent text-slate-600 hover:bg-slate-100 hover:text-slate-900'
	};

	// Feste Höhen statt reinem Padding: Nur so stehen Buttons neben Eingabefeldern und
	// in Tabellenzeilen auf einer Linie. Gemessen wurden vorher NEUN verschiedene
	// Button-Höhen zwischen 15 und 38 px — jede Komponente brachte ihre eigene mit.
	const sizes = {
		sm: 'h-7 px-2.5 text-xs',
		md: 'h-8 px-3 text-sm',
		lg: 'h-10 px-4 text-sm'
	};

	// rounded-md (4 px) statt rounded-full: Die Pille war das deutlichste
	// Consumer-App-Signal der Anwendung — ausgerechnet in der Komponente, die als
	// Referenz für alle anderen dient. Pillen bleiben Badges und Segmented Controls
	// vorbehalten, wo die Form eine Bedeutung trägt.
	const baseClasses =
		'inline-flex items-center justify-center gap-2 font-semibold transition-colors border rounded-md cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 focus-visible:ring-offset-2';
</script>

<button class="{baseClasses} {sizes[size]} {variants[variant]} {className}" {...rest}>
	{@render children?.()}
</button>
