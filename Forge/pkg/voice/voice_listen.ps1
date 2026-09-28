# pkg/voice/voice_listen.ps1
# Offline Voice Dictation Worker for Forge OS (R2 Push-to-Talk)

param(
    [int]$TimeoutSeconds = 5
)

$ErrorActionPreference = "Stop"

try {
    Add-Type -AssemblyName System.Speech
    $engine = New-Object System.Speech.Recognition.SpeechRecognitionEngine
    $grammar = New-Object System.Speech.Recognition.DictationGrammar
    $engine.LoadGrammar($grammar)
    $engine.SetInputToDefaultAudioDevice()

    $timeout = [TimeSpan]::FromSeconds($TimeoutSeconds)
    $result = $engine.Recognize($timeout)

    if ($null -ne $result -and [string]::IsNullOrWhiteSpace($result.Text) -eq $false) {
        [Console]::Out.WriteLine($result.Text)
    }
} catch {
    [Console]::Error.WriteLine("Voice Recognition Error: $_")
    exit 1
}
