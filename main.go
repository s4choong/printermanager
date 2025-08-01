//go:build windows
// +build windows

package main

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

type Printer struct {
	Name     string
	PortName string
	Driver   string
	OldName  string
}

var cachedPrinters []Printer

func getPrinters() ([]Printer, error) {
	cmd := exec.Command("powershell", "-Command",
		`Get-Printer | Select-Object Name,PortName,DriverName | ConvertTo-Json -Compress`)
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var raw []map[string]string
	if err := json.Unmarshal(out, &raw); err != nil {
		// 只有一个结果时返回 map，不是数组
		var single map[string]string
		if err := json.Unmarshal(out, &single); err != nil {
			return nil, fmt.Errorf("JSON parse failed: %v\nRaw output:\n%s", err, string(out))
		}
		raw = append(raw, single)
	}

	var printers []Printer
	for _, p := range raw {
		printers = append(printers, Printer{
			Name:     p["Name"],
			PortName: p["PortName"],
			Driver:   p["DriverName"],
			OldName:  p["Name"],
		})
	}
	return printers, nil
}

func renamePrinter(oldName, newName string) error {
	cmdText := fmt.Sprintf(`Rename-Printer -Name "%s" -NewName "%s"`, escapeQuotes(oldName), escapeQuotes(newName))
	cmd := exec.Command("powershell", "-Command", cmdText)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Rename failed: %v\n%s", err, string(out))
	}
	return nil
}

func escapeQuotes(s string) string {
	return strings.ReplaceAll(s, `"`, "`\"")
}

// mimicLabelWithCopy 返回一个 Label + Copy 按钮组合
func mimicLabelWithCopy(value string, win fyne.Window) fyne.CanvasObject {
	label := widget.NewLabel(value)
	copyBtn := widget.NewButton("📋", func() {
		win.Clipboard().SetContent(value)
	})
	copyBtn.Importance = widget.LowImportance
	copyBtn.Resize(fyne.NewSize(30, 24))
	return container.NewBorder(nil, nil, nil, copyBtn, label)
}

func buildPrinterTable(printers []Printer, w fyne.Window) fyne.CanvasObject {
	rows := []fyne.CanvasObject{}

	// 移除 Index 栏位，只保留这四项
	title := container.New(layout.NewGridLayout(4),
		widget.NewLabelWithStyle("New Name", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabelWithStyle("Current Name", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabelWithStyle("Old Name", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabelWithStyle("Port Name", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
	)
	rows = append(rows, title)

	entryMap := make(map[int]*widget.Entry)

	for i, p := range printers {
		entry := widget.NewEntry()
		entry.SetPlaceHolder("New name")
		entryMap[i] = entry

		row := container.New(layout.NewGridLayout(4),
			entry,
			mimicLabelWithCopy(p.Name, w),
			mimicLabelWithCopy(p.OldName, w),
			mimicLabelWithCopy(p.PortName, w),
		)
		rows = append(rows, row)
	}

	scroll := container.NewVScroll(container.NewVBox(rows...))
	scroll.SetMinSize(fyne.NewSize(880, 420))

	btn := widget.NewButton("Rename All", func() {
		hasError := false
		for i, p := range printers {
			newName := strings.TrimSpace(entryMap[i].Text)
			if newName != "" && newName != p.Name {
				if err := renamePrinter(p.Name, newName); err != nil {
					hasError = true
					dialog.ShowError(err, w)
					return
				}
			}
		}
		if !hasError {
			dialog.ShowInformation("Success", "✅ All printers renamed successfully.", w)
			w.SetContent(buildUI(w))
		}
	})

	return container.NewBorder(nil, btn, nil, nil, scroll)
}

func buildUI(w fyne.Window) fyne.CanvasObject {
	currentPrinters, err := getPrinters()
	if err != nil {
		return widget.NewLabel("Failed to retrieve printers: " + err.Error())
	}

	// 恢复 Old Name（来自缓存）
	for i := range currentPrinters {
		current := &currentPrinters[i]
		for _, cached := range cachedPrinters {
			if current.PortName == cached.PortName && current.Driver == cached.Driver {
				current.OldName = cached.Name
				break
			}
		}
	}
	cachedPrinters = currentPrinters
	return buildPrinterTable(currentPrinters, w)
}

func main() {
	cachedPrinters = []Printer{}
	a := app.New()
	w := a.NewWindow("Printer Manager")
	w.Resize(fyne.NewSize(960, 600))
	w.SetContent(buildUI(w))
	w.ShowAndRun()
}
