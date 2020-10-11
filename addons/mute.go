package addons

import (
	"bytes"
	"image/color"
	"os/exec"

	"github.com/magicmonkey/go-streamdeck"
	sdactionhandlers "github.com/magicmonkey/go-streamdeck/actionhandlers"
	"github.com/magicmonkey/go-streamdeck/buttons"
	sddecorators "github.com/magicmonkey/go-streamdeck/decorators"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

type Mute struct {
	SD        *streamdeck.StreamDeck
	Status    bool // true if muted
	Button_id int
}

func (s *Mute) Init() {
	// not much to initialise but should probably read some config for source name
	// or calculate it, try this (yes, really)
	// pulsemixer --list-sources | cut -f3 | grep 'UMC404HD 192k Multichannel' | cut -d ',' -f 1 | cut -c 6-

}

func (s *Mute) Buttons() {
	// Command
	shotbutton, _ := buttons.NewImageFileButton(viper.GetString("buttons.images") + "/mic.png")
	shotaction := &sdactionhandlers.CustomAction{}
	shotaction.SetHandler(func(btn streamdeck.Button) {
		go s.toggleMute()
	})
	shotbutton.SetActionHandler(shotaction)
	s.SD.AddButton(s.Button_id, shotbutton)
	s.updateButtonDecoration()
}

func (s *Mute) toggleMute() {
	if s.Status {
		// unmute
		log.Debug().Msg("Unmuting")
		cmd := exec.Command("/usr/bin/pulsemixer", "--id", "source-7", "--unmute")
		if err := cmd.Run(); err != nil {
			log.Warn().Err(err)
		}
	} else {
		log.Debug().Msg("Muting")
		cmd := exec.Command("/usr/bin/pulsemixer", "--id", "source-7", "--mute")
		if err := cmd.Run(); err != nil {
			log.Warn().Err(err)
		}
	}
	s.updateButtonDecoration()
}

func (s *Mute) readMuteStatus() bool {
	cmd := exec.Command("/usr/bin/pulsemixer", "--id", "source-7", "--get-mute")
	var outb bytes.Buffer
	cmd.Stdout = &outb
	if err := cmd.Run(); err != nil {
		log.Warn().Err(err)
	} else {
		// there's a newline in the stdout output!
		if outb.String() == "0\n" {
			log.Info().Msg("Mic is LIVE")
			s.Status = false
		} else {
			log.Info().Msg("Mic is muted")
			s.Status = true
		}
	}
	return s.Status

}

func (s *Mute) updateButtonDecoration() {
	status := s.readMuteStatus()
	decorate_on := sddecorators.NewBorder(12, color.RGBA{255, 120, 150, 255})
	decorate_off := sddecorators.NewBorder(12, color.RGBA{120, 120, 120, 255})
	if status {
		s.SD.SetDecorator(s.Button_id, decorate_off)
	} else {
		s.SD.SetDecorator(s.Button_id, decorate_on)
	}
}
