Dim response
Dim fso, outFile
response = InputBox("What do you want to automate?", "Forge 2.0")
Set fso = CreateObject("Scripting.FileSystemObject")
Set outFile = fso.CreateTextFile("intent.txt", True)
outFile.WriteLine(response)
outFile.Close
