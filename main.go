// main.go
package main

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

type Printer struct {
	Name     string
	PortName string
	PortType string
	OldName  string
}

func detectPortType(port string) string {
	if strings.Contains(strings.ToUpper(port), "USB") {
		return "USB"
	}
	if matched, _ := regexp.MatchString(`\d+\.\d+\.\d+\.\d+`, port); matched {
		return "IP"
	}
	return port
}

func fetchPrinters() []Printer {
	cmd := exec.Command("powershell", "Get-Printer | Select-Object Name,PortName")
	out, err := cmd.Output()
	if err != nil {
		fmt.Println("PowerShell Error:", err)
		return nil
	}

	lines := strings.Split(string(out), "\n")
	var list []Printer
	for _, line := range lines[2:] {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		name := strings.Join(parts[:len(parts)-1], " ")
		port := parts[len(parts)-1]
		list = append(list, Printer{
			Name:     name,
			PortName: port,
			PortType: detectPortType(port),
			OldName:  "",
		})
	}
	return list
}

func renamePrinter(oldName, newName string) {
	cmd := exec.Command("powershell", "Rename-Printer", "-Name", oldName, "-NewName", newName)
	_ = cmd.Run()
}

func buildPrinterUI(printers []Printer, w fyne.Window) *fyne.Container {
	rows := []fyne.CanvasObject{}

	header := container.NewGridWithColumns(4,
		widget.NewLabelWithStyle("名称", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabelWithStyle("端口类型", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabelWithStyle("Port", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabelWithStyle("旧名称", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
	)
	rows = append(rows, header)

	for i := range printers {
		p := &printers[i]
		nameLabel := widget.NewLabel(p.Name)
		portType := widget.NewLabel(p.PortType)
		port := widget.NewLabel(p.PortName)
		oldName := widget.NewLabel(p.OldName)

		nameLabel.Wrapping = fyne.TextTruncate
		nameLabel.Alignment = fyne.TextAlignLeading

		nameLabel.OnTapped = func() {
			dialog.ShowEntryDialog("重命名打印机", "新名称：", func(input string) {
				if input != "" && input != p.Name {
					renamePrinter(p.Name, input)
					p.OldName = p.Name
					p.Name = input
					nameLabel.SetText(p.Name)
					oldName.SetText(p.OldName)
				}
			}, w)
		}

		row := container.NewGridWithColumns(4, nameLabel, portType, port, oldName)
		rows = append(rows, row)
	}

	return container.NewVScroll(container.NewVBox(rows...))
}

func main() {
	a := app.New()
	w := a.NewWindow("打印机管理工具")
	w.Resize(fyne.NewSize(750, 500))

	printers := fetchPrinters()
	table := buildPrinterUI(printers, w)

	refreshBtn := widget.NewButton("刷新 🔄", func() {
		newList := fetchPrinters()
		newTable := buildPrinterUI(newList, w)
		w.SetContent(container.NewBorder(refreshBtn, nil, nil, nil, newTable))
	})

	w.SetContent(container.NewBorder(refreshBtn, nil, nil, nil, table))
	w.ShowAndRun()
}
