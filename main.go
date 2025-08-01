//go:build windows
// +build windows

package main

import (
	"fmt"
	"strings"
	"syscall"
	"unsafe"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

type PRINTER_INFO_2 struct {
	pServerName         *uint16
	pPrinterName        *uint16
	pShareName          *uint16
	pPortName           *uint16
	pDriverName         *uint16
	pComment            *uint16
	pLocation           *uint16
	pDevMode            uintptr
	pSepFile            *uint16
	pPrintProcessor     *uint16
	pDatatype           *uint16
	pParameters         *uint16
	pSecurityDescriptor uintptr
	Attributes          uint32
	Priority            uint32
	DefaultPriority     uint32
	StartTime           uint32
	UntilTime           uint32
	Status              uint32
	cJobs               uint32
	AveragePPM          uint32
}

var (
	modwinspool      = syscall.NewLazyDLL("winspool.drv")
	procEnumPrinters = modwinspool.NewProc("EnumPrintersW")
	procOpenPrinter  = modwinspool.NewProc("OpenPrinterW")
	procSetPrinter   = modwinspool.NewProc("SetPrinterW")
	procClosePrinter = modwinspool.NewProc("ClosePrinter")
	procGetPrinter   = modwinspool.NewProc("GetPrinterW")
)

type Printer struct {
	Name     string
	PortName string
	Driver   string
	OldName  string
}

func UTF16PtrToString(ptr *uint16) string {
	if ptr == nil {
		return ""
	}
	return syscall.UTF16ToString((*[1 << 20]uint16)(unsafe.Pointer(ptr))[:])
}

func ListPrinters() ([]Printer, error) {
	const PRINTER_ENUM_LOCAL = 2
	var flags uint32 = PRINTER_ENUM_LOCAL

	var needed, returned uint32
	_, _, _ = procEnumPrinters.Call(
		uintptr(flags),
		0,
		2,
		0,
		0,
		uintptr(unsafe.Pointer(&needed)),
		uintptr(unsafe.Pointer(&returned)),
	)

	if needed == 0 {
		return nil, fmt.Errorf("No printer info needed")
	}

	buffer := make([]byte, needed)

	r1, _, err := procEnumPrinters.Call(
		uintptr(flags),
		0,
		2,
		uintptr(unsafe.Pointer(&buffer[0])),
		uintptr(needed),
		uintptr(unsafe.Pointer(&needed)),
		uintptr(unsafe.Pointer(&returned)),
	)

	if r1 == 0 {
		return nil, fmt.Errorf("EnumPrinters failed: %v", err)
	}

	results := make([]Printer, 0, returned)
	size := unsafe.Sizeof(PRINTER_INFO_2{})
	for i := 0; i < int(returned); i++ {
		offset := uintptr(i) * size
		info := (*PRINTER_INFO_2)(unsafe.Pointer(&buffer[offset]))
		printer := Printer{
			Name:     UTF16PtrToString(info.pPrinterName),
			PortName: UTF16PtrToString(info.pPortName),
			Driver:   UTF16PtrToString(info.pDriverName),
			OldName:  "",
		}
		results = append(results, printer)
	}

	return results, nil
}

func RenamePrinterWinAPI(oldName, newName string) error {
	var handle syscall.Handle
	oldNameUTF16, err := syscall.UTF16PtrFromString(oldName)
	if err != nil {
		return err
	}
	ret, _, err := procOpenPrinter.Call(
		uintptr(unsafe.Pointer(oldNameUTF16)),
		uintptr(unsafe.Pointer(&handle)),
		0,
	)
	if ret == 0 {
		return fmt.Errorf("OpenPrinter failed: %v", err)
	}
	defer procClosePrinter.Call(uintptr(handle))

	var needed uint32
	procGetPrinter.Call(uintptr(handle), 2, 0, 0, uintptr(unsafe.Pointer(&needed)))
	if needed == 0 {
		return fmt.Errorf("GetPrinter: no buffer size returned")
	}
	buffer := make([]byte, needed)
	ret, _, err = procGetPrinter.Call(
		uintptr(handle),
		2,
		uintptr(unsafe.Pointer(&buffer[0])),
		uintptr(needed),
		uintptr(unsafe.Pointer(&needed)),
	)
	if ret == 0 {
		return fmt.Errorf("GetPrinter failed: %v", err)
	}

	info := (*PRINTER_INFO_2)(unsafe.Pointer(&buffer[0]))
	newNameUTF16, err := syscall.UTF16PtrFromString(newName)
	if err != nil {
		return err
	}
	info.pPrinterName = newNameUTF16

	ret, _, err = procSetPrinter.Call(
		uintptr(handle),
		2,
		uintptr(unsafe.Pointer(&buffer[0])),
		0,
	)
	if ret == 0 {
		return fmt.Errorf("SetPrinter failed: %v", err)
	}
	return nil
}

func buildPrinterTable(printers []Printer, w fyne.Window) fyne.CanvasObject {
	rows := []fyne.CanvasObject{}

	header := container.NewGridWithColumns(4,
		widget.NewLabelWithStyle("名称", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabelWithStyle("端口", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabelWithStyle("驱动", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabelWithStyle("旧名称", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
	)
	rows = append(rows, header)

	for i := range printers {
		p := &printers[i]
		btn := widget.NewButton(p.Name, func() {
			entry := widget.NewEntry()
			entry.Text = p.Name
			entry.Resize(fyne.NewSize(400, 40))
			content := container.NewVBox(
				widget.NewLabel("请输入新的打印机名称："),
				entry,
			)
			dialog.ShowCustomConfirm("重命名打印机", "确定", "取消", content, func(ok bool) {
				if ok {
					input := strings.TrimSpace(entry.Text)
					if input != "" && input != p.Name {
						err := RenamePrinterWinAPI(p.Name, input)
						if err != nil {
							dialog.ShowError(err, w)
							return
						}
						p.OldName = p.Name
						p.Name = input
						w.SetContent(buildUI(w))
					}
				}
			}, w)
		})

		row := container.NewGridWithColumns(4,
			btn,
			widget.NewLabel(p.PortName),
			widget.NewLabel(p.Driver),
			widget.NewLabel(p.OldName),
		)
		rows = append(rows, row)
	}

	return container.NewVScroll(container.NewVBox(rows...))
}

func buildUI(w fyne.Window) fyne.CanvasObject {
	printers, err := ListPrinters()
	if err != nil {
		return widget.NewLabel("获取打印机失败: " + err.Error())
	}
	table := buildPrinterTable(printers, w)
	refreshBtn := widget.NewButton("刷新 🔄", func() {
		w.SetContent(buildUI(w))
	})
	return container.NewBorder(refreshBtn, nil, nil, nil, table)
}

func main() {
	a := app.New()
	w := a.NewWindow("打印机管理工具 - 完整版")
	w.Resize(fyne.NewSize(800, 600))
	w.SetContent(buildUI(w))
	w.ShowAndRun()
}
