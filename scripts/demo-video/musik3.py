# Energetisches, lizenzfreies Musikbett mit 8-Minuten-Dramaturgie (NumPy).
# Aufruf: python3 musik3.py <dauer_s> <out.wav> [cues.json]
import sys, json, wave, numpy as np
dauer = float(sys.argv[1]); out = sys.argv[2]; cues = json.load(open(sys.argv[3])) if len(sys.argv) > 3 else []
sr = 44100; bpm = 122; beat = 60/bpm; bar = 4*beat
N = int(dauer*sr); rng = np.random.default_rng(11)
def ex(x, tau): return np.exp(-x/tau)
def put(buf, s, at, g=1.0):
    i = int(at*sr)
    if i >= N or i < 0: return
    n = min(len(s), N-i); buf[i:i+n] += s[:n]*g
# Harmonik: drei Progressionen (Akkordtöne, Bassgrundton)
def chord(root, minor=False, add7=False):
    r = root; th = r*(2**(3/12) if minor else 2**(4/12)); fi = r*2**(7/12)
    notes = [r, th, fi, r*2]
    if add7: notes.append(r*2**(10/12))
    return notes
A3,C4,D4,E4,F4,G4,B3 = 220.0,261.63,293.66,329.63,349.23,392.0,246.94
PROGS = {
 'A': [(chord(A3,True), A3/2), (chord(F4/2*2/2*2/2*2), F4/4), (chord(C4), C4/2), (chord(G4/2*2/2*2), G4/4)],   # Am F C G
 'B': [(chord(F4/2), F4/4), (chord(G4/2), G4/4), (chord(A3,True), A3/2), (chord(A3,True,True), A3/2)],       # F G Am Am7
 'C': [(chord(D4/2,True), D4/4), (chord(F4/2), F4/4), (chord(A3,True), A3/2), (chord(E4/2,True), E4/4)],     # Dm F Am Em
 'D': [(chord(C4), C4/2), (chord(G4/2), G4/4), (chord(A3,True), A3/2), (chord(F4/2), F4/4)],                # C G Am F
}
# Arrangement in 8-Takt-Blöcken: (prog, drums, bass, chords, lead, arp) 0..1
ARR = [
 ('A',0.0,0.0,0.6,0.0,0.6),  # Intro
 ('A',0.5,0.8,0.8,0.0,0.7),  # Hats+Bass
 ('A',1.0,1.0,1.0,0.0,0.8),  # Beat
 ('B',1.0,1.0,1.0,0.9,0.0),  # Lead 1
 ('C',0.35,0.6,0.9,0.0,0.9), # Break 1 (kein Kick)
 ('C',1.0,1.0,1.0,0.0,0.9),  # Beat, andere Harmonik
 ('D',1.0,1.0,1.0,1.0,0.0),  # Lead 2 (Chorus)
 ('D',1.0,1.0,1.0,1.0,0.6),  # Chorus dicht
 ('A',0.0,0.0,0.5,0.0,0.8),  # Breakdown
 ('A',0.5,0.9,0.9,0.0,0.8),  # Aufbau
 ('B',1.0,1.0,1.0,0.9,0.0),  # Lead 1 wieder
 ('C',1.0,1.0,1.0,0.0,1.0),  # Beat
 ('D',1.0,1.0,1.0,1.0,0.7),  # Chorus
 ('D',1.0,1.0,1.0,1.0,0.7),
 ('A',0.35,0.6,0.9,0.0,0.9), # Break 2
 ('A',1.0,1.0,1.0,0.0,0.9),
 ('B',1.0,1.0,1.0,0.9,0.4),
 ('D',1.0,1.0,1.0,1.0,0.7),
 ('C',0.5,0.8,0.8,0.0,1.0),  # Halbzeit-Break
 ('C',1.0,1.0,1.0,0.0,1.0),
 ('A',1.0,1.0,1.0,0.9,0.5),  # Lead 1 auf A
 ('B',1.0,1.0,1.0,0.9,0.5),
 ('D',0.35,0.6,0.9,1.0,0.0), # Lead-Break ohne Kick
 ('D',1.0,1.0,1.0,1.0,0.7),  # Chorus
 ('D',1.0,1.0,1.0,1.0,0.7),
 ('A',0.0,0.0,0.5,0.0,0.9),  # Breakdown 2
 ('A',0.5,0.9,0.9,0.0,0.9),
 ('C',1.0,1.0,1.0,0.0,1.0),
 ('B',1.0,1.0,1.0,0.9,0.6),
 ('D',1.0,1.0,1.0,1.0,0.8),  # Finale
 ('D',1.0,1.0,1.0,1.0,0.8),
 ('A',0.0,0.0,0.6,0.0,0.7),  # Outro
]
def sec(b): return ARR[min(len(ARR)-1, b//8)]
def prog(b): return PROGS[sec(b)[0]][b % 4]
nbeats = int(dauer/beat)+1; nbars = int(dauer/bar)+1
# ---- Drums
kick=np.zeros(N); snare=np.zeros(N); hat=np.zeros(N); clap=np.zeros(N)
kt=np.arange(int(0.4*sr))/sr; kick_s=np.sin(2*np.pi*(48*kt+65*np.exp(-kt*20)))*ex(kt,0.13)
st=np.arange(int(0.25*sr))/sr; snare_s=(rng.standard_normal(len(st))*0.55+np.sin(2*np.pi*185*st)*0.45)*ex(st,0.07)
ct=np.arange(int(0.3*sr))/sr; clap_s=sum(np.roll(rng.standard_normal(len(ct))*ex(ct,0.05),int(k*0.012*sr)) for k in range(3))*0.4
ht=np.arange(int(0.07*sr))/sr; hat_s=np.diff(rng.standard_normal(len(ht))*ex(ht,0.018),prepend=0)
oht=np.arange(int(0.25*sr))/sr; ohat_s=np.diff(rng.standard_normal(len(oht))*ex(oht,0.09),prepend=0)*0.7
for k in range(nbeats):
    tb=k*beat; b=int(tb//bar); _,dr,_,_,_,_=sec(b)
    if dr<=0: continue
    bar_in_block=b%8
    if dr>=0.5: put(kick,kick_s,tb,0.95 if dr==1 else 0.0)
    if k%2==1: put(snare,snare_s,tb,0.5*dr); put(clap,clap_s,tb,0.35*dr if dr==1 else 0)
    for sub in (0,0.5): put(hat,hat_s,tb+sub*beat,(0.24 if sub==0.5 else 0.11)*dr)
    if k%4==3: put(hat,ohat_s,tb+0.5*beat,0.18*dr)
    if k%8==7 and bar_in_block==7:  # Fill am Blockende
        for j in range(4): put(snare,snare_s,tb+(0.5+j*0.125)*beat,0.28*dr*(1+j*0.15))
    if dr==1 and k%16==0 and b>=16: put(kick,kick_s,tb+0.75*beat,0.5)  # Ghost-Kick
# ---- Bass
bass=np.zeros(N)
for k in range(nbeats*2):
    tb=k*beat/2; b=int(tb//bar); _,_,bg,_,_,_=sec(b)
    if bg<=0: continue
    f=prog(b)[1]*(2 if k%4==2 else 1)
    L=int(beat/2*0.92*sr); tt=np.arange(L)/sr
    saw=2*((tt*f)%1)-1; sq=np.sign(np.sin(2*np.pi*f*0.5*tt))
    s=(saw*0.7+sq*0.3)*ex(tt,0.2)*0.36*bg
    put(bass,s,tb)
bass=np.convolve(bass,np.ones(20)/20,mode='same')
# ---- Pluck-Akkorde
chords=np.zeros(N); pat=[1,0,1,0,1,0,0,1,0,1,0,1,0,0,1,0]
for k in range(nbeats*4):
    tb=k*beat/4; b=int(tb//bar); _,_,_,cg,_,_=sec(b)
    if cg<=0 or pat[k%16]==0: continue
    L=int(beat/4*1.7*sr); tt=np.arange(L)/sr; s=np.zeros(L)
    for f in prog(b)[0]: s+=(np.sin(2*np.pi*f*tt)+0.28*np.sin(2*np.pi*2*f*tt))*ex(tt,0.1)
    put(chords,s*0.105*cg,tb)
# ---- Arpeggio (16tel, hoch, weich)
arp=np.zeros(N)
for k in range(nbeats*4):
    tb=k*beat/4; b=int(tb//bar); _,_,_,_,_,ag=sec(b)
    if ag<=0: continue
    notes=prog(b)[0]; f=notes[k%len(notes)]*2*(2 if (k//16)%2 else 1)
    L=int(beat/4*1.2*sr); tt=np.arange(L)/sr
    s=np.sin(2*np.pi*f*tt)*ex(tt,0.07)*np.minimum(1,tt/0.004)
    put(arp,s*0.05*ag,tb)
# ---- Lead: zwei Melodien (Pentatonik A-Moll), je 32 Achtel
lead=np.zeros(N); penta=[440.0,523.25,587.33,659.25,783.99,880.0,1046.5]
mel1=[0,2,3,2,5,3,2,0, 1,2,3,5,3,2,1,0, 0,2,3,5,4,3,2,0, 3,5,4,3,2,1,0,0]
mel2=[5,5,4,3,5,6,5,3, 2,3,4,5,4,3,2,2, 0,2,3,4,5,4,3,2, 3,3,2,1,0,0,1,2]
for k in range(nbeats*2):
    tb=k*beat/2; b=int(tb//bar); pr,_,_,_,lg,_=sec(b)
    if lg<=0: continue
    m=mel2 if pr=='D' else mel1; f=penta[m[k%32]]
    L=int(beat/2*0.88*sr); tt=np.arange(L)/sr
    s=(np.sin(2*np.pi*f*tt)+0.3*np.sin(2*np.pi*3*f*tt)+0.22*np.sin(2*np.pi*f*1.006*tt))*np.minimum(1,tt/0.012)*ex(tt,0.28)
    put(lead,s*0.085*lg,tb)
# ---- Riser vor Drops (weißes Rauschen, ansteigend) — vor jedem Block mit drums==1 nach einem Block <1
riser=np.zeros(N)
for i in range(1,len(ARR)):
    if ARR[i][1]==1.0 and ARR[i-1][1]<1.0:
        at=i*8*bar-2*bar; L=int(2*bar*sr); tt=np.arange(L)/sr
        s=rng.standard_normal(L)*(tt/(2*bar))**2.2*0.22
        put(riser,s,at)
# ---- Sidechain-Pumpen
pump=np.ones(N)
for k in range(nbeats):
    tb=k*beat; b=int(tb//bar)
    if sec(b)[1]<1: continue
    i=int(tb*sr); L=int(0.26*sr); x=np.arange(min(L,N-i))/sr
    pump[i:i+len(x)]*=0.42+0.58*np.minimum(1,x/0.23)
mix=kick+snare*0.8+clap*0.7+hat*0.6+riser+(bass+chords+lead+arp)*pump
# ---- Whoosh/Impact an Kapitelwechseln
for c in cues:
    at=c['t']; L=int(0.9*sr); tt=np.arange(L)/sr
    sweep=rng.standard_normal(L)*np.sin(np.pi*tt/0.9)**2*0.16
    sweep=np.convolve(sweep,np.ones(40)/40,mode='same'); put(mix,sweep,max(0,at-0.6))
    put(mix,np.sin(2*np.pi*70*tt)*ex(tt,0.15)*0.45,at)
t=np.arange(N)/sr
fade=np.minimum(1,t/1.5)*np.minimum(1,np.maximum(0,(dauer-t)/4.0)); mix*=fade
mix=np.tanh(mix*1.35)*0.8
Lc=mix+0.02*np.roll(chords,300)+0.015*np.roll(arp,-200); Rc=mix-0.02*np.roll(chords,300)-0.015*np.roll(arp,-200)
stereo=np.clip(np.stack([Lc,Rc],axis=1),-0.95,0.95)
with wave.open(out,'wb') as w:
    w.setnchannels(2); w.setsampwidth(2); w.setframerate(sr); w.writeframes((stereo*32767).astype('<i2').tobytes())
print('ok',out,'Bloecke',len(ARR),'=',round(len(ARR)*8*bar),'s')
