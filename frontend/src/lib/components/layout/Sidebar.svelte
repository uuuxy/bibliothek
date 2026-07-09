<script>
  import { authStore } from "../../stores/authStore.svelte.js";
  import { uiStore } from "../../stores/uiStore.svelte.js";
  import { menuGroups, canSeeItem } from "../../menu.js";
  import { sidebarExtensions } from "../../plugins.svelte.js";
  import BackupStatusBadge from "../system/BackupStatusBadge.svelte";
  
  let systemOpen = $state(false);

  function handleLogout() {
    authStore.handleLogout(() => {
        uiStore.activeTab = "kiosk";
    });
  }

  function handleNavigate(id) {
      uiStore.activeTab = id;
      uiStore.selectedBook = null;
  }
</script>

<aside class="bg-white border-r border-slate-200 flex flex-col justify-between transition-all duration-300 no-print shrink-0 {uiStore.isSidebarCollapsed ? 'w-16' : 'w-64'} h-screen">
  <div class="flex flex-col h-full justify-between overflow-y-auto">
    <div>
      <div class="h-16 px-4 flex items-center border-b border-slate-100 shrink-0 {uiStore.isSidebarCollapsed ? 'justify-center' : 'justify-between'}">
        {#if !uiStore.isSidebarCollapsed}
          <div class="flex items-center gap-3 overflow-hidden">
            <div class="w-8 h-8 rounded-xl bg-blue-600 flex items-center justify-center text-white shrink-0 shadow-sm animate-fade-in">
              <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5"><path stroke-linecap="round" stroke-linejoin="round" d="M12 6.253v13m0-13C10.832 5.477 9.246 5 7.5 5S4.168 5.477 3 6.253v13C4.168 18.477 5.754 18 7.5 18s3.332.477 4.5 1.253m0-13C13.168 5.477 14.754 5 16.5 5c1.747 0 3.332.477 4.5 1.253v13C19.832 18.477 18.247 18 16.5 18c-1.746 0-3.332.477-4.5 1.253" /></svg>
            </div>
            <span class="font-bold text-slate-800 tracking-tight animate-fade-in">Bibliothek</span>
          </div>
          <button onclick={() => uiStore.isSidebarCollapsed = true} class="p-1.5 rounded-lg text-slate-400 hover:text-slate-600 hover:bg-slate-50 transition-colors cursor-pointer" aria-label="Navigation einklappen">
            <svg xmlns="http://www.w3.org/2000/svg" class="h-4.5 w-4.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5"><path stroke-linecap="round" stroke-linejoin="round" d="M11 19l-7-7 7-7m8 14l-7-7 7-7" /></svg>
          </button>
        {:else}
          <button onclick={() => uiStore.isSidebarCollapsed = false} class="p-1.5 rounded-lg text-slate-400 hover:text-slate-600 hover:bg-slate-50 transition-colors cursor-pointer" aria-label="Navigation ausklappen">
            <svg xmlns="http://www.w3.org/2000/svg" class="h-4.5 w-4.5 rotate-180" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5"><path stroke-linecap="round" stroke-linejoin="round" d="M11 19l-7-7 7-7m8 14l-7-7 7-7" /></svg>
          </button>
        {/if}
      </div>

      <nav class="py-6 px-3 space-y-6">
        {#each menuGroups as group}
          {#if group.items.some(item => canSeeItem(item, authStore.currentUser))}
            <div class="space-y-1">
              {#if group.name === 'System'}
                {#if !uiStore.isSidebarCollapsed}
                  <button 
                    onclick={() => systemOpen = !systemOpen} 
                    class="w-full flex items-center justify-between px-3 mb-2 text-left cursor-pointer group/sys"
                  >
                    <span class="text-[10px] font-bold text-slate-400 uppercase tracking-wider group-hover/sys:text-slate-600 transition-colors animate-fade-in">{group.name}</span>
                    <svg class="w-3.5 h-3.5 text-slate-400 group-hover/sys:text-slate-600 transition-transform duration-200 {systemOpen ? 'rotate-180' : ''}" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7"></path></svg>
                  </button>
                {/if}
                
                {#if systemOpen || uiStore.isSidebarCollapsed}
                  <div class="space-y-1 animate-fade-in">
                    {#each group.items as item}
                      {#if canSeeItem(item, authStore.currentUser)}
                        <button onclick={() => handleNavigate(item.id)} class="relative w-full flex items-center rounded-xl text-sm font-semibold transition-all {uiStore.isSidebarCollapsed ? 'justify-center py-2.5 px-0' : 'gap-3 px-3 py-2'} {uiStore.activeTab === item.id ? 'bg-blue-50 text-blue-700 font-bold' : 'text-slate-600 hover:bg-slate-50 cursor-pointer'}" title={item.label}>
                          <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                            {#if item.icon === 'kiosk'}
                              <path stroke-linecap="round" stroke-linejoin="round" d="M12 4v1m6 11h2m-6 0h-2v4m0-11v3m0 0h.01M12 12h4.01M16 20h4M4 12h4m12 0h.01M5 8h2a1 1 0 001-1V5a1 1 0 00-1-1H5a1 1 0 00-1 1v2a1 1 0 001 1zm12 0h2a1 1 0 001-1V5a1 1 0 00-1-1h-2a1 1 0 00-1 1v2a1 1 0 001 1zM5 20h2a1 1 0 001-1v-2a1 1 0 00-1-1H5a1 1 0 00-1 1v2a1 1 0 001 1z" />
                            {:else if item.icon === 'users'}
                              <path stroke-linecap="round" stroke-linejoin="round" d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z" />
                            {:else if item.icon === 'book'}
                              <path stroke-linecap="round" stroke-linejoin="round" d="M12 6.253v13m0-13C10.832 5.477 9.246 5 7.5 5S4.168 5.477 3 6.253v13C4.168 18.477 5.754 18 7.5 18s3.332.477 4.5 1.253m0-13C13.168 5.477 14.754 5 16.5 5c1.747 0 3.332.477 4.5 1.253v13C19.832 18.477 18.247 18 16.5 18c-1.746 0-3.332.477-4.5 1.253" />
                            {:else if item.icon === 'clipboard'}
                              <path stroke-linecap="round" stroke-linejoin="round" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-6 9l2 2 4-4" />
                            {:else if item.icon === 'shopping-bag'}
                              <path stroke-linecap="round" stroke-linejoin="round" d="M16 11V7a4 4 0 00-8 0v4M5 9h14l1 12H4L5 9z" />
                            {:else if item.icon === 'academic-cap'}
                              <path stroke-linecap="round" stroke-linejoin="round" d="M12 14l9-5-9-5-9 5 9 5z" /><path stroke-linecap="round" stroke-linejoin="round" d="M12 14l6.16-3.422a12.083 12.083 0 01.665 6.479A11.952 11.952 0 0012 20.055a11.952 11.952 0 00-6.824-2.998 12.078 12.078 0 01.665-6.479L12 14z" />
                            {:else if item.icon === 'chart-bar'}
                              <path stroke-linecap="round" stroke-linejoin="round" d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10a2 2 0 002 2h2a2 2 0 002-2V5a2 2 0 00-2-2h-2a2 2 0 00-2 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
                            {:else if item.icon === 'clock'}
                              <path stroke-linecap="round" stroke-linejoin="round" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
                            {:else if item.icon === 'identification'}
                              <path stroke-linecap="round" stroke-linejoin="round" d="M3 10h18M7 15h1m4 0h1m-7 4h12a3 3 0 003-3V8a3 3 0 00-3-3H6a3 3 0 00-3 3v8a3 3 0 003 3z" />
                            {:else if item.icon === 'printer'}
                              <path stroke-linecap="round" stroke-linejoin="round" d="M7 7h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
                            {:else if item.icon === 'catalog'}
                              <path stroke-linecap="round" stroke-linejoin="round" d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" />
                            {:else if item.icon === 'shield'}
                              <path stroke-linecap="round" stroke-linejoin="round" d="M9 12.75 11.25 15 15 9.75m-3-7.036A11.959 11.959 0 0 1 3.598 6 11.99 11.99 0 0 0 3 9.749c0 5.592 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.31-.21-2.571-.598-3.751h-.152c-3.196 0-6.1-1.248-8.25-3.285Z" />
                            {:else if item.icon === 'bell'}
                              <path stroke-linecap="round" stroke-linejoin="round" d="M14.857 17.082a23.848 23.848 0 0 0 5.454-1.31A8.967 8.967 0 0 1 18 9.75V9A6 6 0 0 0 6 9v.75a8.967 8.967 0 0 1-2.312 6.022c1.733.64 3.56 1.085 5.455 1.31m5.714 0a24.255 24.255 0 0 1-5.714 0m5.714 0a3 3 0 1 1-5.714 0" />
                            {:else if item.icon === 'cog'}
                              <path stroke-linecap="round" stroke-linejoin="round" d="M9.594 3.94c.09-.542.56-.94 1.11-.94h2.593c.55 0 1.02.398 1.11.94l.213 1.281c.063.374.313.686.645.87.074.04.147.083.22.127.325.196.72.257 1.075.124l1.217-.456a1.125 1.125 0 0 1 1.37.49l1.296 2.247a1.125 1.125 0 0 1-.26 1.431l-1.003.827c-.293.241-.438.613-.43.992a7.723 7.723 0 0 1 0 .255c-.008.378.137.75.43.991l1.004.827c.424.35.534.955.26 1.43l-1.298 2.247a1.125 1.125 0 0 1-1.369.491l-1.217-.456c-.355-.133-.75-.072-1.076.124a6.47 6.47 0 0 1-.22.128c-.331.183-.581.495-.644.869l-.213 1.281c-.09.543-.56.94-1.11.94h-2.594c-.55 0-1.019-.398-1.11-.94l-.213-1.281c-.062-.374-.312-.686-.644-.87a6.52 6.52 0 0 1-.22-.127c-.325-.196-.72-.257-1.076-.124l-1.217.456a1.125 1.125 0 0 1-1.369-.49l-1.297-2.247a1.125 1.125 0 0 1 .26-1.431l1.004-.827c.292-.24.437-.613.43-.991a6.932 6.932 0 0 1 0-.255c.007-.38-.138-.751-.43-.992l-1.004-.827a1.125 1.125 0 0 1-.26-1.43l1.297-2.247a1.125 1.125 0 0 1 1.37-.491l1.216.456c.356.133.751.072 1.076-.124.072-.044.146-.086.22-.128.332-.183.582-.495.644-.869l.214-1.28Z" /><path stroke-linecap="round" stroke-linejoin="round" d="M15 12a3 3 0 1 1-6 0 3 3 0 0 1 6 0Z" />
                            {/if}
                          </svg>
                          {#if !uiStore.isSidebarCollapsed}
                            <span class="animate-fade-in flex-1 text-left">{item.label}</span>
                            {#if item.id === 'orders' && uiStore.pendingReservierungen > 0}
                              <span class="ml-auto min-w-5 h-5 flex items-center justify-center rounded-full bg-rose-500 text-white text-[10px] font-bold px-1">{uiStore.pendingReservierungen}</span>
                            {/if}
                          {:else if item.id === 'orders' && uiStore.pendingReservierungen > 0}
                            <span class="absolute top-0.5 right-0.5 w-2.5 h-2.5 rounded-full bg-rose-500 ring-2 ring-white"></span>
                          {/if}
                        </button>
                      {/if}
                    {/each}
                  </div>
                {/if}
              {:else}
                {#if !uiStore.isSidebarCollapsed}
                  <span class="px-3 text-[10px] font-bold text-slate-400 uppercase tracking-wider block mb-2 animate-fade-in">{group.name}</span>
                {/if}
                {#each group.items as item}
                  {#if canSeeItem(item, authStore.currentUser)}
                    <button onclick={() => handleNavigate(item.id)} class="relative w-full flex items-center rounded-xl text-sm font-semibold transition-all {uiStore.isSidebarCollapsed ? 'justify-center py-2.5 px-0' : 'gap-3 px-3 py-2'} {uiStore.activeTab === item.id ? 'bg-blue-50 text-blue-700 font-bold' : 'text-slate-600 hover:bg-slate-50 cursor-pointer'}" title={item.label}>
                      <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                        {#if item.icon === 'kiosk'}
                          <path stroke-linecap="round" stroke-linejoin="round" d="M12 4v1m6 11h2m-6 0h-2v4m0-11v3m0 0h.01M12 12h4.01M16 20h4M4 12h4m12 0h.01M5 8h2a1 1 0 001-1V5a1 1 0 00-1-1H5a1 1 0 00-1 1v2a1 1 0 001 1zm12 0h2a1 1 0 001-1V5a1 1 0 00-1-1h-2a1 1 0 00-1 1v2a1 1 0 001 1zM5 20h2a1 1 0 001-1v-2a1 1 0 00-1-1H5a1 1 0 00-1 1v2a1 1 0 001 1z" />
                        {:else if item.icon === 'users'}
                          <path stroke-linecap="round" stroke-linejoin="round" d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z" />
                        {:else if item.icon === 'book'}
                          <path stroke-linecap="round" stroke-linejoin="round" d="M12 6.253v13m0-13C10.832 5.477 9.246 5 7.5 5S4.168 5.477 3 6.253v13C4.168 18.477 5.754 18 7.5 18s3.332.477 4.5 1.253m0-13C13.168 5.477 14.754 5 16.5 5c1.747 0 3.332.477 4.5 1.253v13C19.832 18.477 18.247 18 16.5 18c-1.746 0-3.332.477-4.5 1.253" />
                        {:else if item.icon === 'clipboard'}
                          <path stroke-linecap="round" stroke-linejoin="round" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-6 9l2 2 4-4" />
                        {:else if item.icon === 'shopping-bag'}
                          <path stroke-linecap="round" stroke-linejoin="round" d="M16 11V7a4 4 0 00-8 0v4M5 9h14l1 12H4L5 9z" />
                        {:else if item.icon === 'academic-cap'}
                          <path stroke-linecap="round" stroke-linejoin="round" d="M12 14l9-5-9-5-9 5 9 5z" /><path stroke-linecap="round" stroke-linejoin="round" d="M12 14l6.16-3.422a12.083 12.083 0 01.665 6.479A11.952 11.952 0 0012 20.055a11.952 11.952 0 00-6.824-2.998 12.078 12.078 0 01.665-6.479L12 14z" />
                        {:else if item.icon === 'chart-bar'}
                          <path stroke-linecap="round" stroke-linejoin="round" d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10a2 2 0 002 2h2a2 2 0 002-2V5a2 2 0 00-2-2h-2a2 2 0 00-2 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
                        {:else if item.icon === 'clock'}
                          <path stroke-linecap="round" stroke-linejoin="round" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
                        {:else if item.icon === 'identification'}
                          <path stroke-linecap="round" stroke-linejoin="round" d="M3 10h18M7 15h1m4 0h1m-7 4h12a3 3 0 003-3V8a3 3 0 00-3-3H6a3 3 0 00-3 3v8a3 3 0 003 3z" />
                        {:else if item.icon === 'printer'}
                          <path stroke-linecap="round" stroke-linejoin="round" d="M7 7h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
                        {:else if item.icon === 'catalog'}
                          <path stroke-linecap="round" stroke-linejoin="round" d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" />
                        {:else if item.icon === 'shield'}
                          <path stroke-linecap="round" stroke-linejoin="round" d="M9 12.75 11.25 15 15 9.75m-3-7.036A11.959 11.959 0 0 1 3.598 6 11.99 11.99 0 0 0 3 9.749c0 5.592 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.31-.21-2.571-.598-3.751h-.152c-3.196 0-6.1-1.248-8.25-3.285Z" />
                        {:else if item.icon === 'bell'}
                          <path stroke-linecap="round" stroke-linejoin="round" d="M14.857 17.082a23.848 23.848 0 0 0 5.454-1.31A8.967 8.967 0 0 1 18 9.75V9A6 6 0 0 0 6 9v.75a8.967 8.967 0 0 1-2.312 6.022c1.733.64 3.56 1.085 5.455 1.31m5.714 0a24.255 24.255 0 0 1-5.714 0m5.714 0a3 3 0 1 1-5.714 0" />
                        {:else if item.icon === 'cog'}
                          <path stroke-linecap="round" stroke-linejoin="round" d="M9.594 3.94c.09-.542.56-.94 1.11-.94h2.593c.55 0 1.02.398 1.11.94l.213 1.281c.063.374.313.686.645.87.074.04.147.083.22.127.325.196.72.257 1.075.124l1.217-.456a1.125 1.125 0 0 1 1.37.49l1.296 2.247a1.125 1.125 0 0 1-.26 1.431l-1.003.827c-.293.241-.438.613-.43.992a7.723 7.723 0 0 1 0 .255c-.008.378.137.75.43.991l1.004.827c.424.35.534.955.26 1.43l-1.298 2.247a1.125 1.125 0 0 1-1.369.491l-1.217-.456c-.355-.133-.75-.072-1.076.124a6.47 6.47 0 0 1-.22.128c-.331.183-.581.495-.644.869l-.213 1.281c-.09.543-.56.94-1.11.94h-2.594c-.55 0-1.019-.398-1.11-.94l-.213-1.281c-.062-.374-.312-.686-.644-.87a6.52 6.52 0 0 1-.22-.127c-.325-.196-.72-.257-1.076-.124l-1.217.456a1.125 1.125 0 0 1-1.369-.49l-1.297-2.247a1.125 1.125 0 0 1 .26-1.431l1.004-.827c.292-.24.437-.613.43-.991a6.932 6.932 0 0 1 0-.255c.007-.38-.138-.751-.43-.992l-1.004-.827a1.125 1.125 0 0 1-.26-1.43l1.297-2.247a1.125 1.125 0 0 1 1.37-.491l1.216.456c.356.133.751.072 1.076-.124.072-.044.146-.086.22-.128.332-.183.582-.495.644-.869l.214-1.28Z" /><path stroke-linecap="round" stroke-linejoin="round" d="M15 12a3 3 0 1 1-6 0 3 3 0 0 1 6 0Z" />
                        {/if}
                      </svg>
                      {#if !uiStore.isSidebarCollapsed}
                        <span class="animate-fade-in flex-1 text-left">{item.label}</span>
                        {#if item.id === 'orders' && uiStore.pendingReservierungen > 0}
                          <span class="ml-auto min-w-5 h-5 flex items-center justify-center rounded-full bg-rose-500 text-white text-[10px] font-bold px-1">{uiStore.pendingReservierungen}</span>
                        {/if}
                      {:else if item.id === 'orders' && uiStore.pendingReservierungen > 0}
                        <span class="absolute top-0.5 right-0.5 w-2.5 h-2.5 rounded-full bg-rose-500 ring-2 ring-white"></span>
                      {/if}
                    </button>
                  {/if}
                {/each}
              {/if}
            </div>
          {/if}
        {/each}

        {#if sidebarExtensions.length > 0}
          <div class="pt-4 border-t border-slate-100 space-y-1">
            {#if !uiStore.isSidebarCollapsed}
              <span class="px-3 text-[10px] font-bold text-slate-400 uppercase tracking-wider block mb-2">Erweiterungen</span>
            {/if}
            {#each sidebarExtensions as ext}
              {@const Component = ext.component}
              <Component {...ext.props} collapsed={uiStore.isSidebarCollapsed} />
            {/each}
          </div>
        {/if}
      </nav>
    </div>

    <div class="border-t border-slate-100 mt-auto">
      <!-- Backup-Wächter: nur Admins können das Problem beheben -->
      {#if authStore.currentUser?.rolle === "admin"}
        <BackupStatusBadge collapsed={uiStore.isSidebarCollapsed} />
      {/if}
      {#if !uiStore.isSidebarCollapsed}
        <div class="p-4 flex flex-col gap-3 animate-fade-in no-print shrink-0 text-left">
          <div class="flex flex-col">
            {#if authStore.currentUser}
              <span class="text-xs font-bold text-slate-800 truncate">
                {authStore.currentUser.vorname} {authStore.currentUser.nachname}
              </span>
              <span class="text-[10px] text-slate-500 font-medium capitalize mt-0.5">
                Rolle: {authStore.currentUser.rolle}
              </span>
            {/if}
          </div>
          <button onclick={handleLogout} class="w-full flex items-center justify-center gap-1.5 px-3 py-2 bg-rose-50 hover:bg-rose-100/60 border border-rose-100 text-rose-600 hover:text-rose-700 font-bold text-xs rounded-xl transition-all cursor-pointer">
            <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 16l4-4m0 0l-4-4m4 4H7m6 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h4a3 3 0 013 3v1"></path></svg>
            <span>Abmelden</span>
          </button>
        </div>
        <div class="px-4 pb-4 text-center no-print animate-fade-in shrink-0">
          <div class="inline-flex items-center gap-1.5 py-1 px-3 rounded-full bg-emerald-50 border border-emerald-100/50 text-emerald-700 text-[10px] font-semibold tracking-wide">
            <span>🛡️ DSGVO anonymisiert</span>
          </div>
        </div>
      {:else}
        <div class="p-4 flex flex-col items-center gap-3 no-print shrink-0">
          <div class="w-8 h-8 rounded-full bg-slate-100 border border-slate-200 flex items-center justify-center text-xs font-bold text-slate-650 cursor-default" title="{authStore.currentUser?.vorname} {authStore.currentUser?.nachname} ({authStore.currentUser?.rolle})">
            {authStore.currentUser ? (authStore.currentUser.vorname[0] + (authStore.currentUser.nachname ? authStore.currentUser.nachname[0] : '')) : 'U'}
          </div>
          <button onclick={handleLogout} class="w-8 h-8 flex items-center justify-center bg-rose-50 hover:bg-rose-100 border border-rose-100 text-rose-600 hover:text-rose-700 rounded-full transition-colors cursor-pointer" title="Abmelden">
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 16l4-4m0 0l-4-4m4 4H7m6 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h4a3 3 0 013 3v1"></path></svg>
          </button>
        </div>
        <div class="px-4 pb-4 text-center no-print flex justify-center shrink-0">
          <span class="text-emerald-600 text-sm cursor-default" title="Scans nach 14 Tagen anonymisiert">🛡️</span>
        </div>
      {/if}
    </div>
  </div>
</aside>
