import zipfile
import sys

def extract_files(zip_path, extract_dir, files_to_extract):
    with zipfile.ZipFile(zip_path, 'r') as z:
        for f in files_to_extract:
            print(f"Extracting {f}")
            z.extract(f, extract_dir)

if __name__ == '__main__':
    zip_path = sys.argv[1]
    extract_dir = sys.argv[2]
    files = sys.argv[3:]
    extract_files(zip_path, extract_dir, files)
