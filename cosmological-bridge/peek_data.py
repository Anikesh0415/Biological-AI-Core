import pandas as pd
import sys

def main():
    path = "d:/Neuro-pro/fs369data/fs369_package.hdf5"
    print("Loading data...")
    df = pd.read_hdf(path, key="/fs369_wholelife_events")
    print(df.head())
    print(df.dtypes)

if __name__ == "__main__":
    main()
