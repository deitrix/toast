package toast

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"text/template"
)

type Duration string

const (
	Short Duration = "short"
	Long  Duration = "long"
)

func (d Duration) IsValid() bool {
	switch d {
	case Short, Long:
		return true
	default:
		return false
	}
}

// Notification
//
// The toast notification data. The following fields are strongly recommended;
//   - AppID
//   - Title
//
// The AppID is shown beneath the toast message (in certain cases), and above the notification within the Action
// Center - and is used to group your notifications together. It is recommended that you provide a "pretty"
// name for your app, and not something like "com.example.MyApp".
//
// If no Title is provided, but a Message is, the message will display as the toast notification's title -
// which is a slightly different font style (heavier).
//
// The Icon should be an absolute path to the icon (as the toast is invoked from a temporary path on the user's
// system, not the working directory).
//
// If you would like the toast to call an external process/open a webpage, then you can set ActivationArguments
// to the uri you would like to trigger when the toast is clicked. For example: "https://google.com" would open
// the Google homepage when the user clicks the toast notification.
// By default, clicking the toast just hides/dismisses it.
//
// The following would show a notification to the user letting them know they received an email, and opens
// gmail.com when they click the notification. It also makes the Windows 10 "mail" sound effect.
//
//	toast := toast.Notification{
//	    AppID:               "Google Mail",
//	    Title:               email.Subject,
//	    Message:             email.Preview,
//	    Icon:                "C:/Program Files/Google Mail/icons/logo.png",
//	    ActivationArguments: "https://gmail.com",
//	}
//
//	err := toast.Push()
type Notification struct {
	// The name of your app. This value shows up in Windows 10's Action Centre, so make it
	// something readable for your users. It can contain spaces, however special characters
	// (eg. é) are not supported.
	AppID string

	// The main title/heading for the toast notification.
	Title string

	// The single/multi line message to display for the toast notification.
	Message string

	// An optional path to an image on the OS to display to the left of the title & message.
	Icon string

	// The type of notification level action (like toast.Action)
	ActivationType string

	// The activation/action arguments (invoked when the user clicks the notification)
	ActivationArguments string

	// Optional action buttons to display below the notification title & message.
	Actions []Action

	// How long the toast should show up for (short/long)
	Duration Duration
}

// Action
//
// Defines an actionable button.
// See https://msdn.microsoft.com/en-us/windows/uwp/controls-and-patterns/tiles-and-notifications-adaptive-interactive-toasts for more info.
//
// Only protocol type action buttons are actually useful, as there's no way of receiving feedback from the
// user's choice. Examples of protocol type action buttons include: "bingmaps:?q=sushi" to open up Windows 10's
// maps app with a pre-populated search field set to "sushi".
//
//	toast.Action{"protocol", "Open Maps", "bingmaps:?q=sushi"}
type Action struct {
	Type      string
	Label     string
	Arguments string
}

// Push will invoke the notification.
func (n *Notification) Push() error {
	// Apply defaults
	if n.ActivationType == "" {
		n.ActivationType = "protocol"
	}
	if n.Duration == "" {
		n.Duration = Short
	}
	if !n.Duration.IsValid() {
		return fmt.Errorf("invalid duration: %s", n.Duration)
	}

	var buf bytes.Buffer
	buf.Write([]byte{0xEF, 0xBB, 0xBF})

	tmpl, err := template.New("").Parse(toastTemplate)
	if err != nil {
		return fmt.Errorf("parsing template: %w", err)
	}

	if err := tmpl.Execute(&buf, n); err != nil {
		return fmt.Errorf("executing template: %w", err)
	}

	// Create a temporary file to store the script.
	file, err := os.CreateTemp("", "toast_*.ps1")
	if err != nil {
		return fmt.Errorf("creating temp ps1 file: %w", err)
	}
	defer os.Remove(file.Name())

	// Write the script to the file.
	if _, err := file.Write(buf.Bytes()); err != nil {
		return fmt.Errorf("writing to temp file: %w", err)
	}

	// Invoke the script using PowerShell.
	cmd := exec.Command("powershell.exe", "-ExecutionPolicy", "Bypass", "-File", file.Name())
	if err = cmd.Run(); err != nil {
		return fmt.Errorf("invoking powershell script: %w", err)
	}

	return nil
}

var toastTemplate = `
[Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType = WindowsRuntime] | Out-Null
[Windows.UI.Notifications.ToastNotification, Windows.UI.Notifications, ContentType = WindowsRuntime] | Out-Null
[Windows.Data.Xml.Dom.XmlDocument, Windows.Data.Xml.Dom.XmlDocument, ContentType = WindowsRuntime] | Out-Null

$APP_ID = '{{if .AppID}}{{.AppID}}{{else}}Windows App{{end}}'

$template = @"
<toast activationType="{{.ActivationType}}" launch="{{.ActivationArguments}}" duration="{{.Duration}}">
    <visual>
        <binding template="ToastGeneric">
            {{if .Icon}}
            <image placement="appLogoOverride" src="{{.Icon}}" />
            {{end}}
            {{if .Title}}
            <text><![CDATA[{{.Title}}]]></text>
            {{end}}
            {{if .Message}}
            <text><![CDATA[{{.Message}}]]></text>
            {{end}}
        </binding>
    </visual>
	<audio src="ms-winsoundevent:Notification.Default" loop="false" />
    {{if .Actions}}
    <actions>
        {{range .Actions}}
        <action activationType="{{.Type}}" content="{{.Label}}" arguments="{{.Arguments}}" />
        {{end}}
    </actions>
    {{end}}
</toast>
"@

$xml = New-Object Windows.Data.Xml.Dom.XmlDocument
$xml.LoadXml($template)
$toast = New-Object Windows.UI.Notifications.ToastNotification $xml
[Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier($APP_ID).Show($toast)
`
