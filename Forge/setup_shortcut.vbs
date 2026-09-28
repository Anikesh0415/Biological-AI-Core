Set WshShell = CreateObject("WScript.Shell")
strDesktop = WshShell.SpecialFolders("Desktop")
Set oShellLink = WshShell.CreateShortcut(strDesktop & "\Forge 2.0.lnk")
oShellLink.TargetPath = "E:\AIF_Project\forge.exe"
oShellLink.WorkingDirectory = "E:\AIF_Project"
oShellLink.Hotkey = "CTRL+ALT+F"
oShellLink.Description = "Forge 2.0 - Zero Idle RAM"
oShellLink.Save
