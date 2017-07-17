package main

import (
	"github.com/g3n/engine/audio"
	"github.com/g3n/engine/audio/al"
	"github.com/g3n/engine/audio/ov"
	"github.com/g3n/engine/audio/vorbis"
	"github.com/g3n/engine/camera"
	"github.com/g3n/engine/camera/control"
	"github.com/g3n/engine/core"
	"github.com/g3n/engine/gls"
	"github.com/g3n/engine/graphic"
	"github.com/g3n/engine/gui"
	"github.com/g3n/engine/light"
	"github.com/g3n/engine/loader/obj"
	"github.com/g3n/engine/material"
	"github.com/g3n/engine/math32"
	"github.com/g3n/engine/renderer"
	"github.com/g3n/engine/util/logger"
	"github.com/g3n/engine/window"

	"fmt"
	"runtime"
	"strconv"
	"time"
	"io/ioutil"
)

//      ____       _         _
//     / ___| ___ | | _____ | |__   __ _ _ __
//    | |  _ / _ \| |/ / _ \| '_ \ / _` | '_ \
//    | |_| | (_) |   < (_) | |_) | (_| | | | |
//     \____|\___/|_|\_\___/|_.__/ \__,_|_| |_|
//

const CREDITS_LINE1 string = "Open source game by Daniel Salvadori (github.com/danaugrs). Written in Go and powered by g3n (github.com/g3n/engine)."
const CREDITS_LINE2 string = "Music by Eric Matyas (www.soundimage.org)."

var log *logger.Logger

type GokobanGame struct {
	win      		window.IWindow
	gs       		*gls.GLS
	renderer 		*renderer.Renderer
	scene    		*core.Node
	camera       	*camera.Perspective
	orbitControl 	*control.OrbitControl

	userData *UserData

	root     *gui.Root
	menu     *gui.Panel
	main     *gui.Panel
	controls *gui.Panel

	musicCheckbox 	*gui.CheckRadio
	musicSlider 	*gui.Slider

	sfxCheckbox 	*gui.CheckRadio
	sfxSlider 		*gui.Slider

	loadingLabel  		*gui.ImageLabel
	levelLabel    		*gui.ImageButton
	titleImage   	 	*gui.ImageButton
	nextButton    		*gui.ImageButton
	prevButton    		*gui.ImageButton
	restartButton 		*gui.ImageButton
	menuButton 		    *gui.ImageButton
	quitButton 		    *gui.ImageButton
	playButton 		    *gui.ImageButton
	sfxButton 		    *gui.ImageButton
	musicButton 		*gui.ImageButton
	fullScreenButton 	*gui.ImageButton

	levelScene *core.Node
	levelStyle *LevelStyle
	levels     []*Level
	levelsRaw  []string
	level      *Level
	leveln     int

	gopherLocked bool
	gopherNode   *core.Node
	steps        int
	audioAvailable bool

	// Sound/music players
	musicPlayer 		  *audio.Player
	musicPlayerMenu 	  *audio.Player
	clickPlayer           *audio.Player
	hoverPlayer           *audio.Player
	walkPlayer            *audio.Player
	bumpPlayer            *audio.Player
	gopherHurtPlayer      *audio.Player
	gopherFallEndPlayer   *audio.Player
	gopherFallStartPlayer *audio.Player
	boxPushPlayer         *audio.Player
	boxOnPadPlayer        *audio.Player
	boxOffPadPlayer       *audio.Player
	boxFallEndPlayer      *audio.Player
	boxFallStartPlayer    *audio.Player
	elevatorUpPlayer      *audio.Player
	elevatorDownPlayer    *audio.Player
	levelDonePlayer       *audio.Player
	levelRestartPlayer    *audio.Player
	levelFailPlayer       *audio.Player
	gameCompletePlayer    *audio.Player
}

// RestartLevel restarts the current level
func (g *GokobanGame) RestartLevel() {
	log.Debug("Restart Level")

	g.levels[g.leveln].Restart(true)
}

// NextLevel loads the next level if exists
func (g *GokobanGame) NextLevel() {
	log.Debug("Next Level")

	if g.leveln < len(g.levels) {
		g.InitLevel(g.leveln + 1)
	}
}

// PreviousLevel loads the previous level if exists
func (g *GokobanGame) PreviousLevel() {
	log.Debug("Previous Level")

	if g.leveln > 0 {
		g.InitLevel(g.leveln - 1)
	}
}

// ToggleFullScreen toggles whether is game is fullscreen or windowed
func (g *GokobanGame) ToggleFullScreen() {
	log.Debug("Toggle FullScreen")

	g.win.SetFullScreen(!g.win.FullScreen())
}

// ToggleMenu switched the menu, title, and credits overlay for the in-level corner buttons
func (g *GokobanGame) ToggleMenu() {
	log.Debug("Toggle Menu")

	if g.audioAvailable {
		if g.menu.Visible() {
			g.musicPlayerMenu.Stop()
			g.musicPlayer.Play()
		} else {
			g.musicPlayer.Pause()
			g.musicPlayerMenu.Play()
		}
	}

	g.gopherLocked = !g.gopherLocked
	g.menu.SetVisible(!g.menu.Visible())
	g.controls.SetVisible(!g.controls.Visible())
	g.orbitControl.Enabled = !g.orbitControl.Enabled
}

// Quit saves the user data and quits the game
func (g *GokobanGame) Quit() {
	log.Debug("Quit")

	// Copy settings into user data and save
	g.userData.SfxVol = g.sfxSlider.Value()
	g.userData.MusicVol = g.musicSlider.Value()
	g.userData.FullScreen = g.win.FullScreen()
	g.userData.Save()

	// Close the window
	g.win.SetShouldClose(true)
}

