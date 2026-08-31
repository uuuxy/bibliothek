# Vertonung: cues.json (Zeit + Text) -> Sprecherspur (macOS say) + Musikbett -> MP4 mit Ton.
# Aufruf: python3 vertonung.py <video.webm|mp4> <cues.json> <out.mp4> [Stimme] [Rate]
import json, subprocess, sys, os, tempfile
video, cues_path, out = sys.argv[1:4]
voice = sys.argv[4] if len(sys.argv) > 4 else 'Anna'
rate = sys.argv[5] if len(sys.argv) > 5 else '165'
FF = '/opt/homebrew/bin/ffmpeg'; FP = '/opt/homebrew/bin/ffprobe'
here = os.path.dirname(os.path.abspath(__file__))
tmp = tempfile.mkdtemp(prefix='vertonung-')
def dur(f): return float(subprocess.check_output([FP,'-v','error','-show_entries','format=duration','-of','csv=p=0',f]).decode().strip())
D = dur(video)
cues = json.load(open(cues_path))
# 1) Sprachschnipsel erzeugen und einplanen (keine Überlappung: notfalls nach hinten schieben)
parts = []; ende = 0.0
for i, c in enumerate(cues):
    aiff = f'{tmp}/c{i}.aiff'; wav = f'{tmp}/c{i}.wav'
    subprocess.run(['say','-v',voice,'-r',rate,'-o',aiff,c['text']], check=True)
    subprocess.run([FF,'-loglevel','error','-y','-i',aiff,'-ar','44100','-ac','2',wav], check=True)
    start = max(c['t'] + 0.4, ende + 0.6)
    if start - c['t'] > 3: print(f'Hinweis: Cue {i} um {start-c["t"]:.1f}s verschoben ({c["text"][:40]}…)')
    l = dur(wav); parts.append((start, wav)); ende = start + l
L = max(D, ende + 2.5)   # Video ggf. mit Standbild verlängern, damit der Sprecher ausreden kann
print(f'{len(parts)} Sprachschnipsel, letzter endet bei {ende:.1f}s (Video {D:.1f}s → Ausgabe {L:.1f}s)')
# 2) Musikbett
musik = f'{tmp}/musik.wav'
subprocess.run(['python3', f'{here}/musik.py', str(L), musik], check=True)
# 3) Mischen: Stimme (adelay) -> amix; Musik unter der Stimme per Sidechain abgesenkt
inputs = ['-i', video, '-i', musik]
fc = []
for j,(st,w) in enumerate(parts):
    inputs += ['-i', w]; ms = int(st*1000)
    fc.append(f'[{j+2}:a]adelay={ms}|{ms},apad[v{j}]')
fc.append(''.join(f'[v{j}]' for j in range(len(parts))) + f'amix=inputs={len(parts)}:normalize=0:dropout_transition=0,atrim=0:{L},volume=1.6,highpass=f=80,acompressor=threshold=-18dB:ratio=3:attack=5:release=120[voice]')
fc.append('[voice]asplit[vmix][vsc]')
fc.append('[1:a][vsc]sidechaincompress=threshold=0.02:ratio=6:attack=40:release=900:makeup=1[music]')
fc.append('[music][vmix]amix=inputs=2:normalize=0:dropout_transition=0,alimiter=limit=0.95[a]')
fc.append(f'[0:v]tpad=stop_mode=clone:stop_duration={max(0.0, L - D):.2f},fade=t=out:st={L-1.2:.2f}:d=1.2[vout]')
cmd = [FF,'-loglevel','error','-y'] + inputs + ['-filter_complex', ';'.join(fc), '-map','[vout]','-map','[a]',
       '-c:v','libx264','-crf','18','-preset','medium','-pix_fmt','yuv420p','-c:a','aac','-b:a','192k','-movflags','+faststart','-t',f'{L:.2f}', out]
subprocess.run(cmd, check=True)
print('OK', out)
