Add-Type -AssemblyName PresentationFramework
$xaml = @"
<Window xmlns="http://schemas.microsoft.com/winfx/2006/xaml/presentation"
        xmlns:x="http://schemas.microsoft.com/winfx/2006/xaml"
        Title="Forge 2.5" Width="600" Height="80" 
        WindowStyle="None" AllowsTransparency="True" Background="#CC000000"
        WindowStartupLocation="CenterScreen" Topmost="True">
    <Grid>
        <TextBox Name="InputBox" Margin="10" Background="Transparent" Foreground="White" 
                 BorderThickness="0" FontSize="24" HorizontalAlignment="Stretch" VerticalAlignment="Center" />
    </Grid>
</Window>
"@
$reader = (New-Object System.Xml.XmlNodeReader ([xml]$xaml))
$win = [Windows.Markup.XamlReader]::Load($reader)
$inputBox = $win.FindName("InputBox")
$inputBox.Add_KeyDown({
    if ($_.Key -eq 'Enter') {
        Write-Host $inputBox.Text
        $win.Close()
    }
    if ($_.Key -eq 'Escape') {
        $win.Close()
    }
})
$win.ShowDialog() | Out-Null