// onKey handles keyboard events for the game
func (g *GokobanGame) onKey(evname string, ev interface{}) {

	kev := ev.(*window.KeyEvent)
	switch kev.Keycode {
	case window.KeyEscape:
		g.ToggleMenu()
	case window.KeyF:
		g.ToggleFullScreen()
	case window.KeyR:
		g.RestartLevel()
	case window.KeyRightBracket:
		g.NextLevel()
	case window.KeyLeftBracket:
		g.PreviousLevel()
	}
}

// Update updates the current level if any
func (g *GokobanGame) Update(timeDelta float64) {
	if g.level != nil {
		g.level.Update(timeDelta)
	}
}

// LevelComplete updates and saves user data, enables the next button if appropriate, and checks for game completion
func (g *GokobanGame) LevelComplete() {
	log.Debug("Level Complete")

	if g.userData.LastUnlockedLevel == g.leveln {
		g.userData.LastUnlockedLevel++
		g.userData.Save() // Save in case game crashes
		if g.userData.LastUnlockedLevel < len(g.levels) {
			g.nextButton.SetEnabled(true)
		}
		if g.userData.LastUnlockedLevel == len(g.levels) {
			g.GameCompleted()
		}
	}
}

// GameCompleted stops the music, plays the the winning sound, and changes the title image to say "Completed"
func (g *GokobanGame) GameCompleted() {
	log.Debug("Game Completed")

	if g.audioAvailable {
		g.musicPlayer.Stop()
		g.PlaySound(g.gameCompletePlayer, nil)
	}
		g.titleImage.SetImage(gui.ButtonDisabled, "gui/title3_completed.png")
}

// InitLevel initializes the level associated to the provided index
func (g *GokobanGame) InitLevel(n int) {
	log.Debug("Initializing Level %v", n + 1)

	// Always enable the button to return to the previous level except when we are in the very first level
	g.prevButton.SetEnabled(n != 0)

	// The button to go to the next level has 3 different states: disabled, locked and enabled
	// If this is the very last level - disable it completely
	if n == len(g.levels) - 1 {
		g.nextButton.SetImage(gui.ButtonDisabled, "gui/right_disabled2.png")
		g.nextButton.SetEnabled(false)
	} else {
		g.nextButton.SetImage(gui.ButtonDisabled, "gui/right_disabled_locked.png")
		// check last completed level
		if g.userData.LastUnlockedLevel == n {
			g.nextButton.SetEnabled(false)
		} else {
			g.nextButton.SetEnabled(true)
		}
	}

	// Remove level.scene from levelScene and unsubscribe from events
	if len(g.levelScene.Children()) > 0 {
		g.levelScene.Remove(g.level.scene)
		g.win.UnsubscribeID(window.OnKeyDown, g.leveln)
	}

	// Update current level index and level reference
	g.leveln = n
	g.userData.LastLevel = n
	g.level = g.levels[g.leveln]

	g.level.Restart(false)
	g.level.gopherNode.Add(g.gopherNode)
	g.levelLabel.SetText("Level " + strconv.Itoa(n+1))
	g.levelScene.Add(g.level.scene)
	g.win.SubscribeID(window.OnKeyDown, g.leveln, g.level.onKey)

}

// LoadLevels reads and parses the level files inside ./levels, building an array of Level objects
func (g *GokobanGame) LoadLevels() {
	log.Debug("Load Levels")

	files, _ := ioutil.ReadDir("./levels")
	g.levels = make([]*Level, len(files) - 1)

	for i, f := range files {

		// Skip README.txt
		if f.Name() == "README.txt" {
			continue
		}

		log.Debug("Reading level file: %v as level %v", f.Name(), i+1)

		// Read level text file
		b, err := ioutil.ReadFile("./levels/" + f.Name())
		if err != nil {
			fmt.Print(err)
		}
		str := string(b) // convert content to a 'string'

		log.Debug("Parsing level " + strconv.Itoa(i+1))

		// Parse level data
		ld, errParse := ParseLevel(str)
		if errParse != nil {
			panic(errParse)
		}

		log.Debug("Building level " + strconv.Itoa(i+1))

		// Build level
		g.levels[i] = NewLevel(g, ld, g.levelStyle, g.camera)
	}
}

// SetSfxVolume sets the volume of all sound effects
func (g *GokobanGame) SetSfxVolume(vol float32) {
	log.Debug("Set Sfx Volume %v", vol)

	if g.audioAvailable {
		g.clickPlayer.SetGain(vol)
		g.hoverPlayer.SetGain(vol)
		g.walkPlayer.SetGain(vol)
		g.bumpPlayer.SetGain(vol)
		g.gopherHurtPlayer.SetGain(vol)
		g.gopherFallEndPlayer.SetGain(vol)
		g.gopherFallStartPlayer.SetGain(vol)
		g.boxPushPlayer.SetGain(vol)
		g.boxOnPadPlayer.SetGain(vol)
		g.boxOffPadPlayer.SetGain(vol)
		g.boxFallEndPlayer.SetGain(vol)
		g.boxFallStartPlayer.SetGain(vol)
		g.elevatorUpPlayer.SetGain(vol)
		g.elevatorDownPlayer.SetGain(vol)
		g.levelDonePlayer.SetGain(vol)
		g.levelRestartPlayer.SetGain(vol)
		g.levelFailPlayer.SetGain(vol)
	}
}

