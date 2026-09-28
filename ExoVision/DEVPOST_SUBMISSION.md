# Devpost Project Submission Draft

## Project Name
**ExoVision AI: High-Precision Kepler Exoplanet Classification**

## Elevator Pitch
An advanced machine learning pipeline using XGBoost, SMOTE class balancing, and hyperparameter optimization to classify NASA Kepler Exoplanet Objects with 93.88% accuracy.

---

## 🌌 Inspiration
The search for exoplanets—planets outside our solar system—is one of the most exciting frontiers in modern astrophysics. However, analyzing raw light curve signal data and distinguishing genuine exoplanet candidates from false positives (such as binary star eclipses or instrumental noise) requires immense effort. We wanted to leverage state-of-the-art machine learning to automate this process with human-expert level precision.

---

## 🪐 What it Does
ExoVision AI processes astronomical observations from NASA's Kepler Space Telescope dataset (`KOI_Cumulative_clean.csv`) and predicts whether a Kepler Object of Interest (KOI) is:
1. **CONFIRMED** Exoplanet
2. **CANDIDATE** Exoplanet
3. **FALSE POSITIVE**

By pre-processing 120+ spatial, orbital, and stellar features, our model can evaluate potential exoplanet signals in seconds.

---

## 🛠️ How We Built It
We developed an end-to-end Python machine learning pipeline:

1. **Exploratory Data Analysis & Preprocessing**:
   - Filtered out target leakage columns (e.g., preliminary disposition `koi_pdisposition`) to ensure true generalization.
   - Cleaned missing numerical features using median imputation and encoded categorical variables.
   - Standardized features with `StandardScaler`.

2. **Overcoming Class Imbalance (SMOTE)**:
   - Astronomical datasets frequently suffer from class imbalance. We applied **SMOTE (Synthetic Minority Over-sampling Technique)** to synthesize training samples for `CANDIDATE` and `CONFIRMED` classes, ensuring equal model attention across all outcomes.

3. **Model Selection & Optimization (XGBoost)**:
   - Baseline Random Forest Model achieved ~92.47% accuracy.
   - We upgraded to **XGBoost (eXtreme Gradient Boosting)** paired with **GridSearchCV** hyperparameter tuning (`n_estimators=200`, `learning_rate=0.1`, `max_depth=5`, `subsample=0.8`).

---

## 📊 Results & Accomplishments
- **Overall Accuracy**: **93.88%**
- **False Positive Detection**: **99% Precision & 99% Recall**
- **Candidate Recall Upgrade**: Improved candidate recall from **78% to 88%** via SMOTE resampling!

---

## 🔬 Key Learnings & Challenges
- **Preventing Target Leakage**: Realizing that certain metadata fields in astronomical databases contain retroactive annotations that could artificially inflate test metrics.
- **Handling High-Dimensional Data**: Working with over 120 astronomical parameters (orbital periods, transit depths, stellar radii, limb darkening coefficients).
- **Balancing Trade-offs**: Fine-tuning XGBoost depth to avoid overfitting while maximizing recall on rare candidate signals.

---

## 🚀 What's Next for ExoVision AI
- Integrating deep learning models (1D CNNs) to directly analyze raw photometric light curve time-series data.
- Expanding the model to process TESS (Transiting Exoplanet Survey Satellite) and James Webb Space Telescope (JWST) datasets.
