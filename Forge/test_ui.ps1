Add-Type -AssemblyName PresentationFramework
$xaml = @"
<Window xmlns="http://schemas.microsoft.com/winfx/2006/xaml/presentation"
        xmlns:x="http://schemas.microsoft.com/winfx/2006/xaml"
        Title="Forge" Width="800" Height="90" 
        WindowStyle="None" AllowsTransparency="True" Background="Transparent"
        WindowStartupLocation="CenterScreen" Topmost="True">
    <Border CornerRadius="20" Background="#CC111111" BorderBrush="#33FFFFFF" BorderThickness="1.5" Margin="5">
        <Grid Margin="10,0,10,0">
            <TextBlock Name="Placeholder" Text="What do you want to automate?" Foreground="#66FFFFFF" FontSize="32" 
                       VerticalAlignment="Center" Margin="10,0,0,0" IsHitTestVisible="False" FontFamily="Segoe UI" FontWeight="Light"/>
            <TextBox Name="InputBox" Margin="5,0,5,0" Background="Transparent" Foreground="White" CaretBrush="White"
                     BorderThickness="0" FontSize="32" VerticalAlignment="Center" FontFamily="Segoe UI" FontWeight="Light"/>
        </Grid>
    </Border>
</Window>
"@
$reader = (New-Object System.Xml.XmlNodeReader ([xml]$xaml))
$win = [Windows.Markup.XamlReader]::Load($reader)
$inputBox = $win.FindName("InputBox")
$placeholder = $win.FindName("Placeholder")

$win.Add_Loaded({ $inputBox.Focus() })
$inputBox.Add_TextChanged({
    if ($inputBox.Text -eq "") {
        $placeholder.Visibility = 'Visible'
    } else {
        $placeholder.Visibility = 'Hidden'
    }
})
$inputBox.Add_KeyDown({
    if ($_.Key -eq 'Enter') {
        [Console]::Out.WriteLine($inputBox.Text)
        $win.Close()
    }
    if ($_.Key -eq 'Escape') {
        $win.Close()
    }
})
$win.ShowDialog() | Out-Null
