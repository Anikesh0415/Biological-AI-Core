import argparse
import queue
import sys
import sounddevice as sd
from vosk import Model, KaldiRecognizer
import json
import os

# Create a queue to hold audio data
q = queue.Queue()

def callback(indata, frames, time, status):
    if status:
        print(status, file=sys.stderr)
    q.put(bytes(indata))

def main():
    parser = argparse.ArgumentParser(add_help=False)
    parser.add_argument('-m', '--model', type=str, help='Path to the model')
    args, remaining = parser.parse_known_args()

    model_path = args.model if args.model else "model"
    if not os.path.exists(model_path):
        print(f"Please download the model from https://alphacephei.com/vosk/models and unpack as '{model_path}' in the current folder.")
        sys.exit(1)

    device_info = sd.query_devices(sd.default.device[0], 'input')
    samplerate = int(device_info['default_samplerate'])

    model = Model(model_path)
    rec = KaldiRecognizer(model, samplerate)

    with sd.RawInputStream(samplerate=samplerate, blocksize=8000, device=None, dtype='int16',
                            channels=1, callback=callback):
        while True:
            data = q.get()
            if rec.AcceptWaveform(data):
                res = json.loads(rec.Result())
                if res['text']:
                    print(res['text'])
                    sys.stdout.flush()

if __name__ == '__main__':
    try:
        main()
    except KeyboardInterrupt:
        print("\nDone")
        sys.exit(0)
    except Exception as e:
        sys.exit(type(e).__name__ + ': ' + str(e))
