package notifier

import (
	"fmt"
	"log"
	"os/exec"
	"strings"
)

// Notifier handles sending OS-level notifications to the user.
type Notifier struct {
	appName string
}

// New creates a new Notifier with the given application name.
func New(appName string) *Notifier {
	if appName == "" {
		appName = "Planner Bot"
	}
	return &Notifier{appName: appName}
}

// Send displays a desktop notification with the given title and message.
// On Windows, it uses PowerShell to create a toast notification.
func (n *Notifier) Send(title, message string) error {
	return n.sendWindows(title, message)
}

// sendWindows creates a Windows toast notification using PowerShell.
func (n *Notifier) sendWindows(title, message string) error {
	// Escape single quotes for PowerShell
	safeTitle := strings.ReplaceAll(title, "'", "''")
	safeMessage := strings.ReplaceAll(message, "'", "''")
	safeAppName := strings.ReplaceAll(n.appName, "'", "''")

	// Use Windows 10+ toast notification via PowerShell
	script := fmt.Sprintf(`
[Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType = WindowsRuntime] > $null
[Windows.Data.Xml.Dom.XmlDocument, Windows.Data.Xml.Dom, ContentType = WindowsRuntime] > $null

$template = @"
<toast>
    <visual>
        <binding template="ToastGeneric">
            <text>%s</text>
            <text>%s</text>
        </binding>
    </visual>
    <audio src="ms-winsoundevent:Notification.Default"/>
</toast>
"@

$xml = New-Object Windows.Data.Xml.Dom.XmlDocument
$xml.LoadXml($template)
$toast = [Windows.UI.Notifications.ToastNotification]::new($xml)
[Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier('%s').Show($toast)
`, safeTitle, safeMessage, safeAppName)

	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Fallback: try simpler BurntToast or msg.exe approach
		return n.sendFallback(title, message, output, err)
	}

	return nil
}

// sendFallback tries alternative notification methods if toast fails.
func (n *Notifier) sendFallback(title, message string, priorOutput []byte, priorErr error) error {
	// Fallback 1: Try BurntToast module (if installed)
	safeTitle := strings.ReplaceAll(title, "'", "''")
	safeMessage := strings.ReplaceAll(message, "'", "''")

	btScript := fmt.Sprintf(`New-BurntToastNotification -Text '%s', '%s'`, safeTitle, safeMessage)
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", btScript)
	if err := cmd.Run(); err == nil {
		return nil
	}

	// Fallback 2: Use msg.exe (works on all Windows versions)
	msgCmd := exec.Command("msg", "*", fmt.Sprintf("%s: %s", title, message))
	if err := msgCmd.Run(); err == nil {
		return nil
	}

	// Fallback 3: Just log it to console
	log.Printf("🔔 NOTIFICATION: %s — %s", title, message)
	log.Printf("(Toast notification failed: %v, output: %s)", priorErr, string(priorOutput))

	return nil // Don't fail the whole app just because notifications aren't working
}

// SendReminder is a convenience method for sending a reminder notification.
func (n *Notifier) SendReminder(task string, dueTime string) error {
	title := "⏰ Reminder"
	message := fmt.Sprintf("%s (Due: %s)", task, dueTime)
	return n.Send(title, message)
}

// SendMissedReminder sends a notification for a missed reminder.
func (n *Notifier) SendMissedReminder(task string, originalTime string) error {
	title := "⚠️ Missed Reminder"
	message := fmt.Sprintf("%s (Was due at %s)", task, originalTime)
	return n.Send(title, message)
}