// SetMusicVolume sets the volume of the music
func (g *GokobanGame) SetMusicVolume(vol float32) {
	log.Debug("Set Music Volume %v", vol)

	if g.audioAvailable {
		g.musicPlayer.SetGain(vol)
		g.musicPlayerMenu.SetGain(vol)
	}
}

// LoadAudio loads music and sound effects
func (g *GokobanGame) LoadAudio() {
	log.Debug("Load Audio")

	// Create listener and add it to the current camera
	listener := audio.NewListener()
	cdir := g.camera.Direction()
	listener.SetDirectionv(&cdir)
	g.camera.GetCamera().Add(listener)

	// Helper function to create player and handle errors
	createPlayer := func(fname string) *audio.Player {
		log.Debug("Loading " + fname)
		p, err := audio.NewPlayer(fname)
		if err != nil {
			log.Error("Failed to create player for: %v", fname)
		}
		return p
	}

	g.musicPlayer = createPlayer("audio/music/Lost-Jungle_Looping.ogg")
	g.musicPlayer.SetLooping(true)

	g.musicPlayerMenu = createPlayer("audio/music/Spooky-Island.ogg")
	g.musicPlayerMenu.SetLooping(true)

	rFactor := float32(0.2)

	g.clickPlayer = createPlayer("audio/sfx/button_click.ogg")
	g.clickPlayer.SetRolloffFactor(rFactor)

	g.hoverPlayer = createPlayer("audio/sfx/button_hover.ogg")
	g.hoverPlayer.SetRolloffFactor(rFactor)

	g.walkPlayer = createPlayer("audio/sfx/gopher_walk.ogg")
	g.walkPlayer.SetRolloffFactor(rFactor)

	g.bumpPlayer = createPlayer("audio/sfx/gopher_bump.ogg")
	g.bumpPlayer.SetRolloffFactor(rFactor)

	g.gopherFallStartPlayer = createPlayer("audio/sfx/gopher_fall_start.ogg")
	g.gopherFallStartPlayer.SetRolloffFactor(rFactor)

	g.gopherFallEndPlayer = createPlayer("audio/sfx/gopher_fall_end.ogg")
	g.gopherFallEndPlayer.SetRolloffFactor(rFactor)

	g.gopherHurtPlayer = createPlayer("audio/sfx/gopher_hurt.ogg")
	g.gopherHurtPlayer.SetRolloffFactor(rFactor)

	g.boxPushPlayer = createPlayer("audio/sfx/box_push.ogg")
	g.boxPushPlayer.SetRolloffFactor(rFactor)

	g.boxOnPadPlayer = createPlayer("audio/sfx/box_on.ogg")
	g.boxOnPadPlayer.SetRolloffFactor(rFactor)

	g.boxOffPadPlayer = createPlayer("audio/sfx/box_off.ogg")
	g.boxOffPadPlayer.SetRolloffFactor(rFactor)

	g.boxFallStartPlayer = createPlayer("audio/sfx/box_fall_start.ogg")
	g.boxFallStartPlayer.SetRolloffFactor(rFactor)

	g.boxFallEndPlayer = createPlayer("audio/sfx/box_fall_end.ogg")
	g.boxFallEndPlayer.SetRolloffFactor(rFactor)

	g.elevatorUpPlayer = createPlayer("audio/sfx/elevator_up.ogg")
	g.elevatorUpPlayer.SetLooping(true)
	g.elevatorUpPlayer.SetRolloffFactor(rFactor)

	g.elevatorDownPlayer = createPlayer("audio/sfx/elevator_down.ogg")
	g.elevatorDownPlayer.SetLooping(true)
	g.elevatorDownPlayer.SetRolloffFactor(rFactor)

	g.levelDonePlayer = createPlayer("audio/sfx/level_done.ogg")
	g.levelDonePlayer.SetRolloffFactor(rFactor)

	g.levelRestartPlayer = createPlayer("audio/sfx/level_restart.ogg")
	g.levelRestartPlayer.SetRolloffFactor(rFactor)

	g.levelFailPlayer = createPlayer("audio/sfx/level_fail.ogg")
	g.levelFailPlayer.SetRolloffFactor(rFactor)

	g.gameCompletePlayer = createPlayer("audio/sfx/game_complete.ogg")
	g.gameCompletePlayer.SetRolloffFactor(rFactor)
}

// LoadSkybox loads the space skybox and adds it to the scene
func (g *GokobanGame) LoadSkyBox() {
	log.Debug("Creating Skybox...")

	// Load skybox textures
	skyboxData := graphic.SkyboxData{
		"img/skybox/", "jpg",
		[6]string{"px", "nx", "py", "ny", "pz", "nz"}}

	skybox, err := graphic.NewSkybox(skyboxData)
	if err != nil {
		panic(err)
	}

	// For each skybox face - set the material to not use lights and to have emissive color.
	brightness := float32(0.6)
	sbmats := skybox.Materials()
	for i := 0; i < len(sbmats); i++ {
		sbmat := skybox.Materials()[i].GetMaterial().(*material.Standard)
		sbmat.SetUseLights(material.UseLightNone)
		sbmat.SetEmissiveColor(&math32.Color{brightness, brightness, brightness})
	}
	g.scene.Add(skybox)

	log.Debug("Done creating skybox")
}

