package main

import (
	"context"
	_ "embed"
	"fmt"
	"log"
	"math"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/cnkei/gospline"
	"github.com/endocrimes/keylight-go"
)

type DelayedExec struct {
	val atomic.Value
}

// UpdateVal prevents calling f() too many times.
func (d *DelayedExec) UpdateVal(i int, f func()) {
	d.val.Store(i)
	time.Sleep(700 * time.Millisecond)
	current := d.getVal()
	if current == i {
		f()
	}

}

func (d *DelayedExec) getVal() (i int) {
	return d.val.Load().(int)
}

type AllLights struct {
	sync.RWMutex
	all []*keylight.Device
}

func main() {
	a := app.New()

	timeout := time.Duration(10 * time.Second)
	ctx := context.Background()
	ctxTimeout, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	devicesCh, err := discoverLights()
	if err != nil {
		panic(err)
	}
	var allLights AllLights

	w := a.NewWindow("KeyLight Control")
	var objects []fyne.CanvasObject
	converter := newConverter()

	powerAll := widget.NewButton("Toggle Power All", func() {
		allLights.RLock()
		defer allLights.RUnlock()
		for _, d := range allLights.all {
			lg, err := d.FetchLightGroup(ctx)
			if err != nil {
				continue
			}
			d.UpdateLightGroup(ctx, togglePowerState(lg))
		}
	})
	objects = append(objects, powerAll, widget.NewSeparator())

	go func() {
		for {
			select {
			case device := <-devicesCh:
				if device == nil {
					return
				}
				allLights.Lock()
				allLights.all = append(allLights.all, device)
				allLights.Unlock()

				d := widget.NewLabel(strings.ReplaceAll(device.Name, `\`, ""))
				powerDev := widget.NewButton("Toggle Power", func() {
					lg2, err := device.FetchLightGroup(ctx)
					if err != nil {
						return
					}
					device.UpdateLightGroup(ctx, togglePowerState(lg2))
				})
				lg, err := device.FetchLightGroup(ctx)
				if err != nil {
					continue
				}
				brightness := lg.Lights[0].Brightness
				brightnessLabel := widget.NewLabel("Brightness: " + strconv.Itoa(brightness) + "%")

				brightnessButtonMinus := widget.NewButton("-", func() {
					lg2, err := device.FetchLightGroup(ctx)
					if err != nil {
						return
					}
					brightness2 := lg2.Lights[0].Brightness
					lg2.Lights[0].Brightness = brightness2 - 1
					device.UpdateLightGroup(ctx, lg2)
					brightnessLabel.SetText("Brightness: " + strconv.Itoa(brightness2-1) + "%")
				})

				brightnessButtonPlus := widget.NewButton("+", func() {
					lg2, err := device.FetchLightGroup(ctx)
					if err != nil {
						return
					}
					brightness2 := lg2.Lights[0].Brightness
					lg2.Lights[0].Brightness = brightness2 + 1
					device.UpdateLightGroup(ctx, lg2)
					brightnessLabel.SetText("Brightness: " + strconv.Itoa(brightness2+1) + "%")
				})

				brightnessSlider := widget.NewSlider(float64(3), float64(100))
				brightnessSlider.SetValue(float64(brightness))
				brightnessSlider.Step = float64(1)
				brightnessSliderdelayedExec := DelayedExec{}
				brightnessSlider.OnChanged = func(newval float64) {
					lg2, err := device.FetchLightGroup(ctx)
					if err != nil {
						return
					}
					lg2.Lights[0].Brightness = int(newval)
					go brightnessSliderdelayedExec.UpdateVal(int(newval), func() { device.UpdateLightGroup(ctx, lg2) })
					brightnessLabel.SetText("Brightness: " + strconv.Itoa(int(newval)) + "%")
				}

				temp := fmt.Sprintf("%d K", converter.ToKelvin(lg.Lights[0].Temperature))
				tempLabel := widget.NewLabel("Temperature: " + temp)

				tempSlider := widget.NewSlider(float64(2900), float64(7000))
				tempSlider.SetValue(float64(converter.ToKelvin(lg.Lights[0].Temperature)))
				tempSlider.Step = float64(50)
				tempSliderdelayedExec := DelayedExec{}
				tempSlider.OnChanged = func(newval float64) {
					device := device
					lg2, err := device.FetchLightGroup(ctx)
					if err != nil {
						return
					}
					lg2.Lights[0].Temperature = converter.FromKelvin(int(newval))
					go tempSliderdelayedExec.UpdateVal(int(newval), func() { device.UpdateLightGroup(ctx, lg2) })
					temp := fmt.Sprintf("%d K", int(newval))
					tempLabel.SetText("Temperature: " + temp)
				}

				objects = append(objects,
					d,
					powerDev,
					container.NewVBox(brightnessLabel, container.NewGridWithColumns(2, brightnessButtonMinus, brightnessButtonPlus), brightnessSlider),
					container.NewVBox(tempLabel, tempSlider),
					widget.NewSeparator(),
				)
				w.SetContent(container.NewVBox(objects...))
			case <-ctxTimeout.Done():
				return
			}
		}
	}()

	w.Show()
	a.Run()
}

func discoverLights() (<-chan *keylight.Device, error) {
	discovery, err := keylight.NewDiscovery()
	if err != nil {
		log.Println("failed to initialize keylight discovery: ", err.Error())
		return nil, err
	}

	go func() {
		err := discovery.Run(context.Background())
		if err != nil {
			log.Fatalln("Failed to discover lights: ", err.Error())
		}
	}()

	return discovery.ResultsCh(), nil
}

func isLightOn(state int) bool {
	if state == 0 {
		return true
	}
	return false
}

func togglePowerState(lg *keylight.LightGroup) *keylight.LightGroup {
	newLG := lg.Copy()
	for i, l := range lg.Lights {
		if isLightOn(l.On) {
			newLG.Lights[i].On = 1
		} else {
			newLG.Lights[i].On = 0
		}
	}
	return newLG
}

type converter struct {
	kelvin gospline.Spline
	i      gospline.Spline
}

func (c converter) ToKelvin(temp int) int {
	return int(math.Round(c.i.At(float64(temp))))
}

func (c converter) FromKelvin(kelvin int) int {
	return int(math.Round(c.kelvin.At(float64(kelvin))))
}

func newConverter() *converter {
	x := []float64{
		2900,
		3700,
		4100,
		5200,
		6000,
		7000}
	y := []float64{
		344,
		271,
		244,
		192,
		167,
		143}

	kelvin := gospline.NewCubicSpline(x, y)
	y2 := []float64{7000, 6000, 5200, 4100, 3700, 2900}
	x2 := []float64{143, 167, 192, 244, 271, 344}

	i := gospline.NewCubicSpline(x2, y2)
	return &converter{i: i, kelvin: kelvin}
}
