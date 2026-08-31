# Erzeugt ein ruhiges, synthetisches Musikbett (WAV, 44.1 kHz, Stereo) ohne externe Libs.
import math, struct, sys, wave, random
dauer = float(sys.argv[1]); out = sys.argv[2]
sr = 44100
random.seed(7)
# Akkordfolge (Dur/Moll-Pads), je 8 s: C – Am – F – G  (Frequenzen in Hz, tief & mittel)
akkorde = [[130.81,164.81,196.0,261.63,329.63],[110.0,130.81,164.81,220.0,261.63],[87.31,130.81,174.61,220.0,261.63],[98.0,123.47,146.83,196.0,246.94]]
takt = 8.0
n = int(dauer*sr)
frames = bytearray()
# Arpeggio-Noten (leise Glockentöne) auf Viertel
def env(x, a, r):  # einfache Attack/Release
    return min(1.0, x/a) if x < a else max(0.0, 1.0 - (x-a)/r)
for i in range(n):
    t = i/sr
    ak = akkorde[int(t//takt) % 4]; ph = (t % takt)/takt
    fade = min(1.0, t/3.0, max(0.0, (dauer - t)/4.0))
    pad = 0.0
    for k, f in enumerate(ak):
        pad += math.sin(2*math.pi*f*t + 0.3*math.sin(2*math.pi*0.11*t + k)) * (0.9 if k < 2 else 0.5)
    pad *= 0.06 * (0.85 + 0.15*math.sin(2*math.pi*0.07*t))
    # Crossfade an Akkordgrenzen
    pad *= min(1.0, ph*8) * min(1.0, (1-ph)*8) if (ph < 0.125 or ph > 0.875) else 1.0
    # Arpeggio
    beat = t % 1.0; idx = int(t) % 4
    f2 = ak[2 + (idx % 3)] * 2
    arp = math.sin(2*math.pi*f2*t) * env(beat, 0.02, 0.7) * 0.035
    l = (pad + arp*0.8) * fade; r = (pad + arp*1.2) * fade
    frames += struct.pack('<hh', int(max(-1,min(1,l))*32767), int(max(-1,min(1,r))*32767))
with wave.open(out, 'wb') as w:
    w.setnchannels(2); w.setsampwidth(2); w.setframerate(sr); w.writeframes(bytes(frames))
print('ok', out)
