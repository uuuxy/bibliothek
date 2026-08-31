import { chromium } from '@playwright/test';
import { readdirSync, readFileSync } from 'node:fs';
const dir = process.argv[2], out = process.argv[3], per = 12;
const files = readdirSync(dir).filter(f => f.endsWith('.png') && f.startsWith('f_')).sort();
const b = await chromium.launch(); const p = await b.newPage({ viewport: { width: 1930, height: 1100 } });
for (let i = 0; i < files.length; i += per) {
  const html = `<body style="margin:0;background:#222;display:grid;grid-template-columns:repeat(4,1fr);gap:4px">${files.slice(i, i+per).map(f => `<div style="position:relative"><img src="data:image/png;base64,${readFileSync(dir+'/'+f).toString('base64')}" style="width:100%;display:block"><span style="position:absolute;left:4px;top:4px;background:#000c;color:#fff;font:14px monospace;padding:2px 6px">${f}</span></div>`).join('')}</body>`;
  await p.setContent(html); await p.waitForTimeout(500);
  await p.screenshot({ path: `${out}_${i/per+1}.png`, fullPage: true });
}
await b.close();
