<script>
  import { onMount } from 'svelte';
  import { apiGet, apiPut, apiPost } from '../../apiFetch.js';
  import { toastStore } from "../../stores/toastStore.svelte.js";

  let loading = $state(true);
  let saving = $state(false);
  let testing = $state(false);

  let host = $state('');
  let port = $state('');
  let user = $state('');
  let sender = $state('');
  let password = $state('');
  let hasPassword = $state(false);
  
  let testEmail = $state('');

  onMount(async () => {
    try {
      const data = await apiGet('/api/admin/settings/mail');
      host = data.smtp_host || '';
      port = data.smtp_port || '';
      user = data.smtp_user || '';
      sender = data.sender_email || '';
      hasPassword = data.has_password || false;
      testEmail = sender; // default
    } catch (e) {
      console.error(e);
    } finally {
      loading = false;
    }
  });

  async function saveConfig() {
    saving = true;
    try {
      await apiPut('/api/admin/settings/mail', {
        smtp_host: host,
        smtp_port: port,
        smtp_user: user,
        smtp_password: password,
        sender_email: sender
      });
      toastStore.addToast('Mail-Konfiguration gespeichert', 'success');
      password = '';
      if (password !== '') hasPassword = true;
    } catch (e) {
      toastStore.addToast('Fehler beim Speichern', 'error');
    } finally {
      saving = false;
    }
  }

  async function testConfig() {
    if (!testEmail) {
      toastStore.addToast('Bitte eine Test-E-Mail-Adresse angeben', 'error');
      return;
    }
    testing = true;
    try {
      await apiPost('/api/admin/settings/mail/test', {
        to: testEmail
      });
      toastStore.addToast('Test-E-Mail erfolgreich versendet', 'success');
    } catch (e) {
      toastStore.addToast('Fehler beim Testversand. Bitte Logs prüfen.', 'error');
    } finally {
      testing = false;
    }
  }
</script>

{#if loading}
  <div class="py-20 flex justify-center items-center">
    <div class="w-10 h-10 border-4 border-slate-800 border-t-transparent rounded-full animate-spin"></div>
  </div>
{:else}
  <div class="space-y-10 animate-fade-in w-full max-w-3xl py-4">
    
    <div>
      <h3 class="text-xl font-bold text-slate-900 mb-2">SMTP Server Konfiguration</h3>
      <p class="text-sm text-slate-500 mb-8">
        Hinterlege die Zugangsdaten für den E-Mail-Versand.
      </p>

      <div class="grid grid-cols-1 md:grid-cols-2 gap-x-8 gap-y-6">
        <!-- Host -->
        <div class="flex flex-col">
          <label for="smtp_host" class="text-xs font-bold text-slate-500 uppercase tracking-wider mb-2">SMTP Host</label>
          <input 
            id="smtp_host"
            type="text" 
            bind:value={host} 
            placeholder="smtp.example.com"
            class="bg-transparent border-b border-slate-200 py-2 text-slate-800 focus:border-blue-600 focus:outline-none transition-colors w-full" />
        </div>

        <!-- Port -->
        <div class="flex flex-col">
          <label for="smtp_port" class="text-xs font-bold text-slate-500 uppercase tracking-wider mb-2">SMTP Port</label>
          <input 
            id="smtp_port"
            type="text" 
            bind:value={port} 
            placeholder="587"
            class="bg-transparent border-b border-slate-200 py-2 text-slate-800 focus:border-blue-600 focus:outline-none transition-colors w-full" />
        </div>

        <!-- User -->
        <div class="flex flex-col">
          <label for="smtp_user" class="text-xs font-bold text-slate-500 uppercase tracking-wider mb-2">Benutzername</label>
          <input 
            id="smtp_user"
            type="text" 
            bind:value={user} 
            placeholder="Benutzername oder E-Mail"
            class="bg-transparent border-b border-slate-200 py-2 text-slate-800 focus:border-blue-600 focus:outline-none transition-colors w-full" />
        </div>

        <!-- Sender -->
        <div class="flex flex-col">
          <label for="smtp_sender" class="text-xs font-bold text-slate-500 uppercase tracking-wider mb-2">Absender-E-Mail</label>
          <input 
            id="smtp_sender"
            type="email" 
            bind:value={sender} 
            placeholder="noreply@bibliothek-schule.de"
            class="bg-transparent border-b border-slate-200 py-2 text-slate-800 focus:border-blue-600 focus:outline-none transition-colors w-full" />
        </div>

        <!-- Password -->
        <div class="flex flex-col md:col-span-2">
          <label for="smtp_password" class="text-xs font-bold text-slate-500 uppercase tracking-wider mb-2">Passwort</label>
          <input 
            id="smtp_password"
            type="password" 
            bind:value={password} 
            placeholder={hasPassword ? "•••••••• (Passwort ist hinterlegt)" : "Passwort eingeben"}
            class="bg-transparent border-b border-slate-200 py-2 text-slate-800 focus:border-blue-600 focus:outline-none transition-colors placeholder:text-slate-400 w-full" />
          <p class="text-[11px] text-slate-500 mt-2">Aus Sicherheitsgründen wird das gespeicherte Passwort hier nicht angezeigt. Leer lassen, um es nicht zu ändern.</p>
        </div>
      </div>

      <div class="mt-10 flex gap-4">
        <button 
          onclick={saveConfig} 
          disabled={saving}
          class="px-8 py-3 bg-slate-900 hover:bg-slate-800 text-white font-bold text-sm rounded-full transition-colors cursor-pointer disabled:opacity-60 shadow-sm">
          {saving ? 'Wird gespeichert...' : 'Speichern'}
        </button>
      </div>
    </div>

    <!-- Test Mail Section -->
    <div class="pt-10 border-t border-slate-200">
      <h3 class="text-lg font-bold text-slate-900 mb-2">Verbindung testen</h3>
      <p class="text-sm text-slate-500 mb-6">
        Sende eine Test-E-Mail an eine beliebige Adresse, um die aktuelle Konfiguration zu überprüfen.
        <br/><span class="text-amber-600 text-xs font-semibold">Hinweis: Es werden die zuletzt gespeicherten Daten für den Versuch verwendet.</span>
      </p>

      <div class="flex flex-col sm:flex-row gap-4 items-end">
        <div class="flex flex-col w-full sm:w-72">
          <label for="test_email_target" class="text-xs font-bold text-slate-500 uppercase tracking-wider mb-2">Test-Empfänger</label>
          <input 
            id="test_email_target"
            type="email" 
            bind:value={testEmail} 
            placeholder="empfaenger@schule.de"
            class="bg-transparent border-b border-slate-200 py-2 text-slate-800 focus:border-blue-600 focus:outline-none transition-colors w-full" />
        </div>
        <button 
          onclick={testConfig} 
          disabled={testing || !testEmail}
          class="px-6 py-3 bg-white hover:bg-slate-50 text-slate-800 font-bold text-sm rounded-full transition-colors cursor-pointer disabled:opacity-60 border border-slate-200 shadow-sm">
          {testing ? 'Wird gesendet...' : 'Test-E-Mail senden'}
        </button>
      </div>
    </div>

  </div>
{/if}
