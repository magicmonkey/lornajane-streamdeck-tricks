package main

import (
	"fmt"
	"image/color"
	"os/exec"
	"strconv"
	"time"

	"github.com/christopher-dG/go-obs-websocket"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/lornajane/streamdeck-tricks/actionhandlers"
	"github.com/magicmonkey/go-streamdeck"
	sdactionhandlers "github.com/magicmonkey/go-streamdeck/actionhandlers"
	buttons "github.com/magicmonkey/go-streamdeck/buttons"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"

	_ "github.com/godbus/dbus"
	belkin "github.com/magicmonkey/gobelkinwemo"
	"github.com/sqp/pulseaudio"
)

var mqtt_client mqtt.Client
var obs_client obsws.Client
var pulse *pulseaudio.Client

var buttons_wemo map[int]string // button ID and Wemo device name
var buttons_obs map[string]string // scene name and image name

// InitButtons sets up initial button prompts
func InitButtons() {
	// Initialise MQTT to use the shelf light features
	mqtt_client = connectMQTT()

	// Initialise OBS to use OBS features (requires websockets plugin in OBS)
	obs_client = connectOBS()

	if obs_client.Connected() == true {
		obs_client.AddEventHandler("SwitchScenes", func(e obsws.Event) {
			// Make sure to assert the actual event type.
			log.Info().Msg("new scene: " + e.(obsws.SwitchScenesEvent).SceneName)
		})
	}

	// WEMO plugs
	buttons_wemo = make(map[int]string)
	buttons_wemo[16] = "Christmas lights"
	buttons_wemo[17] = "Thinking light"

	go startWemoScan()

	// Get some Audio Setup
	pulse = getPulseConnection()

	// shelf lights
	abutton := buttons.NewColourButton(color.RGBA{255, 0, 255, 255})
	abutton.SetActionHandler(&actionhandlers.MQTTAction{Colour: color.RGBA{255, 0, 255, 255}, Client: mqtt_client})
	sd.AddButton(8, abutton)

	bbutton := buttons.NewColourButton(color.RGBA{0, 0, 255, 255})
	bbutton.SetActionHandler(&actionhandlers.MQTTAction{Colour: color.RGBA{0, 0, 255, 255}, Client: mqtt_client})
	sd.AddButton(9, bbutton)

	cbutton := buttons.NewColourButton(color.RGBA{255, 255, 0, 255})
	cbutton.SetActionHandler(&actionhandlers.MQTTAction{Colour: color.RGBA{255, 255, 0, 255}, Client: mqtt_client})
	sd.AddButton(10, cbutton)

	// OBS
	buttons_obs = make(map[string]string)
	buttons_obs["Camera"] = "/camera.png"
	buttons_obs["Screenshare"] = "/screen-and-cam.png"
	buttons_obs["Secrets"] = "/secrets.png"
	buttons_obs["Offline"] = "/offline.png"
	buttons_obs["Soon"] = "/soon.png"
	buttons_obs["BRB"] = "/garble.png"

	if obs_client.Connected() == true {
		// offset for what number button to start at
		offset := 0
		image_path := viper.GetString("buttons.images")
		var image string

		// what scenes do we have? (max 8)
		scene_req := obsws.NewGetSceneListRequest()
		scenes, err := scene_req.SendReceive(obs_client)
		if err != nil {
			log.Warn().Err(err)
		}
		// fmt.Printf("%#v\n", scenes.CurrentScene)
		fmt.Printf("%#v\n", scenes.Scenes[2])

		// make buttons for these scenes
		for i, scene := range scenes.Scenes {
			log.Debug().Msg("Scene: " + scene.Name)
			// default image

			image = image_path + "/play.jpg"
			if buttons_obs[scene.Name] != "" {
				image = image_path + buttons_obs[scene.Name]
			}

			oaction := &actionhandlers.OBSSceneAction{Scene: scene.Name, Client: obs_client}
			obutton, err := buttons.NewImageFileButton(image)
			if err == nil {
				obutton.SetActionHandler(oaction)
				sd.AddButton(i + offset, obutton)
			} else {
				log.Warn().Err(err)
				// use a text button
				oopbutton := buttons.NewTextButton(scene.Name)
				oopbutton.SetActionHandler(oaction)
				sd.AddButton(i + offset, oopbutton)
			}

			// only need a few scenes
			if i > 6 {
				break
			}
		}
	}

	// Command
	eyesbutton := buttons.NewTextButton("Eyes")
	eyesaction := &sdactionhandlers.CustomAction{}
	eyesaction.SetHandler(func(btn streamdeck.Button) {
		cmd := exec.Command("xeyes")
		cmd.Start()
	})
	eyesbutton.SetActionHandler(eyesaction)
	sd.AddButton(7, eyesbutton)

	/*
		// example of multiple actions
		thisActionHandler := &sdactionhandlers.ChainedAction{}
		thisActionHandler.AddAction(&sdactionhandlers.TextPrintAction{Label: "Purple press"})
		thisActionHandler.AddAction(&sdactionhandlers.ColourChangeAction{NewColour: color.RGBA{255, 0, 0, 255}})
		multiActionButton := buttons.NewColourButton(color.RGBA{255, 0, 255, 255})
		multiActionButton.SetActionHandler(thisActionHandler)
		sd.AddButton(0, multiActionButton)
	*/

}

