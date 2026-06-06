<script>
    import { authStore } from "../../stores/authStore.svelte.js";

    $effect(() => {
        setTimeout(() => document.getElementById("login-input")?.focus(), 50);
    });
</script>

<div class="min-h-screen flex items-center justify-center p-6 bg-slate-50">
  <form onsubmit={(e) => authStore.handleLogin(e, undefined)} class="w-full max-w-md p-8 rounded-3xl bg-white border border-slate-100 shadow-xl flex flex-col items-center space-y-6 animate-fade-in no-print">
    <div class="w-16 h-16 rounded-2xl bg-slate-50 border border-slate-100 flex items-center justify-center text-slate-600"><svg xmlns="http://www.w3.org/2000/svg" class="h-8 w-8" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M12 4v1m6 11h2m-6 0h-2v4m0-11v3m0 0h.01M12 12h4.01M16 20h4M4 12h4m12 0h.01M5 8h2a1 1 0 001-1V5a1 1 0 00-1-1H5a1 1 0 00-1 1v2a1 1 0 001 1zm12 0h2a1 1 0 001-1V5a1 1 0 00-1-1h-2a1 1 0 00-1 1v2a1 1 0 001 1zM5 20h2a1 1 0 001-1v-2a1 1 0 00-1-1H5a1 1 0 00-1 1v2a1 1 0 001 1z" /></svg></div>
    <div class="text-center space-y-1.5">
      <h2 class="text-base font-bold text-slate-800">Scanner-Login erforderlich</h2>
      <p class="text-xs text-slate-400 font-medium">Scanne deine Barcode-Karte, um die Kiosk-Station freizuschalten.</p>
    </div>
    <input id="login-input" type="password" autocomplete="new-password" bind:value={authStore.loginBarcode} class="w-full bg-slate-50 border border-slate-200 rounded-xl py-3.5 px-4 text-center tracking-widest text-slate-800 focus:outline-none focus:ring-2 focus:ring-slate-500/20 focus:border-slate-300 transition-all" placeholder="••••••••••••" />
    {#if authStore.loginError}
      <p class="text-xs text-rose-500 font-semibold animate-slide-up">{authStore.loginError}</p>
    {/if}
  </form>
</div>
