Add-Type -AssemblyName System.Speech

$engine = New-Object System.Speech.Recognition.SpeechRecognitionEngine
$choices = New-Object System.Speech.Recognition.Choices
$choices.Add("hey flixi")
$choices.Add("hey flexi")
$choices.Add("hey flixy")
$choices.Add("hey felix")
$choices.Add("hey lexi")
$choices.Add("hey foxy")
$choices.Add("hey trixie")
$choices.Add("hey fleecy")
$choices.Add("hey pixie")
$choices.Add("hey dixie")

$grammarBuilder = New-Object System.Speech.Recognition.GrammarBuilder
$grammarBuilder.Append($choices)
$grammar = New-Object System.Speech.Recognition.Grammar($grammarBuilder)

$engine.LoadGrammar($grammar)
$engine.SetInputToDefaultAudioDevice()

Register-ObjectEvent -InputObject $engine -EventName "SpeechRecognized" -Action {
    Invoke-RestMethod -Uri "http://127.0.0.1:18080/wake" -Method Post -ErrorAction SilentlyContinue
}

$engine.RecognizeAsync([System.Speech.Recognition.RecognizeMode]::Multiple)

# Keep the script running forever
while ($true) {
    Start-Sleep -Seconds 1
}
