
Add-Type -AssemblyName Microsoft.VisualBasic
$intent = [Microsoft.VisualBasic.Interaction]::InputBox("What did you just do? (This will be used as the AI training prompt)", "Forge Macro Recorder", "")
Write-Output $intent