// LoadGopher loads the gopher model and adds to it the sound players associated to it
func (g *GokobanGame) LoadGopher() {
	log.Debug("Decoding gopher model...")

	// Decode model in OBJ format
	dec, err := obj.Decode("gopher/gopher.obj", "gopher/gopher.mtl")
	// dec, err := obj.Decode("data/gopher_low_poly2.obj", "data/gopher_low_poly2.mtl")
	if err != nil {
		panic(err.Error())
	}

	// Create a new node with all the objects in the decoded file and adds it to the scene
	g.gopherNode, err = dec.NewGroup()
	if err != nil {
		panic(err.Error())
	}

	log.Debug("Done decoding gopher model")

	// Add gopher-related sound players to gopher node for correct 3D sound positioning
	if g.audioAvailable {
		g.gopherNode.Add(g.walkPlayer)
		g.gopherNode.Add(g.bumpPlayer)
		g.gopherNode.Add(g.gopherFallStartPlayer)
		g.gopherNode.Add(g.gopherFallEndPlayer)
		g.gopherNode.Add(g.gopherHurtPlayer)
	}
}



func (g *GokobanGame) UpdateMusicButton() {
	if g.userData.MusicOn {
		g.musicButton.SetImage(gui.ButtonNormal, "gui/music_normal.png")
		g.musicButton.SetImage(gui.ButtonOver, "gui/music_hover.png")
		g.musicButton.SetImage(gui.ButtonPressed, "gui/music_click.png")
		g.musicSlider.SetEnabled(true)
		g.musicSlider.SetValue(g.musicSlider.Value())
	} else {
		g.musicButton.SetImage(gui.ButtonNormal, "gui/music_normal_off.png")
		g.musicButton.SetImage(gui.ButtonOver, "gui/music_hover_off.png")
		g.musicButton.SetImage(gui.ButtonPressed, "gui/music_click_off.png")
		g.musicSlider.SetEnabled(false)
		g.SetMusicVolume(0)
	}
}

func (g *GokobanGame) UpdateSfxButton() {
	if g.userData.SfxOn {
		g.sfxButton.SetImage(gui.ButtonNormal, "gui/sound_normal.png")
		g.sfxButton.SetImage(gui.ButtonOver, "gui/sound_hover.png")
		g.sfxButton.SetImage(gui.ButtonPressed, "gui/sound_click.png")
		g.sfxSlider.SetEnabled(true)
		g.sfxSlider.SetValue(g.sfxSlider.Value())
	} else {
		g.sfxButton.SetImage(gui.ButtonNormal, "gui/sound_normal_off.png")
		g.sfxButton.SetImage(gui.ButtonOver, "gui/sound_hover_off.png")
		g.sfxButton.SetImage(gui.ButtonPressed, "gui/sound_click_off.png")
		g.sfxSlider.SetEnabled(false)
		g.SetSfxVolume(0)
	}
}



