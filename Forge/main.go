package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"

	"forge/pkg/db"
	"forge/pkg/executor"
	"forge/pkg/planner"
	"forge/pkg/recorder"
	"forge/pkg/skills"
	"forge/pkg/telegram"
	"forge/pkg/uia"
	"forge/pkg/vision"
	"forge/pkg/voice"
)

var dispatchMutex sync.Mutex

func main() {
	logFile, err := os.OpenFile("forge.log", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	if err == nil {
		os.Stdout = logFile
		os.Stderr = logFile
		defer logFile.Close()
	}
	
	db.InitBrain()
	voice.SetIntentHandler(DispatchIntent)
	recorder.StartHooks()
	skills.InitBuiltinSkills()
	skills.LoadLearnedSkills()

	// Initialize Telegram Listener if configured in environment
	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	chatIDStr := os.Getenv("TELEGRAM_CHAT_ID")
	if botToken != "" && chatIDStr != "" {
		var allowedChatID int64
		if _, err := fmt.Sscanf(chatIDStr, "%d", &allowedChatID); err == nil {
			fmt.Printf("Starting Telegram listener for chat ID: %d\n", allowedChatID)
			ctx := context.Background()
			go telegram.StartListener(ctx, botToken, allowedChatID, DispatchIntent)
		}
	}

	for {
		handleSummon()
	}
}

// DispatchIntent is the thread-safe centralized dispatcher for intent routing across UI, Telegram, and Voice.
func DispatchIntent(intent string) (string, error) {
	dispatchMutex.Lock()
	defer dispatchMutex.Unlock()

	intent = strings.TrimSpace(strings.ToLower(intent))
	if intent == "" {
		return "", fmt.Errorf("empty intent")
	}

	isLearningMode := false
	if strings.HasPrefix(intent, "learn ") {
		isLearningMode = true
		intent = strings.TrimSpace(strings.TrimPrefix(intent, "learn "))
		if strings.HasPrefix(intent, "how to ") {
			intent = strings.TrimSpace(strings.TrimPrefix(intent, "how to "))
		}
		fmt.Printf("Learning Mode activated for: '%s'\n", intent)
	}
	fmt.Printf("User Intent: %s\n\n", intent)

	if !isLearningMode {
		// 1. Check if intent matches any registered Skills
		matchedSkill := skills.MatchIntent(intent)
		if matchedSkill != nil {
			fmt.Printf("Orchestrator: Routing to Skill '%s'\n", matchedSkill.Name())
			err := matchedSkill.Execute(intent)
			if err != nil {
				return "", fmt.Errorf("skill execution failed: %w", err)
			}
			return fmt.Sprintf("Executed skill: %s", matchedSkill.Name()), nil
		}

		// 2. Legacy simple rule-based fallback for "open X" (under 3 words, no commas)
		words := strings.Fields(intent)
		if strings.HasPrefix(intent, "open ") && len(words) <= 3 && !strings.Contains(intent, ",") {
			appName := strings.TrimPrefix(intent, "open ")
			fmt.Printf("Rule-based fallback: opening '%s'\n", appName)
			actions := []executor.Action{
				{Type: "key", Key: "win"},
				{Type: "sleep", Ms: 800},
				{Type: "type", Text: appName},
				{Type: "sleep", Ms: 800},
				{Type: "key", Key: "enter"},
			}
			executor.ExecutePlan(actions)
			return fmt.Sprintf("Opened %s", appName), nil
		}
	}

	// 3. Dynamic AI Orchestrator (Planner fallback loop)
	history := "None yet"
	var allActions []executor.Action
	var lastUiaContext string
	var historyLines []string

	// Initialize the Finite State Automaton (FSA) for tracking risk across steps
	fsa := NewSafeguardFSA()

	for step := 1; step <= 15; step++ {
		fmt.Printf("\n--- Step %d ---\n", step)

		notifyUser("Forge Orchestrator", "Analyzing screen...", step, 15)
		fmt.Println("Capturing screen and analyzing via Moondream...")
		visionContext, err := vision.CaptureAndAnalyze()
		if err != nil {
			fmt.Printf("Vision failed: %v\n", err)
			visionContext = "Screen context unavailable."
		}

		fmt.Println("Extracting UI Automation Elements...")
		uiaContext, err := uia.DumpUI()
		if err != nil {
			fmt.Printf("UIA failed: %v\n", err)
			uiaContext = "[]"
		}

		if step > 1 && uiaContext == lastUiaContext {
			history += "\nFAILED: Screen unchanged. The last action did not work. Try a different element."
			notifyUser("Forge Error", "Action failed, retrying...", step, 15)
		}
		lastUiaContext = uiaContext

		fmt.Println("Planning actions...")
		actions, err := planner.PlanActions(intent, visionContext, uiaContext, history)
		if err != nil {
			fmt.Printf("Planner failed: %v\n", err)
			notifyUser("Forge Error", "Failed to plan action.", step, 15)
			return "", fmt.Errorf("planner failed: %w", err)
		}

		fmt.Printf("Generated Plan: %+v\n", actions)

		if !fsa.Evaluate(actions, intent) {
			fmt.Println("FSA Safeguard triggered: User denied permission for high-risk action. Aborting.")
			notifyUser("Forge Security", "Action aborted by user.", step, 15)
			return "Action aborted by user due to safeguard", nil
		}

		isDone := false
		var executedThisStep []executor.Action
		for _, act := range actions {
			if act.Type == "done" {
				isDone = true
				break
			}
			executedThisStep = append(executedThisStep, act)
		}

		if len(executedThisStep) > 0 {
			notifyUser("Forge Executor", fmt.Sprintf("Executing: %s", executedThisStep[0].Type), step, 15)
			fmt.Println("Executing...")
			executor.ExecutePlan(executedThisStep)
			allActions = append(allActions, executedThisStep...)

			// Build human-readable history for the tiny model
			for _, act := range executedThisStep {
				switch act.Type {
				case "key":
					historyLines = append(historyLines, fmt.Sprintf("Step %d: pressed key %s", step, act.Key))
				case "type":
					historyLines = append(historyLines, fmt.Sprintf("Step %d: typed '%s'", step, act.Text))
				case "click_element":
					historyLines = append(historyLines, fmt.Sprintf("Step %d: clicked '%s'", step, act.Name))
				case "sleep":
					historyLines = append(historyLines, fmt.Sprintf("Step %d: waited %dms", step, act.Ms))
				default:
					historyLines = append(historyLines, fmt.Sprintf("Step %d: %s", step, act.Type))
				}
			}
			history = strings.Join(historyLines, "\n")
		}

		if isDone {
			fmt.Println("Intent fully achieved. Exiting loop.")
			notifyUser("Forge Complete", "Task achieved successfully!", 15, 15)
			if isLearningMode {
				saveLearnedSkill(intent, allActions)
			}
			return fmt.Sprintf("Intent achieved in %d steps", step), nil
		}

		time.Sleep(1 * time.Second)
	}

	return "Reached maximum planning steps", nil
}

func handleSummon() {
	// 1. Pop up native WPF Input Box
	ps1Script := `
Add-Type -AssemblyName PresentationFramework
$xaml = @"
<Window xmlns="http://schemas.microsoft.com/winfx/2006/xaml/presentation"
        xmlns:x="http://schemas.microsoft.com/winfx/2006/xaml"
        Title="Forge" Width="450" Height="46" 
        WindowStyle="None" AllowsTransparency="True" Background="Transparent"
        WindowStartupLocation="CenterScreen" Topmost="True">
    <Border CornerRadius="22" Background="#CC000000" BorderBrush="#33FFFFFF" BorderThickness="1" Margin="2">
        <Grid Margin="15,0,10,0">
            <Grid.ColumnDefinitions>
                <ColumnDefinition Width="*" />
                <ColumnDefinition Width="Auto" />
            </Grid.ColumnDefinitions>
            <TextBlock Name="Placeholder" Grid.Column="0" Text="What do you want to automate?" Foreground="#88FFFFFF" FontSize="16" 
                       VerticalAlignment="Center" Margin="5,0,0,0" IsHitTestVisible="False" FontFamily="Segoe UI" FontWeight="Light"/>
            <TextBox Name="InputBox" Grid.Column="0" Margin="3,0,5,0" Padding="0,0,0,0" Background="Transparent" Foreground="White" CaretBrush="White"
                     BorderThickness="0" FontSize="16" VerticalAlignment="Center" FontFamily="Segoe UI" FontWeight="Light"/>
            <StackPanel Grid.Column="1" Orientation="Horizontal" VerticalAlignment="Center">
                <Button Name="MinimizeBtn" Content=" _ " Foreground="White" Background="Transparent" BorderThickness="0" FontSize="12" Margin="0,0,8,0" Cursor="Hand" ToolTip="Minimize"/>
                <Button Name="CloseBtn" Content=" X " Foreground="White" Background="Transparent" BorderThickness="0" FontSize="12" Cursor="Hand" ToolTip="Close"/>
            </StackPanel>
        </Grid>
    </Border>
</Window>
"@
$reader = (New-Object System.Xml.XmlNodeReader ([xml]$xaml))
$win = [Windows.Markup.XamlReader]::Load($reader)
$inputBox = $win.FindName("InputBox")
$placeholder = $win.FindName("Placeholder")
$minBtn = $win.FindName("MinimizeBtn")
$closeBtn = $win.FindName("CloseBtn")

$win.Add_MouseLeftButtonDown({ $win.DragMove() })
$minBtn.Add_Click({ $win.WindowState = 'Minimized' })
$closeBtn.Add_Click({ $win.Close() })

$win.Add_Loaded({ 
    $win.Top = 15
    $inputBox.Focus() 
})
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
`
	os.WriteFile("input.ps1", []byte(ps1Script), 0644)
	cmd := exec.Command("powershell", "-ExecutionPolicy", "Bypass", "-File", "input.ps1")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.Output()
	if err != nil {
		return
	}

	intent := strings.TrimSpace(string(out))
	if intent == "" {
		os.Exit(0)
	}

	if strings.ToLower(intent) == "exit" || strings.ToLower(intent) == "quit" {
		os.Exit(0)
	}

	res, err := DispatchIntent(intent)
	if err != nil {
		fmt.Printf("Dispatch error: %v\n", err)
	} else {
		fmt.Printf("Dispatch result: %s\n", res)
	}
}

type SecurityState int

const (
	StateSafe SecurityState = iota
	StateSuspicious
	StateElevated
	StateHighRisk
)

type SafeguardFSA struct {
	CurrentState SecurityState
}

func NewSafeguardFSA() *SafeguardFSA {
	return &SafeguardFSA{CurrentState: StateSafe}
}

// Evaluate runs the FSA across the current step's actions and tracks state over time.
func (fsa *SafeguardFSA) Evaluate(actions []executor.Action, intent string) bool {
	dangerousKeywords := []string{"delete", "remove", "pay", "buy", "transfer"}
	suspiciousApps := []string{"browser", "chrome", "edge", "powershell", "cmd", "terminal"}

	// Analyze intent to set baseline state
	lowerIntent := strings.ToLower(intent)
	for _, app := range suspiciousApps {
		if strings.Contains(lowerIntent, app) && fsa.CurrentState < StateSuspicious {
			fsa.CurrentState = StateSuspicious
		}
	}

	for _, act := range actions {
		if act.Type == "type" || act.Type == "click_element" {
			target := strings.ToLower(act.Text)
			if act.Type == "click_element" {
				target = strings.ToLower(act.Name)
			}
			
			// Typing into a suspicious app elevates the risk state
			if act.Type == "type" && fsa.CurrentState == StateSuspicious {
				fsa.CurrentState = StateElevated
			}

			// Check for explicitly dangerous keywords
			for _, kw := range dangerousKeywords {
				if strings.Contains(target, kw) {
					fsa.CurrentState = StateHighRisk
					return askUserPermission(fmt.Sprintf("FSA Triggered (High-Risk sequence detected): '%s'. Proceed?", kw))
				}
			}
		}
	}
	return true
}

func askUserPermission(msg string) bool {
	vbsScript := fmt.Sprintf(`
Dim result
result = MsgBox("%s", vbYesNo + vbExclamation + vbSystemModal, "Forge Safeguard")
If result = vbYes Then
    WScript.Quit 0
Else
    WScript.Quit 1
End If
`, msg)

	os.WriteFile("safeguard.vbs", []byte(vbsScript), 0644)
	defer os.Remove("safeguard.vbs")

	cmd := exec.Command("wscript", "//nologo", "safeguard.vbs")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	err := cmd.Run()
	
	// Exit code 0 means Yes, 1 means No
	if exitErr, ok := err.(*exec.ExitError); ok {
		return exitErr.ExitCode() == 0
	}
	return err == nil
}

// notifyUser shows a HUD overlay with optional step progress.
// Pass currentStep=0 and totalSteps=0 for a simple no-progress notification.
func notifyUser(title, message string, currentStep, totalSteps int) {
	args := []string{
		"-ExecutionPolicy", "Bypass",
		"-File", "notify.ps1",
		"-Title", title,
		"-Message", message,
	}
	if totalSteps > 0 {
		args = append(args,
			"-Step", fmt.Sprintf("Step %d", currentStep),
			"-CurrentStep", fmt.Sprintf("%d", currentStep),
			"-TotalSteps", fmt.Sprintf("%d", totalSteps),
		)
	}
	cmd := exec.Command("powershell", args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	cmd.Start() // run asynchronously
}

func saveLearnedSkill(intent string, actions []executor.Action) {
	os.Mkdir("skills_db", 0755)
	safeName := strings.ReplaceAll(intent, " ", "_")
	
	skill := skills.DynamicSkill{
		SkillName: intent,
		Actions:   actions,
	}
	
	data, _ := json.MarshalIndent(skill, "", "  ")
	os.WriteFile(fmt.Sprintf("skills_db/learned_%s.json", safeName), data, 0644)
	fmt.Printf("Successfully learned and saved skill: %s\n", intent)
	notifyUser("Forge Learn", fmt.Sprintf("Learned new skill: %s", intent), 0, 0)
}
