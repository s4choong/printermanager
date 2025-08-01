//go:build windows
// +build windows

package main

import (
	"fmt"
	"os/exec"
	"strconv"
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
	cmd := exec.Command("powershell", "Get-Printer | Select-Object Name,PortName,DriverName | Format-Table -HideTableHeaders")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(out), "\n")
	printers := []Printer{}
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		name := fields[0]
		port := fields[1]
		driver := ""
		if len(fields) >= 3 {
			driver = strings.Join(fields[2:], " ")
		}
		printers = append(printers, Printer{
			Name:     name,
			PortName: port,
			Driver:   driver,
			OldName:  name,
		})
	}
	return printers, nil
}

func renamePrinter(oldName, newName string) error {
	cmd := exec.Command("powershell", "Rename-Printer", "-Name", oldName, "-NewName", newName)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Rename failed: %v\n%s", err, string(out))
	}
	return nil
}

// mimicLabelWithCopy 返回一个 Label + Copy 按钮组合
func mimicLabelWithCopy(value string, win fyne.Window) fyne.CanvasObject {
	label := widget.NewLabel(value)
	copyBtn := widget.NewButton("📋", func() {
		clip := win.Clipboard()
		clip.SetContent(value)
	})
	copyBtn.Importance = widget.LowImportance
	copyBtn.Resize(fyne.NewSize(30, 24))
	return container.NewBorder(nil, nil, nil, copyBtn, label)
}

func buildPrinterTable(printers []Printer, w fyne.Window) fyne.CanvasObject {
	rows := []fyne.CanvasObject{}

	title := container.New(layout.NewGridLayout(5),
		widget.NewLabelWithStyle("Index", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabelWithStyle("New Name", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabelWithStyle("Current Name", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabelWithStyle("Old Name", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabelWithStyle("Port Name", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
	)
	rows = append(rows, title)

	entryMap := make(map[int]*widget.Entry)

	for i, p := range printers {
		index := i
		entry := widget.NewEntry()
		entry.SetPlaceHolder("New name")
		entryMap[index] = entry

		row := container.New(layout.NewGridLayout(5),
			widget.NewLabel(strconv.Itoa(index)),
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
				err := renamePrinter(p.Name, newName)
				if err != nil {
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

	cachedPrinters = currentPrinters // 更新缓存
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
