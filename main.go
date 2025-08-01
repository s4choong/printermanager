import (
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/canvas"
)

func createPrinterTable(printers []Printer, w fyne.Window) *fyne.Container {
	rows := []fyne.CanvasObject{}

	// Header
	header := container.NewHBox(
		widget.NewLabelWithStyle("名称", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		layout.NewSpacer(),
		widget.NewLabelWithStyle("端口类型", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		layout.NewSpacer(),
		widget.NewLabelWithStyle("Port Name", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		layout.NewSpacer(),
		widget.NewLabelWithStyle("旧名称", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
	)
	rows = append(rows, header)

	for i := range printers {
		p := printers[i] // for closure
		row := container.NewHBox(
			widget.NewLabel(p.Name),
			layout.NewSpacer(),
			widget.NewLabel(p.PortType),
			layout.NewSpacer(),
			widget.NewLabel(p.PortName),
			layout.NewSpacer(),
			widget.NewLabel(p.OldName),
		)

		// 双击行为：点击名称 → 弹窗重命名
		row.(*fyne.Container).Objects[0].(*widget.Label).OnTapped = func() {
			dialog.ShowEntryDialog("重命名", "输入新名称：", func(input string) {
				if input != "" && input != p.Name {
					renamePrinter(p.Name, input)
					p.OldName = p.Name
					p.Name = input
					w.Content().Refresh()
				}
			}, w)
		}

		rows = append(rows, row)
	}

	return container.NewVScroll(container.NewVBox(rows...))
}
