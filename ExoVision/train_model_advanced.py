import pandas as pd
import numpy as np
from sklearn.model_selection import train_test_split, GridSearchCV
from sklearn.preprocessing import StandardScaler, LabelEncoder
from sklearn.metrics import classification_report, confusion_matrix, accuracy_score
from imblearn.over_sampling import SMOTE
import xgboost as xgb
import joblib
import matplotlib.pyplot as plt
import seaborn as sns
import warnings
warnings.filterwarnings('ignore')

def main():
    print("Starting ADVANCED Exoplanet Classification Model Training (Beast Mode!)...")

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
    target_col = 'koi_disposition'
    
    # Drop identifiers and columns that could cause data leakage
    columns_to_drop = [
        'rowid', 'kepid', 'kepoi_name', 'kepler_name', 
        'koi_pdisposition', 'koi_vet_stat', 'koi_vet_date', 
        'koi_disp_prov', 'koi_comment', 'koi_sparprov',
        'koi_tce_delivname'
    ]
    df = df.drop(columns=[col for col in columns_to_drop if col in df.columns])
    
    X = df.drop(columns=[target_col])
    y = df[target_col]
    
    # Handle missing values: Median for numerical
    numerical_cols = X.select_dtypes(include=['float64', 'int64']).columns
    X[numerical_cols] = X[numerical_cols].fillna(X[numerical_cols].median())
    X[numerical_cols] = X[numerical_cols].fillna(0) # For columns entirely NaN
    
    # Handle missing values: Mode for categorical, then one-hot encode
    categorical_cols = X.select_dtypes(include=['object']).columns
    for col in categorical_cols:
        X[col] = X[col].fillna(X[col].mode()[0] if not X[col].mode().empty else 'Unknown')
        if len(X[col].unique()) < 20: 
            X = pd.get_dummies(X, columns=[col], drop_first=True)
        else:
            X = X.drop(columns=[col])
            
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

    # 6. Handle Class Imbalance using SMOTE
    print("Applying SMOTE to balance classes in the training set...")
    smote = SMOTE(random_state=42)
    X_train_resampled, y_train_resampled = smote.fit_resample(X_train_scaled, y_train)
    print(f"Original training shape: {X_train_scaled.shape}, Resampled shape: {X_train_resampled.shape}")

    # 7. Model Training (XGBoost with GridSearchCV)
    print("Training XGBoost Classifier with Hyperparameter Tuning (this might take a few minutes)...")
    
    xgb_clf = xgb.XGBClassifier(use_label_encoder=False, eval_metric='mlogloss', random_state=42)
    
    # Define a parameter grid for XGBoost (kept relatively small to run in a reasonable time)
    param_grid = {
        'n_estimators': [100, 200],
        'max_depth': [3, 5],
        'learning_rate': [0.05, 0.1],
        'subsample': [0.8, 1.0]
    }
    
    grid_search = GridSearchCV(estimator=xgb_clf, param_grid=param_grid, cv=3, scoring='accuracy', n_jobs=-1, verbose=1)
    grid_search.fit(X_train_resampled, y_train_resampled)
    
    best_xgb = grid_search.best_estimator_
    print(f"Best hyperparameters found: {grid_search.best_params_}")
    print("Training complete!")

    # 8. Evaluation
    y_pred = best_xgb.predict(X_test_scaled)
    acc = accuracy_score(y_test, y_pred)
    
    print("\n" + "="*50)
    print(f"Advanced Model (XGBoost) Accuracy: {acc * 100:.2f}%")
    print("="*50 + "\n")
    
    print("Classification Report:")
    print(classification_report(y_test, y_pred, target_names=classes))
    
    # Plotting Feature Importances
    print("Generating Feature Importance Plot...")
    importances = best_xgb.feature_importances_
    indices = np.argsort(importances)[::-1]
    top_n = 20
    
    plt.figure(figsize=(12, 8))
    plt.title(f'Top {top_n} Feature Importances (XGBoost)')
    plt.bar(range(top_n), importances[indices][:top_n], align='center')
    plt.xticks(range(top_n), [X.columns[i] for i in indices[:top_n]], rotation=90)
    plt.tight_layout()
    plt.savefig('advanced_feature_importances.png')
    print("Saved feature importances plot to 'advanced_feature_importances.png'")
    
    # Plotting Confusion Matrix
    print("Generating Confusion Matrix Plot...")
    cm = confusion_matrix(y_test, y_pred)
    plt.figure(figsize=(8, 6))
    sns.heatmap(cm, annot=True, fmt='d', cmap='Blues', xticklabels=classes, yticklabels=classes)
    plt.title('Advanced Confusion Matrix (XGBoost)')
    plt.ylabel('Actual Label')
    plt.xlabel('Predicted Label')
    plt.tight_layout()
    plt.savefig('advanced_confusion_matrix.png')
    print("Saved confusion matrix plot to 'advanced_confusion_matrix.png'")

    # 9. Save the Model and Preprocessors
    joblib.dump(best_xgb, 'exoplanet_xgboost_model.pkl')
    joblib.dump(scaler, 'advanced_scaler.pkl')
    joblib.dump(le, 'advanced_label_encoder.pkl')
    print("Saved advanced model ('exoplanet_xgboost_model.pkl'), scaler, and label encoder.")
    
    print("\nBeast mode complete! You now have a highly optimized pipeline.")

if __name__ == '__main__':
    main()