func (g *GokobanGame) SetupGui(width, height int) {
	log.Debug("Creating GUI...")

	transparent := math32.Color4{0, 0, 0, 0}
	blackTextColor := math32.Color{0.3,0.3,0.3}
	creditsColor := math32.Color{0.6, 0.6, 0.6}
	sliderColor := math32.Color4{0.628, 0.882, 0.1, 1}
	sliderColorOff := math32.Color4{0.82, 0.48, 0.48, 1}
	sliderColorOver := math32.Color4{0.728, 0.982, 0.2, 1}
	sliderBorderColor := math32.Color4{0.71, 0.482, 0.26, 1}

	sliderBorder := gui.BorderSizes{3, 3, 3, 3}
	zeroBorder := gui.BorderSizes{0, 0, 0, 0}

	gui.StyleDefault.ImageButton = gui.ImageButtonStyles{
		Normal: gui.ImageButtonStyle{
			Border:      zeroBorder,
			Paddings:    zeroBorder,
			BorderColor: transparent,
			BgColor:     transparent,
			FgColor:     blackTextColor,
		},
		Over: gui.ImageButtonStyle{
			Border:      zeroBorder,
			Paddings:    zeroBorder,
			BorderColor: transparent,
			BgColor:     transparent,
			FgColor:     blackTextColor,
		},
		Focus: gui.ImageButtonStyle{
			Border:      zeroBorder,
			Paddings:    zeroBorder,
			BorderColor: transparent,
			BgColor:     transparent,
			FgColor:     blackTextColor,
		},
		Pressed: gui.ImageButtonStyle{
			Border:      zeroBorder,
			Paddings:    zeroBorder,
			BorderColor: transparent,
			BgColor:     transparent,
			FgColor:     blackTextColor,
		},
		Disabled: gui.ImageButtonStyle{
			Border:      zeroBorder,
			Paddings:    zeroBorder,
			BorderColor: transparent,
			BgColor:     transparent,
			FgColor:     blackTextColor,
		},
	}

	gui.StyleDefault.Slider = gui.SliderStyles{
		Normal: gui.SliderStyle{
			Border:      sliderBorder,
			BorderColor: sliderBorderColor,
			Paddings:    gui.BorderSizes{0, 0, 0, 0},
			BgColor:     math32.Color4{0.2, 0.2, 0.2, 1},
			FgColor:     sliderColor,
		},
		Over: gui.SliderStyle{
			Border:      sliderBorder,
			BorderColor: sliderBorderColor,
			Paddings:    gui.BorderSizes{0, 0, 0, 0},
			BgColor:     math32.Color4{0.3, 0.3, 0.3, 1},
			FgColor:     sliderColorOver,
		},
		Focus: gui.SliderStyle{
			Border:      sliderBorder,
			BorderColor: sliderBorderColor,
			Paddings:    gui.BorderSizes{0, 0, 0, 0},
			BgColor:     math32.Color4{0.3, 0.3, 0.3, 1},
			FgColor:     sliderColorOver,
		},
		Disabled: gui.SliderStyle{
			Border:      sliderBorder,
			BorderColor: sliderBorderColor,
			Paddings:    gui.BorderSizes{0, 0, 0, 0},
			BgColor:     math32.Color4{0.2, 0.2, 0.2, 1},
			FgColor:     sliderColorOff,
		},
	}

	var err error

	hoverSound := func(evname string, ev interface{}) {
		g.PlaySound(g.hoverPlayer, nil)
	}

	// Menu
	g.menu = gui.NewPanel(100, 100)
	g.menu.SetColor4(math32.NewColor4(0.1, 0.1, 0.1, 0.6))
	g.root.Subscribe(gui.OnResize, func(evname string, ev interface{}) {
		g.menu.SetWidth(g.root.ContentWidth())
		g.menu.SetHeight(g.root.ContentHeight())
	})

	// Controls
	g.controls = gui.NewPanel(100, 100)
	g.root.Subscribe(gui.OnResize, func(evname string, ev interface{}) {
		g.controls.SetWidth(g.root.ContentWidth())
		g.controls.SetHeight(g.root.ContentHeight())
	})

	// Header panel
	header := gui.NewPanel(0, 0)
	header.SetPosition(0, 0)
	header.SetLayout(gui.NewHBoxLayout())
	header.SetPaddings(20, 20, 20, 20)
	header.SetSize(float32(width), 160)
	g.root.Subscribe(gui.OnResize, func(evname string, ev interface{}) {
		header.SetWidth(g.root.ContentWidth())
	})
	g.controls.Add(header)

	// Previous Level Button
	g.prevButton, err = gui.NewImageButton("gui/left_normal.png")
	g.prevButton.SetImage(gui.ButtonOver, "gui/left_hover.png")
	g.prevButton.SetImage(gui.ButtonPressed, "gui/left_click.png")
	g.prevButton.SetImage(gui.ButtonDisabled, "gui/left_disabled2.png")
	if err != nil {
		panic(err)
	}
	g.prevButton.Subscribe(gui.OnMouseUp, func(evname string, ev interface{}) {
		g.PlaySound(g.clickPlayer, nil)
		g.PreviousLevel()
	})
	g.prevButton.Subscribe(gui.OnCursorEnter, hoverSound)
	header.Add(g.prevButton)

	params := gui.HBoxLayoutParams{Expand: 1, AlignV: gui.AlignCenter}

	spacer1 := gui.NewPanel(0, 0)
	spacer1.SetLayoutParams(&params)
	header.Add(spacer1)

	// Level Number Label
	g.levelLabel, err = gui.NewImageButton("gui/panel.png")
	g.levelLabel.SetImage(gui.ButtonDisabled, "gui/panel.png")
	g.levelLabel.SetColor(math32.NewColor(0.8,0.8,0.8))
	g.levelLabel.SetText("TEST")
	g.levelLabel.SetFontSize(35)
	g.levelLabel.SetEnabled(false)
	header.Add(g.levelLabel)

	spacer2 := gui.NewPanel(0, 0)
	spacer2.SetLayoutParams(&params)
	header.Add(spacer2)

	// Next Level Button
	g.nextButton, err = gui.NewImageButton("gui/right_normal.png")
	g.nextButton.SetImage(gui.ButtonOver, "gui/right_hover.png")
	g.nextButton.SetImage(gui.ButtonPressed, "gui/right_click.png")
	g.nextButton.SetImage(gui.ButtonDisabled, "gui/right_disabled2.png")
	if err != nil {
		panic(err)
	}
	g.nextButton.Subscribe(gui.OnMouseUp, func(evname string, ev interface{}) {
		g.PlaySound(g.clickPlayer, nil)
		g.NextLevel()
	})
	g.nextButton.Subscribe(gui.OnCursorEnter, hoverSound)
	header.Add(g.nextButton)

	// Footer panel
	footer := gui.NewPanel(0, 0)
	footer_height := 140
	footer.SetLayout(gui.NewHBoxLayout())
	footer.SetPaddings(20, 20, 20, 20)
	footer.SetSize(g.root.ContentHeight(), float32(footer_height))
	g.root.Subscribe(gui.OnResize, func(evname string, ev interface{}) {
		footer.SetWidth(g.root.ContentWidth())
		footer.SetPositionY(g.root.ContentHeight() - float32(footer_height))
	})
	g.controls.Add(footer)

	// Restart Level Button
	g.restartButton, err = gui.NewImageButton("gui/restart_normal.png")
	g.restartButton.SetImage(gui.ButtonOver, "gui/restart_hover.png")
	g.restartButton.SetImage(gui.ButtonPressed, "gui/restart_click.png")
	g.restartButton.SetImage(gui.ButtonDisabled, "gui/restart_disabled2.png")
	if err != nil {
		panic(err)
	}
	g.restartButton.Subscribe(gui.OnMouseUp, func(evname string, ev interface{}) {
		g.RestartLevel()
	})
	g.restartButton.Subscribe(gui.OnCursorEnter, hoverSound)
	footer.Add(g.restartButton)

	spacer3 := gui.NewPanel(0, 0)
	spacer3.SetLayoutParams(&params)
	footer.Add(spacer3)

	// Restart Level Button
	g.menuButton, err = gui.NewImageButton("gui/menu_normal.png")
	g.menuButton.SetImage(gui.ButtonOver, "gui/menu_hover.png")
	g.menuButton.SetImage(gui.ButtonPressed, "gui/menu_click.png")
	g.menuButton.SetImage(gui.ButtonDisabled, "gui/menu_disabled2.png")
	if err != nil {
		panic(err)
	}
	g.menuButton.Subscribe(gui.OnMouseUp, func(evname string, ev interface{}) {
		g.PlaySound(g.clickPlayer, nil)
		g.ToggleMenu()
	})
	g.menuButton.Subscribe(gui.OnCursorEnter, hoverSound)
	footer.Add(g.menuButton)

	g.titleImage, err = gui.NewImageButton("gui/title3.png")
	g.titleImage.SetImage(gui.ButtonDisabled, "gui/title3.png")
	g.titleImage.SetEnabled(false)
	g.root.Subscribe(gui.OnResize, func(evname string, ev interface{}) {
		g.titleImage.SetPositionX((g.root.ContentWidth() - g.titleImage.ContentWidth()) / 2)
	})
	g.menu.Add(g.titleImage)

	// Loading Text
	g.loadingLabel = gui.NewImageLabel("Loading...")
	g.loadingLabel.SetColor(math32.NewColor(1, 1, 1))
	g.loadingLabel.SetFontSize(40)
	g.root.Subscribe(gui.OnResize, func(evname string, ev interface{}) {
		g.loadingLabel.SetPositionX((g.root.ContentWidth() - g.loadingLabel.ContentWidth()) / 2)
		g.loadingLabel.SetPositionY((g.root.ContentHeight() - g.loadingLabel.ContentHeight()) / 2)
	})
	g.root.Add(g.loadingLabel)

	g.controls.SetVisible(false)
	g.root.Add(g.controls)

	// Main panel
	g.main = gui.NewPanel(600, 300)
	mainLayout := gui.NewVBoxLayout()
	mainLayout.SetAlignV(gui.AlignHeight)
	g.main.SetLayout(mainLayout)
	g.main.SetBorders(2, 2, 2, 2)
	g.main.SetBordersColor4(&sliderBorderColor)
	g.main.SetColor4(math32.NewColor4(0.2, 0.2, 0.2, 0.6))
	g.root.Subscribe(gui.OnResize, func(evname string, ev interface{}) {
		g.main.SetPositionX((g.root.Width() - g.main.Width()) / 2)
		g.main.SetPositionY((g.root.Height() - g.main.Height()) / 2 + 50)
	})

	topRow := gui.NewPanel(g.main.ContentWidth(), 100)
	topRowLayout := gui.NewHBoxLayout()
	topRowLayout.SetAlignH(gui.AlignWidth)
	topRow.SetLayout(topRowLayout)

	// Music Control
	musicControl := gui.NewPanel(130, 100)
	musicControl.SetLayout(topRowLayout)

	g.musicButton, err = gui.NewImageButton("gui/music_normal.png")
	g.musicButton.SetImage(gui.ButtonOver, "gui/music_hover.png")
	g.musicButton.SetImage(gui.ButtonPressed, "gui/music_click.png")
	g.musicButton.SetImage(gui.ButtonDisabled, "gui/music_disabled2.png")
	if err != nil {
		panic(err)
	}
	g.musicButton.Subscribe(gui.OnMouseUp, func(evname string, ev interface{}) {
		g.PlaySound(g.clickPlayer, nil)
		g.userData.MusicOn = !g.userData.MusicOn
		g.UpdateMusicButton()
	})
	g.musicButton.Subscribe(gui.OnCursorEnter, hoverSound)
	musicControl.Add(g.musicButton)

	// Music Volume Slider
	g.musicSlider = gui.NewVSlider(20, 80)
	g.musicSlider.SetValue(0.5)
	g.musicSlider.Subscribe(gui.OnChange, func(evname string, ev interface{}) {
		g.SetMusicVolume(g.musicSlider.Value())
	})
	g.musicSlider.Subscribe(gui.OnCursorEnter, hoverSound)
	g.musicSlider.SetMargins(5, 0, 30, 4)
	musicControl.Add(g.musicSlider)

	topRow.Add(musicControl)

	// Sound Effects Control
	sfxControl := gui.NewPanel(130, 100)
	sfxControl.SetLayout(topRowLayout)

	g.sfxButton, err = gui.NewImageButton("gui/sound_normal.png")
	g.sfxButton.SetImage(gui.ButtonOver, "gui/sound_hover.png")
	g.sfxButton.SetImage(gui.ButtonPressed, "gui/sound_click.png")
	g.sfxButton.SetImage(gui.ButtonDisabled, "gui/sound_disabled2.png")
	if err != nil {
		panic(err)
	}
	g.sfxButton.Subscribe(gui.OnMouseUp, func(evname string, ev interface{}) {
		g.PlaySound(g.clickPlayer, nil)
		g.userData.SfxOn = !g.userData.SfxOn
		g.UpdateSfxButton()
	})
	g.sfxButton.Subscribe(gui.OnCursorEnter, hoverSound)
	sfxControl.Add(g.sfxButton)

	// Sound Effects Volume Slider
	g.sfxSlider = gui.NewVSlider(20, 80)
	g.sfxSlider.SetValue(0.5)
	g.sfxSlider.Subscribe(gui.OnChange, func(evname string, ev interface{}) {
		g.SetSfxVolume(4*g.sfxSlider.Value())
	})
	g.sfxSlider.Subscribe(gui.OnCursorEnter, hoverSound)
	g.sfxSlider.SetMargins(5, 0, 30, 4)
	sfxControl.Add(g.sfxSlider)

	topRow.Add(sfxControl)

	// FullScreen Button
	g.fullScreenButton, err = gui.NewImageButton("gui/screen_normal.png")
	g.fullScreenButton.SetImage(gui.ButtonOver, "gui/screen_hover.png")
	g.fullScreenButton.SetImage(gui.ButtonPressed, "gui/screen_click.png")
	g.fullScreenButton.SetImage(gui.ButtonDisabled, "gui/screen_disabled2.png")
	if err != nil {
		panic(err)
	}
	g.fullScreenButton.Subscribe(gui.OnMouseUp, func(evname string, ev interface{}) {
		g.PlaySound(g.clickPlayer, nil)
		g.ToggleFullScreen()
	})
	g.fullScreenButton.Subscribe(gui.OnCursorEnter, hoverSound)
	topRow.Add(g.fullScreenButton)

	g.main.Add(topRow)

	buttonRow := gui.NewPanel(g.main.ContentWidth(), 100)
	buttonRowLayout := gui.NewHBoxLayout()
	buttonRowLayout.SetAlignH(gui.AlignWidth)
	buttonRow.SetLayout(buttonRowLayout)

	// Quit Button
	g.quitButton, err = gui.NewImageButton("gui/quit_normal.png")
	g.quitButton.SetImage(gui.ButtonOver, "gui/quit_hover.png")
	g.quitButton.SetImage(gui.ButtonPressed, "gui/quit_click.png")
	g.quitButton.SetImage(gui.ButtonDisabled, "gui/quit_disabled2.png")
	if err != nil {
		panic(err)
	}
	g.quitButton.Subscribe(gui.OnMouseUp, func(evname string, ev interface{}) {
		g.Quit()
	})
	g.quitButton.Subscribe(gui.OnCursorEnter, hoverSound)
	buttonRow.Add(g.quitButton)

	// Play Button
	g.playButton, err = gui.NewImageButton("gui/play_normal.png")
	g.playButton.SetImage(gui.ButtonOver, "gui/play_hover.png")
	g.playButton.SetImage(gui.ButtonPressed, "gui/play_click.png")
	g.playButton.SetImage(gui.ButtonDisabled, "gui/play_disabled2.png")
	if err != nil {
		panic(err)
	}
	g.playButton.Subscribe(gui.OnMouseUp, func(evname string, ev interface{}) {
		g.PlaySound(g.clickPlayer, nil)
		g.ToggleMenu()
	})
	g.playButton.Subscribe(gui.OnCursorEnter, hoverSound)
	buttonRow.Add(g.playButton)

	g.main.Add(buttonRow)

	// Add credits labels
	lCredits1 := gui.NewImageLabel(CREDITS_LINE1)
	lCredits1.SetColor(&creditsColor)
	lCredits1.SetFontSize(20)
	g.root.Subscribe(gui.OnResize, func(evname string, ev interface{}) {
		lCredits1.SetWidth(g.root.ContentWidth())
		lCredits1.SetPositionY(g.root.ContentHeight() - 2*lCredits1.ContentHeight())
	})
	g.menu.Add(lCredits1)

	lCredits2 := gui.NewImageLabel(CREDITS_LINE2)
	lCredits2.SetColor(&creditsColor)
	lCredits2.SetFontSize(20)
	g.root.Subscribe(gui.OnResize, func(evname string, ev interface{}) {
		lCredits2.SetWidth(g.root.ContentWidth())
		lCredits2.SetPositionY(g.root.ContentHeight() - lCredits2.ContentHeight())
	})
	g.menu.Add(lCredits2)

	g3n := gui.NewImageLabel("")
	g3n.SetSize(57, 50)
	g3n.SetImageFromFile("img/g3n.png")
	g.root.Subscribe(gui.OnResize, func(evname string, ev interface{}) {
		g3n.SetPositionX(g.root.ContentWidth() - g3n.Width())
		g3n.SetPositionY(g.root.ContentHeight() - 1.3 * g3n.Height())
	})
	g.menu.Add(g3n)

	g.root.Add(g.menu)

	// Dispatch a fake OnResize event to update all subscribed elements
	g.root.Dispatch(gui.OnResize, nil)

	log.Debug("Done creating GUI.")
}

