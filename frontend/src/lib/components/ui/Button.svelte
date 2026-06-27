<script>
  /** @type {{
   *   children?: any,
   *   variant?: 'primary' | 'secondary' | 'danger' | 'success' | 'danger-solid' | 'ghost',
   *   size?: 'sm' | 'md' | 'lg',
   *   class?: string,
   *   [key: string]: any
   * }} */
  let { 
    children, 
    variant = 'primary', 
    size = 'md', 
    class: className = '', 
    ...rest 
  } = $props();

  const variants = {
    primary: 'bg-slate-800 text-white hover:bg-slate-900 border-transparent shadow-sm',
    secondary: 'bg-white border-slate-200 text-slate-700 hover:bg-slate-100 shadow-sm',
    danger: 'bg-rose-50 border-rose-200 text-rose-700 hover:bg-rose-100 shadow-sm',
    'danger-solid': 'bg-rose-600 text-white hover:bg-rose-700 border-transparent shadow-sm',
    success: 'bg-emerald-50 border-emerald-200 text-emerald-700 hover:bg-emerald-100 shadow-sm',
    ghost: 'bg-transparent border-transparent text-slate-600 hover:bg-slate-100 hover:text-slate-900'
  };

  const sizes = {
    sm: 'px-3 py-1.5 text-xs',
    md: 'px-4 py-2 text-sm',
    lg: 'px-6 py-3 text-base'
  };

  const baseClasses = 'inline-flex items-center justify-center gap-2 font-bold transition-all border rounded-full cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed hover:shadow focus:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 focus-visible:ring-offset-2';
</script>

<button class="{baseClasses} {sizes[size]} {variants[variant]} {className}" {...rest}>
  {@render children?.()}
</button>
