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

var printerList []Printer

func detectPortType(port string) string {
	if strings.Contains(strings.ToUpper(port), "USB") {
		return "USB"
	}
	ipMatch := regexp.MustCompile(`\d+\.\d+\.\d+\.\d+`)
	if ipMatch.MatchString(port) {
		return "IP"
	}
	return port
}

func fetchPrinters() []Printer {
	cmd := exec.Command("powershell", "Get-Printer | Select-Object Name,PortName")
	out, err := cmd.Output()
	if err != nil {
		fmt.Println("PowerShell error:", err)
		return nil
	}
	lines := strings.Split(string(out), "\n")
	var printers []Printer
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
		printers = append(printers, Printer{
			Name:     name,
			PortName: port,
			PortType: detectPortType(port),
			OldName:  "",
		})
	}
	return printers
}

func renamePrinter(oldName, newName string) {
	cmd := exec.Command("powershell", "Rename-Printer", "-Name", oldName, "-NewName", newName)
	_ = cmd.Run()
}

func main() {
	a := app.New()
	w := a.NewWindow("Printer Manager")
	w.Resize(fyne.NewSize(700, 400))

	table := widget.NewTable(
		func() (int, int) {
			return len(printerList) + 1, 4
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("cell")
		},
		func(i widget.TableCellID, o fyne.CanvasObject) {
			label := o.(*widget.Label)
			if i.Row == 0 {
				// Header
				headers := []string{"名称", "端口类型", "Port Name", "以前的名称"}
				label.SetText(headers[i.Col])
			} else {
				p := printerList[i.Row-1]
				switch i.Col {
				case 0:
					label.SetText(p.Name)
				case 1:
					label.SetText(p.PortType)
				case 2:
					label.SetText(p.PortName)
				case 3:
					label.SetText(p.OldName)
				}
			}
		},
	)

	table.OnSelected = func(id widget.TableCellID) {
		if id.Row == 0 || id.Col != 0 {
			return
		}
		selected := &printerList[id.Row-1]
		dialog.ShowEntryDialog("重命名打印机", "输入新名称：", func(input string) {
			if input != "" && input != selected.Name {
				renamePrinter(selected.Name, input)
				selected.OldName = selected.Name
				selected.Name = input
				table.Refresh()
			}
		}, w)
	}

	refreshBtn := widget.NewButton("🔄 刷新", func() {
		printerList = fetchPrinters()
		table.Refresh()
	})

	content := container.NewBorder(refreshBtn, nil, nil, nil, table)
	w.SetContent(content)

	printerList = fetchPrinters()
	w.ShowAndRun()
}
