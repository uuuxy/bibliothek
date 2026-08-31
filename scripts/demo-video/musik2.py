# Energetisches, lizenzfreies Musikbett (Synth-Pop/EDM-light, 118 BPM), NumPy.
# Aufruf: python3 musik2.py <dauer_s> <out.wav> [cues.json]  (Cues → Whoosh/Impact an Kapitelwechseln)
import sys, json, wave, numpy as np
dauer = float(sys.argv[1]); out = sys.argv[2]; cues = json.load(open(sys.argv[3])) if len(sys.argv) > 3 else []
sr = 44100; bpm = 118; beat = 60/bpm; bar = 4*beat
N = int(dauer*sr); t = np.arange(N)/sr
rng = np.random.default_rng(3)
def note(f): return f
A2,C3,D3,E3,F3,G3 = 110.0,130.81,146.83,164.81,174.61,196.0
# Akkordfolge Am – F – C – G (je 1 Takt), Bassgrundtöne
prog = [([220.0,261.63,329.63,440.0], A2),([174.61,220.0,261.63,349.23], F3/2),([130.81*2,164.81*2,196.0*2,261.63*2], C3),([196.0,246.94,293.66,392.0], G3/2)]
mix = np.zeros(N)
def env_exp(x, tau): return np.exp(-x/tau)
# ---------- Drums
kick = np.zeros(N); snare = np.zeros(N); hat = np.zeros(N)
nbeats = int(dauer/beat)+1
kl = int(0.35*sr); kt = np.arange(kl)/sr
kick_s = np.sin(2*np.pi*(50*kt + 60*np.exp(-kt*22))) * env_exp(kt, 0.12) * 1.0
sl = int(0.22*sr); st = np.arange(sl)/sr
snare_s = (rng.standard_normal(sl)*0.5 + np.sin(2*np.pi*190*st)*0.5) * env_exp(st, 0.06)
hl = int(0.06*sr); ht = np.arange(hl)/sr
hat_s = rng.standard_normal(hl) * env_exp(ht, 0.015)
hat_s = np.diff(hat_s, prepend=0)  # hell
def put(buf, s, at, g=1.0):
    i = int(at*sr)
    if i >= N: return
    n = min(len(s), N-i); buf[i:i+n] += s[:n]*g
def section(b):  # Arrangement: Takt-Index -> (drums, bass, chords, lead)
    blk = (b//8) % 6       # 8-Takt-Blöcke, 6 Varianten im Zyklus
    if b < 4:  return (0.0, 0.0, 0.7, 0.0)   # Intro nur Akkorde
    if b < 8:  return (0.5, 0.8, 0.8, 0.0)   # Hats+Bass kommen
    if blk == 4: return (0.35, 0.6, 0.9, 1.0) # Break: Lead ohne Kick
    return (1.0, 1.0, 1.0, 0.8 if blk in (1,3,5) else 0.0)
for k in range(nbeats):
    tb = k*beat; b = int(tb//bar); dr, _, _, _ = section(b)
    if dr <= 0: continue
    if dr >= 0.5: put(kick, kick_s, tb, 0.9 if dr==1 else 0.0)
    if k % 2 == 1: put(snare, snare_s, tb, 0.55*dr)
    for sub in (0, 0.5):
        put(hat, hat_s, tb+sub*beat, (0.22 if sub==0.5 else 0.12)*dr)
    if k % 8 == 7: put(snare, snare_s, tb+0.5*beat, 0.35*dr)  # Fill
# ---------- Bass (Sägezahn, gefiltert grob per Mittelung), 8tel mit Oktavsprung
bass = np.zeros(N)
for k in range(nbeats*2):
    tb = k*beat/2; b = int(tb//bar); _, bg, _, _ = section(b)
    if bg <= 0: continue
    f = prog[b % 4][1] * (2 if k % 4 == 2 else 1)
    L = int(beat/2*0.9*sr); tt = np.arange(L)/sr
    saw = 2*((tt*f) % 1) - 1
    s = saw * env_exp(tt, 0.18) * 0.35 * bg
    put(bass, s, tb)
bass = np.convolve(bass, np.ones(24)/24, mode='same')  # Tiefpass
# ---------- Pluck-Akkorde (16tel-Rhythmus, gedämpft)
chords = np.zeros(N)
pattern = [1,0,1,0, 1,0,0,1, 0,1,0,1, 0,0,1,0]
for k in range(nbeats*4):
    tb = k*beat/4; b = int(tb//bar); _, _, cg, _ = section(b)
    if cg <= 0 or pattern[k % 16] == 0: continue
    L = int(beat/4*1.6*sr); tt = np.arange(L)/sr
    s = np.zeros(L)
    for f in prog[b % 4][0]:
        s += (np.sin(2*np.pi*f*tt) + 0.3*np.sin(2*np.pi*2*f*tt)) * env_exp(tt, 0.09)
    put(chords, s*0.11*cg, tb)
# ---------- Lead (Melodie in A-Moll-Pentatonik, 8tel, Square-ish)
lead = np.zeros(N)
penta = [440.0, 523.25, 587.33, 659.25, 783.99, 880.0]
mel = [0,2,3,2, 5,3,2,0, 1,2,3,5, 3,2,1,0, 0,2,3,5, 4,3,2,0, 3,5,4,3, 2,1,0,0]
for k in range(nbeats*2):
    tb = k*beat/2; b = int(tb//bar); _, _, _, lg = section(b)
    if lg <= 0: continue
    f = penta[mel[k % 32]]
    L = int(beat/2*0.85*sr); tt = np.arange(L)/sr
    s = (np.sin(2*np.pi*f*tt) + 0.35*np.sin(2*np.pi*3*f*tt) + 0.2*np.sin(2*np.pi*f*1.005*tt)) * np.minimum(1, tt/0.01) * env_exp(tt, 0.25)
    put(lead, s*0.09*lg, tb)
# ---------- Sidechain-Pumpen auf Kick
pump = np.ones(N)
for k in range(nbeats):
    tb = k*beat; b = int(tb//bar)
    if section(b)[0] < 1: continue
    i = int(tb*sr); L = int(0.25*sr); x = np.arange(min(L, N-i))/sr
    pump[i:i+len(x)] *= 0.45 + 0.55*np.minimum(1, x/0.22)
mix = kick*1.0 + snare*0.8 + hat*0.6 + (bass + chords + lead)*pump
# ---------- Whoosh/Impact an Kapitelwechseln
for c in cues:
    at = c['t']; L = int(0.9*sr); tt = np.arange(L)/sr
    noise = rng.standard_normal(L)
    sweep = noise * np.sin(np.pi*tt/0.9)**2 * 0.18
    sweep = np.convolve(sweep, np.ones(int(8+40*(1-tt[0])))/40, mode='same')
    put(mix, sweep, max(0, at-0.6))
    imp = np.sin(2*np.pi*(70*tt)) * env_exp(tt, 0.15) * 0.5
    put(mix, imp, at)
# ---------- Fade, Stereo, Limiter
fade = np.minimum(1, t/1.5) * np.minimum(1, np.maximum(0, (dauer - t)/3.0))
mix *= fade
mix = np.tanh(mix*1.4) * 0.8
L_ = mix + 0.02*np.roll(chords, 300); R_ = mix - 0.02*np.roll(chords, 300)
stereo = np.stack([L_, R_], axis=1)
stereo = np.clip(stereo, -0.95, 0.95)
with wave.open(out, 'wb') as w:
    w.setnchannels(2); w.setsampwidth(2); w.setframerate(sr); w.writeframes((stereo*32767).astype('<i2').tobytes())
print('ok', out)
