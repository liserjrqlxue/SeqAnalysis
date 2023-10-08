package main

import (
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

func main() {
	myApp := app.New()
	myWindow := myApp.NewWindow("Hello")

	helloLabel := widget.NewLabel("Hello, Windows GUI!")

	myWindow.SetContent(container.NewVBox(
		helloLabel,
	))

	myWindow.ShowAndRun()
}
