// Aufnahme-Harness: sichtbarer Cursor, Klick-Ripple, Kapitel-Bauchbinden, ruhiges Tempo.
import { chromium } from '@playwright/test';

export const BASE = process.env.DEMO_BASE || 'http://localhost:8085';
export const W = 1920, H = 1080;
const T = Number(process.env.DEMO_TEMPO || 1); // 1 = normal, 0.5 = doppelt so schnell (Probeläufe)

const OVERLAY = `
(() => {
  const install = () => {
  if (window.__demoOverlay) return;
  const css = document.createElement('style');
  css.textContent = \`
    #__cur{position:fixed;left:0;top:0;width:22px;height:22px;pointer-events:none;z-index:2147483647;
      transform:translate(-4px,-3px);filter:drop-shadow(0 2px 3px rgba(0,0,0,.35))}
    #__cur svg{display:block}
    .__rip{position:fixed;pointer-events:none;z-index:2147483646;width:44px;height:44px;margin:-22px 0 0 -22px;
      border-radius:50%;background:rgba(37,99,235,.28);animation:__rip .55s ease-out forwards}
    @keyframes __rip{from{transform:scale(.3);opacity:1}to{transform:scale(1.6);opacity:0}}
    #__cap{position:fixed;left:300px;bottom:44px;z-index:2147483645;pointer-events:none;
      font-family:Roboto,Inter,-apple-system,"Segoe UI",sans-serif;color:#fff;
      background:linear-gradient(135deg,#1e3a8a,#2563eb);padding:18px 28px 20px 26px;border-radius:14px;
      box-shadow:0 12px 40px rgba(30,58,138,.35);opacity:0;transform:translateY(18px);
      transition:opacity .45s ease,transform .45s ease;max-width:720px}
    #__cap.on{opacity:1;transform:translateY(0)}
    #__cap .k{font-size:13px;letter-spacing:.18em;text-transform:uppercase;opacity:.8;margin-bottom:6px}
    #__cap .t{font-size:30px;font-weight:700;line-height:1.15;letter-spacing:-.01em}
    #__cap .s{font-size:17px;opacity:.92;margin-top:6px;line-height:1.35}
    #__title{position:fixed;inset:0;z-index:2147483644;display:flex;flex-direction:column;align-items:center;justify-content:center;
      background:radial-gradient(1200px 700px at 30% 20%,#1d4ed8,#0f172a 70%);color:#fff;
      font-family:Roboto,Inter,-apple-system,"Segoe UI",sans-serif;opacity:0;transition:opacity .7s ease;pointer-events:none}
    #__title.on{opacity:1}
    #__title .h{font-size:84px;font-weight:800;letter-spacing:-.02em}
    #__title .u{font-size:28px;opacity:.85;margin-top:14px;font-weight:400}
    #__title .m{font-size:16px;opacity:.6;margin-top:48px;letter-spacing:.2em;text-transform:uppercase}
    #__title ul{list-style:none;padding:0;margin:36px 0 0;columns:2;column-gap:64px;font-size:22px;line-height:1.7;opacity:.95;max-width:1200px;text-align:left}
    #__title li::before{content:'✓';color:#93c5fd;margin-right:12px;font-weight:700}
    body{transition:transform .7s cubic-bezier(.4,0,.2,1)}
  \`;
  document.documentElement.appendChild(css);
  const cur = document.createElement('div'); cur.id='__cur';
  cur.innerHTML = '<svg width="22" height="22" viewBox="0 0 24 24"><path d="M5 3l14 8.5-6.2 1.2 3.6 6.4-2.6 1.4-3.6-6.5L5 18z" fill="#111" stroke="#fff" stroke-width="1.6" stroke-linejoin="round"/></svg>';
  document.documentElement.appendChild(cur);
  let cx=-100, cy=-100;
  window.addEventListener('mousemove', e => { cx=e.clientX; cy=e.clientY; cur.style.left=cx+'px'; cur.style.top=cy+'px'; }, true);
  window.addEventListener('mousedown', e => { const r=document.createElement('div'); r.className='__rip'; r.style.left=e.clientX+'px'; r.style.top=e.clientY+'px'; document.documentElement.appendChild(r); setTimeout(()=>r.remove(),700); }, true);
  const cap = document.createElement('div'); cap.id='__cap'; document.documentElement.appendChild(cap);
  window.__caption = (k,t,s) => { if(!t){cap.classList.remove('on');return;} cap.innerHTML = '<div class="k">'+k+'</div><div class="t">'+t+'</div>'+(s?'<div class="s">'+s+'</div>':''); cap.classList.add('on'); };
  const ti = document.createElement('div'); ti.id='__title'; document.documentElement.appendChild(ti);
  window.__title = (h,u,m) => { if(!h){ti.classList.remove('on');return;} ti.innerHTML='<div class="h">'+h+'</div><div class="u">'+(u||'')+'</div>'+(m?'<div class="m">'+m+'</div>':''); ti.classList.add('on'); };
  window.__liste = (h,u,items) => { ti.innerHTML='<div class="h" style="font-size:56px">'+h+'</div><div class="u">'+(u||'')+'</div><ul>'+items.map(i=>'<li>'+i+'</li>').join('')+'</ul>'; ti.classList.add('on'); };
  window.__zoom = (x,y,f) => { const b=document.body; if(!f||f===1){b.style.transform='';b.style.transformOrigin='';return;} b.style.transformOrigin=x+'px '+y+'px'; b.style.transform='scale('+f+')'; };
  window.__demoOverlay = true;
  };
  if (document.body) install(); else document.addEventListener('DOMContentLoaded', install);
})();`;