func connectMQTT() mqtt.Client {
	log.Debug().Msg("Connecting to MQTT...")
	opts := mqtt.NewClientOptions().AddBroker("tcp://10.1.0.1:1883").SetClientID("go-streamdeck")
	mqtt_client = mqtt.NewClient(opts)
	if conn_token := mqtt_client.Connect(); conn_token.Wait() && conn_token.Error() != nil {
		log.Warn().Err(conn_token.Error()).Msg("Cannot connect to MQTT")
	}
	return mqtt_client
}

func connectOBS() obsws.Client {
	log.Debug().Msg("Connecting to OBS...")
	log.Info().Msgf("%#v\n", viper.Get("obs.host"))
	obs_client = obsws.Client{Host: "localhost", Port: 4444}
	err := obs_client.Connect()
	if err != nil {
		log.Warn().Err(err).Msg("Cannot connect to OBS")
	}
	return obs_client
}

/*
// MyButtonPress reacts to a button being pressed
func MyButtonPress(btnIndex int, sd *streamdeck.Device, err error) {
	switch btnIndex {
	case 0:
		sources, _ := pulse.Core().ListPath("Sources")

		for _, src := range sources {
			dev := pulse.Device(src) // Only use the first sink for the test.
			var name string
			var muted bool
			dev.Get("Name", &name)
			dev.Get("Mute", &muted)
			fmt.Println(src, muted, name)

			dev.Set("Mute", true)
		}
	}
}
*/

type AppPulse struct {
	Client *pulseaudio.Client
}

func getPulseConnection() *pulseaudio.Client {
	isLoaded, e := pulseaudio.ModuleIsLoaded()
	testFatal(e, "test pulse dbus module is loaded")
	if !isLoaded {
		e = pulseaudio.LoadModule()
		testFatal(e, "load pulse dbus module")
	}

	// Connect to the pulseaudio dbus service.
	pulse, e := pulseaudio.New()
	testFatal(e, "connect to the pulse service")
	return pulse
}

func closePulseConnection(pulse *pulseaudio.Client) {
	//defer pulseaudio.UnloadModule()
	defer pulse.Close()
}

func testFatal(e error, msg string) {
	if e != nil {
		log.Warn().Err(e).Msg(msg)
	}
}

// Wemo functions from magicmonkey modified library
func startWemoScan() {
	err := belkin.ScanWithCallback(belkin.DTInsight, 10, gotWemoDevice)
	fmt.Println(err)
}

func gotWemoDevice(device belkin.Device) {
	device.Load(1 * time.Second)
	state, err := device.FetchBinaryState(1 * time.Second)
	if err != nil {
		log.Warn().Err(err)
	}
	log.Info().Msg("Found device " + device.FriendlyName)
	log.Debug().Msg("Current device state: " + strconv.Itoa(state)) // 0, 1 or 8 (for standby)

	for i, name := range buttons_wemo {
		if name == device.FriendlyName {
			// colour reflects state: green for on, red for off
			colour := color.RGBA{255, 0, 50, 255}
			if state == 1 {
				colour = color.RGBA{20, 255, 50, 255}
			}
			wemobutton := buttons.NewTextButtonWithColours(name, colour, color.RGBA{0, 0, 0, 255})
			wemoaction := &actionhandlers.WemoAction{Device: device, State: device.BinaryState}
			wemobutton.SetActionHandler(wemoaction)
			sd.AddButton(i, wemobutton)
		}
	}

}
