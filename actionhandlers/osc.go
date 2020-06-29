package actionhandlers

import (
	"fmt"
	"strconv"

	"github.com/hypebeast/go-osc/osc"
	streamdeck "github.com/magicmonkey/go-streamdeck"
)

type OSCAction struct {
	Track int
	btn   streamdeck.Button
}

func (action *OSCAction) Pressed(btn streamdeck.Button) {
	client := osc.NewClient("127.0.0.1", 5051)
	msg := osc.NewMessage("/castersoundboard/board/Twitch/player/" + strconv.Itoa(action.Track) + "/modify/play_state/play")
	msg.Append(int32(1))
	err := client.Send(msg)
	fmt.Printf("%#v\n", err)
}
