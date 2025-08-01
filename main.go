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

func buildPrinterTable(printers []Printer, w fyne.Window) fyne.CanvasObject {
	rows := []fyne.CanvasObject{}

	// 表头
	header := container.NewGridWithColumns(4,
		widget.NewLabelWithStyle("名称", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabelWithStyle("端口类型", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabelWithStyle("Port", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabelWithStyle("旧名称", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
	)
	rows = append(rows, header)

	// 每一列打印机
	for i := range printers {
		p := &printers[i]

		btn := widget.NewButton(p.Name, func() {
			dialog.ShowEntryDialog("重命名打印机", "新名称：", func(input string) {
				if input != "" && input != p.Name {
					renamePrinter(p.Name, input)
					p.OldName = p.Name
					p.Name = input
					// 更新 UI 需要你重新 fetch 一轮
					w.SetContent(buildUI(w))
				}
			}, w)
		})

		row := container.NewGridWithColumns(4,
			btn,
			widget.NewLabel(p.PortType),
			widget.NewLabel(p.PortName),
			widget.NewLabel(p.OldName),
		)
		rows = append(rows, row)
	}

	return container.NewVScroll(container.NewVBox(rows...))
}

func buildUI(w fyne.Window) fyne.CanvasObject {
	printers := fetchPrinters()
	table := buildPrinterTable(printers, w)

	refreshBtn := widget.NewButton("刷新 🔄", func() {
		w.SetContent(buildUI(w))
	})

	return container.NewBorder(refreshBtn, nil, nil, nil, table)
}

func main() {
	a := app.New()
	w := a.NewWindow("打印机管理工具")
	w.Resize(fyne.NewSize(750, 500))

	w.SetContent(buildUI(w))
	w.ShowAndRun()
}