export async function start(videoDir) {
  const browser = await chromium.launch({ headless: true, args: ['--force-device-scale-factor=1'] });
  const ctx = await browser.newContext({
    viewport: { width: W, height: H }, deviceScaleFactor: 1, locale: 'de-DE', timezoneId: 'Europe/Berlin',
    recordVideo: { dir: videoDir, size: { width: W, height: H } }
  });
  await ctx.addInitScript(OVERLAY);
  const page = await ctx.newPage();
  page.setDefaultTimeout(20000);
  const d = new Demo(page);
  return { browser, ctx, page, d };
}

export class Demo {
  constructor(page) { this.page = page; this.x = W/2; this.y = H/2; this.t0 = Date.now(); this.cues = []; }
  cue(text) { if (text) this.cues.push({ t: (Date.now() - this.t0) / 1000, text }); }
  async pause(ms) { await this.page.waitForTimeout(ms * T); }
  async ensureOverlay() { await this.page.evaluate(OVERLAY); }
  async moveTo(x, y, ms = 650) {
    const steps = Math.max(8, Math.round(ms / 16));
    await this.page.mouse.move(x, y, { steps });
    this.x = x; this.y = y;
  }
  async hover(sel, opts = {}) {
    const loc = typeof sel === 'string' ? this.page.locator(sel).first() : sel.first();
    await loc.waitFor({ state: 'visible' });
    await loc.scrollIntoViewIfNeeded();
    await this.pause(120);
    const b = await loc.boundingBox();
    if (!b) throw new Error('kein boundingBox für ' + String(sel));
    const tx = b.x + Math.min(b.width * 0.5, 200), ty = b.y + b.height / 2;
    const dist = Math.hypot(tx - this.x, ty - this.y);
    await this.moveTo(tx, ty, Math.min(1100, 300 + dist * 0.7));
    return loc;
  }
  async click(sel, { settle = 900 } = {}) {
    const loc = await this.hover(sel);
    await this.pause(180);
    await this.page.mouse.down(); await this.page.waitForTimeout(70); await this.page.mouse.up();
    await this.pause(settle);
    return loc;
  }
  // Optionaler Klick: nur wenn das Ziel jetzt sichtbar ist — kein 20-s-Warten auf Dinge, die es nicht gibt
  async clickIf(sel, opts = {}) {
    const loc = typeof sel === 'string' ? this.page.locator(sel).first() : sel.first();
    const da = await loc.isVisible({ timeout: 1500 }).catch(() => false);
    if (!da) return false;
    await this.click(loc, opts); return true;
  }
  // Klick, der sicher landet: danach muss das Ziel verschwinden, sonst Playwright-Klick als Fallback
  async clickSure(sel, { settle = 900 } = {}) {
    const loc = typeof sel === 'string' ? this.page.locator(sel).first() : sel.first();
    await this.click(loc, { settle: 400 });
    const weg = await loc.waitFor({ state: 'hidden', timeout: 2500 }).then(() => true).catch(() => false);
    if (!weg) { await loc.click({ timeout: 5000 }).catch(() => {}); }
    await this.pause(settle);
  }
  async type(sel, text, { delay = 55, settle = 700, clear = true } = {}) {
    const loc = await this.click(sel, { settle: 250 });
    if (clear) await loc.fill('');
    await loc.pressSequentially(text, { delay: delay * T });
    await this.pause(settle);
    return loc;
  }
  // Scanner: schnelle Zeichenfolge + Enter, ohne Cursorfahrt
  async scan(sel, code, { settle = 1400 } = {}) {
    const loc = typeof sel === 'string' ? this.page.locator(sel).first() : sel.first();
    await loc.focus();
    await loc.pressSequentially(code, { delay: 12 });
    await this.page.waitForTimeout(80);
    await loc.press('Enter');
    await this.pause(settle);
  }
  async key(k, settle = 600) { await this.page.keyboard.press(k); await this.pause(settle); }
  async caption(k, t, s, hold = 2600) {
    await this.ensureOverlay();
    if (t) this.cue([t, s].filter(Boolean).join(' '));
    await this.page.evaluate(([k,t,s]) => window.__caption(k,t,s), [k,t,s]);
    if (hold) await this.pause(hold);
  }
  async captionOff() { await this.page.evaluate(() => window.__caption()); await this.pause(400); }
  async title(h, u, m, hold = 3200) {
    await this.ensureOverlay();
    if (u) this.cue(h + '. ' + u);
    await this.page.evaluate(([h,u,m]) => window.__title(h,u,m), [h,u,m]);
    await this.pause(hold);
  }
  async titleOff() { await this.page.evaluate(() => window.__title()); await this.pause(800); }
  async liste(h, u, items, hold = 6000) {
    await this.ensureOverlay();
    this.cue(h + ' ' + u + ': ' + items.slice(0, 6).join(', ') + ' – und vieles mehr.');
    await this.page.evaluate(([h,u,items]) => window.__liste(h,u,items), [h,u,items]);
    await this.pause(hold);
  }
  // Zoom auf ein Element (Body wird skaliert, Overlays bleiben unskaliert)
  async zoom(sel, f = 1.5, hold = 2200) {
    const loc = typeof sel === 'string' ? this.page.locator(sel).first() : sel.first();
    await loc.waitFor({ state: 'visible' });
    const b = await loc.boundingBox();
    if (!b) return;
    const x = b.x + b.width / 2, y = b.y + b.height / 2;
    // Ursprung so wählen, dass der Bildausschnitt im Fenster bleibt
    const ox = Math.min(Math.max(x, W * 0.15), W * 0.85), oy = Math.min(Math.max(y, H * 0.15), H * 0.85);
    await this.page.evaluate(([x,y,f]) => window.__zoom(x,y,f), [ox, oy, f]);
    await this.pause(hold);
  }
  async zoomOff(hold = 1100) { await this.page.evaluate(() => window.__zoom(0,0,1)); await this.pause(hold); }
  async scroll(dy, ms = 900) { 
    const steps = 12; for (let i=0;i<steps;i++){ await this.page.mouse.wheel(0, dy/steps); await this.page.waitForTimeout(ms/steps);} await this.pause(500);
  }
  async nav(label) {
    // Sidebar-Ziel
    return this.click(this.page.locator('aside nav button', { hasText: label }), { settle: 1300 });
  }
  async openSystem() {
    const btn = this.page.locator('aside nav button[aria-expanded]');
    if ((await btn.getAttribute('aria-expanded')) !== 'true') await this.click(btn, { settle: 700 });
  }
}
