import pandas as pd
import numpy as np
from sklearn.model_selection import train_test_split, GridSearchCV
from sklearn.preprocessing import StandardScaler, LabelEncoder
from sklearn.ensemble import RandomForestClassifier
from sklearn.metrics import classification_report, confusion_matrix, accuracy_score
import joblib
import matplotlib.pyplot as plt
import seaborn as sns
import warnings
warnings.filterwarnings('ignore')

def main():
    print("Starting the Exoplanet Classification Model Training...")

    # 1. Load the Data
    data_path = 'KOI_Cumulative_clean.csv'
    try:
        df = pd.read_csv(data_path)
        print(f"Successfully loaded dataset with {df.shape[0]} rows and {df.shape[1]} columns.")
    except FileNotFoundError:
        print(f"Error: Could not find '{data_path}'. Please ensure it is in the same directory.")
        return

    # 2. Data Cleaning & Feature Selection
    print("Cleaning data and selecting features...")
    
    # Target variable
    target_col = 'koi_disposition'
    
    # Drop identifiers and columns that could cause data leakage
    # koi_pdisposition is a preliminary disposition which can leak the final disposition
    columns_to_drop = [
        'rowid', 'kepid', 'kepoi_name', 'kepler_name', 
        'koi_pdisposition', 'koi_vet_stat', 'koi_vet_date', 
        'koi_disp_prov', 'koi_comment', 'koi_sparprov',
        'koi_tce_delivname' # another categorical/metadata column
    ]
    
    # Drop columns if they exist in the dataframe
    df = df.drop(columns=[col for col in columns_to_drop if col in df.columns])
    
    # Separate features and target
    X = df.drop(columns=[target_col])
    y = df[target_col]
    
    # Handle missing values
    # For numerical features, we will fill missing values with the median of the column
    numerical_cols = X.select_dtypes(include=['float64', 'int64']).columns
    X[numerical_cols] = X[numerical_cols].fillna(X[numerical_cols].median())
    
    # For categorical features, fill with mode or a placeholder
    categorical_cols = X.select_dtypes(include=['object']).columns
    for col in categorical_cols:
        X[col] = X[col].fillna(X[col].mode()[0])
        # Convert categorical to numerical using one-hot encoding if any remain
        if len(X[col].unique()) < 20: # arbitrary threshold for one-hot encoding
            X = pd.get_dummies(X, columns=[col], drop_first=True)
        else:
            X = X.drop(columns=[col]) # Drop high-cardinality categorical columns
            
    print(f"Features prepared: {X.shape[1]} features.")

    # 3. Target Encoding
    le = LabelEncoder()
    y_encoded = le.fit_transform(y)
    classes = le.classes_
    print(f"Target classes identified: {classes}")

    # 4. Train/Test Split
    X_train, X_test, y_train, y_test = train_test_split(X, y_encoded, test_size=0.2, random_state=42, stratify=y_encoded)
    
    # 5. Feature Scaling
    scaler = StandardScaler()
    X_train_scaled = scaler.fit_transform(X_train)
    X_test_scaled = scaler.transform(X_test)

    # 6. Model Training (Random Forest)
    print("Training Random Forest Classifier... (this might take a minute)")
    # Using a fairly strong baseline with some default parameters that work well
    rf = RandomForestClassifier(n_estimators=100, random_state=42, n_jobs=-1)
    rf.fit(X_train_scaled, y_train)
    print("Training complete!")

    # 7. Evaluation
    y_pred = rf.predict(X_test_scaled)
    acc = accuracy_score(y_test, y_pred)
    
    print("\n" + "="*50)
    print(f"Model Accuracy: {acc * 100:.2f}%")
    print("="*50 + "\n")
    
    print("Classification Report:")
    print(classification_report(y_test, y_pred, target_names=classes))
    
    # Plotting Feature Importances
    print("Generating Feature Importance Plot...")
    importances = rf.feature_importances_
    indices = np.argsort(importances)[::-1]
    top_n = 20
    
    plt.figure(figsize=(12, 8))
    plt.title(f'Top {top_n} Feature Importances')
    plt.bar(range(top_n), importances[indices][:top_n], align='center')
    plt.xticks(range(top_n), [X.columns[i] for i in indices[:top_n]], rotation=90)
    plt.tight_layout()
    plt.savefig('feature_importances.png')
    print("Saved feature importances plot to 'feature_importances.png'")
    
    # Plotting Confusion Matrix
    print("Generating Confusion Matrix Plot...")
    cm = confusion_matrix(y_test, y_pred)
    plt.figure(figsize=(8, 6))
    sns.heatmap(cm, annot=True, fmt='d', cmap='Blues', xticklabels=classes, yticklabels=classes)
    plt.title('Confusion Matrix')
    plt.ylabel('Actual Label')
    plt.xlabel('Predicted Label')
    plt.tight_layout()
    plt.savefig('confusion_matrix.png')
    print("Saved confusion matrix plot to 'confusion_matrix.png'")

    # 8. Save the Model and Preprocessors
    joblib.dump(rf, 'exoplanet_rf_model.pkl')
    joblib.dump(scaler, 'scaler.pkl')
    joblib.dump(le, 'label_encoder.pkl')
    print("Saved model ('exoplanet_rf_model.pkl'), scaler, and label encoder.")
    
    print("\nAll tasks completed successfully! You are ready for the Devpost Challenge!")

if __name__ == '__main__':
    main()
