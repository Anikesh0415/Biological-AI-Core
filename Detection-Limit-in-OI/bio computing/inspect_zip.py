import zipfile
import sys

def inspect_zip(file_path):
    print(f"Inspecting {file_path}")
    try:
        with zipfile.ZipFile(file_path, 'r') as z:
            for info in z.infolist()[:10]:
                print(f"File: {info.filename}, Size: {info.file_size} bytes")
    except Exception as e:
        print(f"Error reading {file_path}: {e}")

if __name__ == '__main__':
    inspect_zip(sys.argv[1])