// PlaySound attaches the specified player to the specified node and plays the sound
func (g *GokobanGame) PlaySound(player *audio.Player, node *core.Node) {
	if g.audioAvailable {
		if node != nil {
			node.Add(player)
		}
		player.Stop()
		player.Play()
	}
}

// loadAudioLibs
func loadAudioLibs() error {

	// Try to load OpenAL
	err := al.Load()
	if err != nil {
		return err
	}

	// Open default audio device
	dev, err := al.OpenDevice("")
	if dev == nil {
		return fmt.Errorf("Error: %s opening OpenAL default device", err)
	}

	// Create audio context
	acx, err := al.CreateContext(dev, nil)
	if err != nil {
		return fmt.Errorf("Error creating audio context:%s", err)
	}

	// Make the context the current one
	err = al.MakeContextCurrent(acx)
	if err != nil {
		return fmt.Errorf("Error setting audio context current:%s", err)
	}
	log.Debug("%s version: %s", al.GetString(al.Vendor), al.GetString(al.Version))

	// Try to load Ogg Vorbis support
	err = ov.Load()
	if err != nil {
		return err
	}
	err = vorbis.Load()
	if err != nil {
		return err
	}
	log.Debug("%s", vorbis.VersionString())
	return nil
}

func main() {
	// OpenGL functions must be executed in the same thread where
	// the context was created (by window.New())
	runtime.LockOSThread()

	// Create logger
	log = logger.New("Gokoban", nil)
	log.AddWriter(logger.NewConsole(false))
	log.SetFormat(logger.FTIME | logger.FMICROS)
	log.SetLevel(logger.DEBUG)
	log.Info("Initializing Gokoban")

	// Create GokobanGame struct
	g := new(GokobanGame)

	// Load user data from file
	g.userData = NewUserData()

	// Create window and OpenGL context
	var err error
	g.win, err = window.New("glfw", 1200, 900, "Gokoban", g.userData.FullScreen)
	if err != nil {
		panic(err)
	}

	// Create OpenGL state
	g.gs, err = gls.New()
	if err != nil {
		panic(err)
	}

	// Speed up a bit by not checking OpenGL errors
	g.gs.SetCheckErrors(false)

	// Sets window background color
	g.gs.ClearColor(0.1, 0.1, 0.1, 1.0)

	// Sets the OpenGL viewport size the same as the window size
	// This normally should be updated if the window is resized.
	width, height := g.win.GetSize()
	g.gs.Viewport(0, 0, int32(width), int32(height))

	// Creates GUI root panel
	g.root = gui.NewRoot(g.gs, g.win)
	g.root.SetSize(float32(width), float32(height))

	// Subscribe to window resize events. When the window is resized:
	// - Update the viewport size
	// - Update the root panel size
	// - Update the camera aspect ratio
	g.win.Subscribe(window.OnWindowSize, func(evname string, ev interface{}) {
		width, height := g.win.GetSize()
		g.gs.Viewport(0, 0, int32(width), int32(height))
		g.root.SetSize(float32(width), float32(height))
		aspect := float32(width) / float32(height)
		g.camera.SetAspect(aspect)
	})

	// Subscribe window to events
	g.win.Subscribe(window.OnKeyDown, g.onKey)

	// Creates a renderer and adds default shaders
	g.renderer = renderer.NewRenderer(g.gs)
	err = g.renderer.AddDefaultShaders()
	if err != nil {
		panic(err)
	}

	// Adds a perspective camera to the scene
	// The camera aspect ratio should be updated if the window is resized.
	aspect := float32(width) / float32(height)
	g.camera = camera.NewPerspective(65, aspect, 0.01, 1000)
	g.camera.GetCamera().SetPosition(0, 4, 5)

	// Create orbit control and set limits
	g.orbitControl = control.NewOrbitControl(g.camera, g.win)
	g.orbitControl.Enabled = false
	g.orbitControl.EnablePan = false
	g.orbitControl.MaxPolarAngle = 2 * math32.Pi / 3
	g.orbitControl.MinDistance = 5
	g.orbitControl.MaxDistance = 15

	// Create main scene and child levelScene
	g.scene = core.NewNode()
	g.levelScene = core.NewNode()
	g.scene.Add(g.camera)
	g.scene.Add(g.levelScene)
	g.gopherLocked = true

	// Add white ambient light to the scene
	ambLight := light.NewAmbient(&math32.Color{1.0, 1.0, 1.0}, 0.4)
	g.scene.Add(ambLight)

	//dirLight := light.NewDirectional(&math32.Color{1.0, 1.0, 1.0}, 0.1)
	//dirLight.SetPosition(-10, 10, 10)
	//g.scene.Add(dirLight)

	g.levelStyle = NewStandardStyle()

	g.SetupGui(width, height)
	g.RenderFrame()

	// Try to open audio libraries
	err = loadAudioLibs()
	if err != nil {
		log.Error("%s", err)
	} else {
		g.audioAvailable = true
		g.LoadAudio()
		// Queue the music!
		g.musicPlayerMenu.Play()
	}

	g.LoadSkyBox()
	g.LoadGopher()
	g.LoadLevels()

	if g.userData.LastUnlockedLevel == len(g.levels) {
		g.titleImage.SetImage(gui.ButtonDisabled, "gui/title3_completed.png")
	}

	g.musicSlider.SetValue(g.userData.MusicVol)
	g.sfxSlider.SetValue(g.userData.SfxVol)

	g.UpdateMusicButton()
	g.UpdateSfxButton()

	// Done Loading - hide the loading label, show the menu, and initialize the level
	g.loadingLabel.SetVisible(false)
	g.menu.Add(g.main)
	g.InitLevel(g.userData.LastLevel)

	now := time.Now()
	newNow := time.Now()
	log.Info("Starting Render Loop")

	// Start the render loop
	for !g.win.ShouldClose() {

		newNow = time.Now()
		timeDelta := now.Sub(newNow)
		now = newNow

		g.Update(timeDelta.Seconds())
		g.RenderFrame()
	}
}

// RenderFrame renders a frame of the scene with the GUI overlaid
func (g *GokobanGame) RenderFrame() {

	// Clear buffers
	g.gs.Clear(gls.DEPTH_BUFFER_BIT | gls.STENCIL_BUFFER_BIT | gls.COLOR_BUFFER_BIT)

	// Render the scene using the specified scene and camera
	g.renderer.Render(g.scene, g.camera)

	// Process GUI timers
	g.root.TimerManager.ProcessTimers()

	// Render GUI over everything
	g.gs.Clear(gls.DEPTH_BUFFER_BIT)
	err := g.renderer.Render(g.root, g.camera)
	if err != nil {
		log.Fatal("Render error: %s\n", err)
	}

	// Update window and check for I/O events
	g.win.SwapBuffers()
	g.win.PollEvents()
}


// TODO make NewLevel a method of GokobanGame ?