// Copyright 2017 Daniel Salvadori. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"github.com/g3n/engine/audio"
	"github.com/g3n/engine/audio/al"
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

	"flag"
	"fmt"
	"github.com/g3n/engine/geometry"
	"io/ioutil"
	"runtime"
	"strconv"
	"time"
	"os"
	"strings"
	"path/filepath"
)

//      ____       _         _
//     / ___| ___ | | _____ | |__   __ _ _ __
//    | |  _ / _ \| |/ / _ \| '_ \ / _` | '_ \
//    | |_| | (_) |   < (_) | |_) | (_| | | | |
//     \____|\___/|_|\_\___/|_.__/ \__,_|_| |_|
//

const CREDITS_LINE1 string = "Open source game by Daniel Salvadori (github.com/danaugrs/gokoban). Written in Go and powered by g3n (github.com/g3n/engine)."
const CREDITS_LINE2 string = "Music by Eric Matyas (www.soundimage.org)."

const INSTRUCTIONS_LINE1 string = "Click and drag to look around. Use the mouse wheel to zoom."
const INSTRUCTIONS_LINE2 string = "Use WASD or the arrow keys to move the gopher relative to the camera."
const INSTRUCTIONS_LINE3 string = "Push the box on top the yellow pad, Gopher!"
const INSTRUCTIONS_LINE3_COMPLETE string = "Well done! Proceed to the next level by clicking on the top right corner."

var log *logger.Logger

type GokobanGame struct {
	wmgr         window.IWindowManager
	win          window.IWindow
	gs           *gls.GLS
	renderer     *renderer.Renderer
	scene        *core.Node
	camera       *camera.Perspective
	orbitControl *control.OrbitControl
	dataDir      string

	userData *UserData

	root     *gui.Root
	menu     *gui.Panel
	main     *gui.Panel
	controls *gui.Panel

	stepDelta     *math32.Vector2
	musicCheckbox *gui.CheckRadio
	musicSlider   *gui.Slider

	sfxCheckbox *gui.CheckRadio
	sfxSlider   *gui.Slider

	loadingLabel        *gui.ImageLabel
	instructions1       *gui.ImageLabel
	instructions2       *gui.ImageLabel
	instructions3       *gui.ImageLabel
	instructionsRestart *gui.ImageLabel
	instructionsMenu    *gui.ImageLabel

	levelLabel       *gui.ImageButton
	titleImage       *gui.ImageButton
	nextButton       *gui.ImageButton
	prevButton       *gui.ImageButton
	restartButton    *gui.ImageButton
	menuButton       *gui.ImageButton
	quitButton       *gui.ImageButton
	playButton       *gui.ImageButton
	sfxButton        *gui.ImageButton
	musicButton      *gui.ImageButton
	fullScreenButton *gui.ImageButton

	levelScene *core.Node
	levelStyle *LevelStyle
	levels     []*Level
	levelsRaw  []string
	level      *Level
	leveln     int

	gopherLocked   bool
	gopherNode     *core.Node
	arrowNode      *core.Node
	steps          int
	audioAvailable bool

	// Sound/music players
	musicPlayer           *audio.Player
	musicPlayerMenu       *audio.Player
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
func (g *GokobanGame) RestartLevel(playSound bool) {
	log.Debug("Restart Level")

	if g.leveln == 0 {
		g.instructions3.SetText(INSTRUCTIONS_LINE3)
	}

	g.instructions1.SetVisible(g.leveln == 0)
	g.instructions2.SetVisible(g.leveln == 0)
	g.instructions3.SetVisible(g.leveln == 0)
	g.instructionsRestart.SetVisible(g.leveln == 0)
	g.instructionsMenu.SetVisible(g.leveln == 0)
	g.arrowNode.SetVisible(g.leveln == 0)

	// If the menu is not visible then "free" the gopher
	// The menu would be visible if the user fell or dropped a box and then opened the menu before the fall ended
	// If the menu is visible then we want to keep the gopher locked
	if !g.menu.Visible() {
		g.gopherLocked = false
	}

	g.levels[g.leveln].Restart(playSound)
}

// NextLevel loads the next level if exists
func (g *GokobanGame) NextLevel() {
	log.Debug("Next Level")

	if g.leveln < len(g.levels)-1 {
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

	if g.menu.Visible() {

		// Dispatch OnMouseUp to clear the orbit control if user had mouse button pressed when they pressed Esc to hide menu
		g.win.Dispatch(gui.OnMouseUp, &window.MouseEvent{})

		// Dispatch OnCursorLeave to sliders in case user had cursor over sliders when they pressed Esc to hide menu
		g.sfxSlider.Dispatch(gui.OnCursorLeave, &window.MouseEvent{})
		g.musicSlider.Dispatch(gui.OnCursorLeave, &window.MouseEvent{})

		g.menu.SetVisible(false)
		g.controls.SetVisible(true)
		g.orbitControl.Enabled = true
		g.gopherLocked = false
		if g.audioAvailable {
			g.musicPlayerMenu.Stop()
			g.musicPlayer.Play()
		}
	} else {
		g.menu.SetVisible(true)
		g.controls.SetVisible(false)
		g.orbitControl.Enabled = false
		g.gopherLocked = true
		if g.audioAvailable {
			g.musicPlayer.Stop()
			g.musicPlayerMenu.Play()
		}
	}
}

// Quit saves the user data and quits the game
func (g *GokobanGame) Quit() {
	log.Debug("Quit")

	// Copy settings into user data and save
	g.userData.SfxVol = g.sfxSlider.Value()
	g.userData.MusicVol = g.musicSlider.Value()
	g.userData.FullScreen = g.win.FullScreen()
	g.userData.Save(g.dataDir)

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
		if !g.menu.Visible() && g.steps > 0 {
			g.RestartLevel(true)
		}
	}
}

// onMouse handles mouse events for the game
func (g *GokobanGame) onMouse(evname string, ev interface{}) {
	mev := ev.(*window.MouseEvent)

	if g.gopherLocked == false && g.leveln > 0 {
		// Mouse button pressed
		if mev.Action == window.Press {
			// Left button pressed
			if mev.Button == window.MouseButtonLeft {
				g.arrowNode.SetVisible(true)
			}
		} else if mev.Action == window.Release {
			g.arrowNode.SetVisible(false)
		}
	}
}

// onCursor handles cursor movement for the game
func (g *GokobanGame) onCursor(evname string, ev interface{}) {

	// Calculate direction of potential movement based on camera angle
	var dir math32.Vector3
	g.camera.WorldDirection(&dir)
	g.stepDelta.Set(0, 0)

	if math32.Abs(dir.Z) > math32.Abs(dir.X) {
		if dir.Z > 0 {
			g.arrowNode.SetRotationY(3 * math32.Pi / 2)
			g.stepDelta.Y = 1
		} else {
			g.arrowNode.SetRotationY(1 * math32.Pi / 2)
			g.stepDelta.Y = -1
		}
	} else {
		if dir.X > 0 {
			g.arrowNode.SetRotationY(4 * math32.Pi / 2)
			g.stepDelta.X = 1
		} else {
			g.arrowNode.SetRotationY(2 * math32.Pi / 2)
			g.stepDelta.X = -1
		}
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

	if g.leveln == 0 {
		g.instructions3.SetText(INSTRUCTIONS_LINE3_COMPLETE)
	}

	if g.userData.LastUnlockedLevel == g.leveln {
		g.userData.LastUnlockedLevel++
		g.userData.Save(g.dataDir) // Save in case game crashes
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
	g.titleImage.SetImage(gui.ButtonDisabled, g.dataDir + "/gui/title3_completed.png")
}

// InitLevel initializes the level associated to the provided index
func (g *GokobanGame) InitLevel(n int) {
	log.Debug("Initializing Level %v", n+1)

	// Always enable the button to return to the previous level except when we are in the very first level
	g.prevButton.SetEnabled(n != 0)

	// The button to go to the next level has 3 different states: disabled, locked and enabled
	// If this is the very last level - disable it completely
	if n == len(g.levels)-1 {
		g.nextButton.SetImage(gui.ButtonDisabled, g.dataDir + "/gui/right_disabled2.png")
		g.nextButton.SetEnabled(false)
	} else {
		g.nextButton.SetImage(gui.ButtonDisabled, g.dataDir + "/gui/right_disabled_locked.png")
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

	g.RestartLevel(false)
	g.level.gopherNodeRotate.Add(g.gopherNode)
	g.level.gopherNodeTranslate.Add(g.arrowNode)
	g.levelLabel.SetText("Level " + strconv.Itoa(n+1))
	g.levelScene.Add(g.level.scene)
	g.win.SubscribeID(window.OnKeyDown, g.leveln, g.level.onKey)

}

// LoadLevels reads and parses the level files inside ./levels, building an array of Level objects
func (g *GokobanGame) LoadLevels() {
	log.Debug("Load Levels")

	files, _ := ioutil.ReadDir(g.dataDir + "/levels")
	g.levels = make([]*Level, len(files)-1)

	for i, f := range files {

		// Skip README.md
		if f.Name() == "README.md" {
			continue
		}

		log.Debug("Reading level file: %v as level %v", f.Name(), i+1)

		// Read level text file
		b, err := ioutil.ReadFile(g.dataDir + "/levels/" + f.Name())
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
	listener.SetDirectionVec(&cdir)
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

	g.musicPlayer = createPlayer(g.dataDir + "/audio/music/Lost-Jungle_Looping.ogg")
	g.musicPlayer.SetLooping(true)

	g.musicPlayerMenu = createPlayer(g.dataDir + "/audio/music/Spooky-Island.ogg")
	g.musicPlayerMenu.SetLooping(true)

	rFactor := float32(0.2)

	g.clickPlayer = createPlayer(g.dataDir + "/audio/sfx/button_click.ogg")
	g.clickPlayer.SetRolloffFactor(rFactor)

	g.hoverPlayer = createPlayer(g.dataDir + "/audio/sfx/button_hover.ogg")
	g.hoverPlayer.SetRolloffFactor(rFactor)

	g.walkPlayer = createPlayer(g.dataDir + "/audio/sfx/gopher_walk.ogg")
	g.walkPlayer.SetRolloffFactor(rFactor)

	g.bumpPlayer = createPlayer(g.dataDir + "/audio/sfx/gopher_bump.ogg")
	g.bumpPlayer.SetRolloffFactor(rFactor)

	g.gopherFallStartPlayer = createPlayer(g.dataDir + "/audio/sfx/gopher_fall_start.ogg")
	g.gopherFallStartPlayer.SetRolloffFactor(rFactor)

	g.gopherFallEndPlayer = createPlayer(g.dataDir + "/audio/sfx/gopher_fall_end.ogg")
	g.gopherFallEndPlayer.SetRolloffFactor(rFactor)

	g.gopherHurtPlayer = createPlayer(g.dataDir + "/audio/sfx/gopher_hurt.ogg")
	g.gopherHurtPlayer.SetRolloffFactor(rFactor)

	g.boxPushPlayer = createPlayer(g.dataDir + "/audio/sfx/box_push.ogg")
	g.boxPushPlayer.SetRolloffFactor(rFactor)

	g.boxOnPadPlayer = createPlayer(g.dataDir + "/audio/sfx/box_on.ogg")
	g.boxOnPadPlayer.SetRolloffFactor(rFactor)

	g.boxOffPadPlayer = createPlayer(g.dataDir + "/audio/sfx/box_off.ogg")
	g.boxOffPadPlayer.SetRolloffFactor(rFactor)

	g.boxFallStartPlayer = createPlayer(g.dataDir + "/audio/sfx/box_fall_start.ogg")
	g.boxFallStartPlayer.SetRolloffFactor(rFactor)

	g.boxFallEndPlayer = createPlayer(g.dataDir + "/audio/sfx/box_fall_end.ogg")
	g.boxFallEndPlayer.SetRolloffFactor(rFactor)

	g.elevatorUpPlayer = createPlayer(g.dataDir + "/audio/sfx/elevator_up.ogg")
	g.elevatorUpPlayer.SetLooping(true)
	g.elevatorUpPlayer.SetRolloffFactor(rFactor)

	g.elevatorDownPlayer = createPlayer(g.dataDir + "/audio/sfx/elevator_down.ogg")
	g.elevatorDownPlayer.SetLooping(true)
	g.elevatorDownPlayer.SetRolloffFactor(rFactor)

	g.levelDonePlayer = createPlayer(g.dataDir + "/audio/sfx/level_done.ogg")
	g.levelDonePlayer.SetRolloffFactor(rFactor)

	g.levelRestartPlayer = createPlayer(g.dataDir + "/audio/sfx/level_restart.ogg")
	g.levelRestartPlayer.SetRolloffFactor(rFactor)

	g.levelFailPlayer = createPlayer(g.dataDir + "/audio/sfx/level_fail.ogg")
	g.levelFailPlayer.SetRolloffFactor(rFactor)

	g.gameCompletePlayer = createPlayer(g.dataDir + "/audio/sfx/game_complete.ogg")
	g.gameCompletePlayer.SetRolloffFactor(rFactor)
}

// LoadSkybox loads the space skybox and adds it to the scene
func (g *GokobanGame) LoadSkyBox() {
	log.Debug("Creating Skybox...")

	// Load skybox textures
	skyboxData := graphic.SkyboxData{
		g.dataDir + "/img/skybox/", "jpg",
		[6]string{"px", "nx", "py", "ny", "pz", "nz"}}

	skybox, err := graphic.NewSkybox(skyboxData)
	if err != nil {
		panic(err)
	}
	skybox.SetRenderOrder(-1) // The skybox should always be rendered first

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
	dec, err := obj.Decode(g.dataDir + "/gopher/gopher.obj", g.dataDir + "/gopher/gopher.mtl")
	if err != nil {
		panic(err.Error())
	}

	// Create a new node with all the objects in the decoded file and adds it to the scene
	gopherTop, err := dec.NewGroup()
	if err != nil {
		panic(err.Error())
	}

	g.gopherNode = core.NewNode()
	g.gopherNode.Add(gopherTop)

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

// NewArrowGeometry returns a pointer to a new arrow-shaped Geometry
func NewArrowGeometry(p float32) *geometry.Geometry {

	// Builds array with vertex positions and texture coordinates
	positions := math32.NewArrayF32(0, 20)
	positions.Append(
		0, 0.25, 0, 0, 0, 1,
		0, -0.25, 0, 0, 0, 1,
		1, -0.25, 0, 0, 0, 1,
		1, 0.25, 0, 0, 0, 1,
		1, 0.25+p, 0, 0, 0, 1,
		1, -0.25-p, 0, 0, 0, 1,
		2, 0, 0, 0, 0, 1,
	)
	// Builds array of indices
	indices := math32.NewArrayU32(0, 6)
	indices.Append(
		0, 1, 2,
		0, 2, 3,
		4, 5, 6,
	)

	// Creates geometry
	geom := geometry.NewGeometry()
	geom.SetIndices(indices)
	geom.AddVBO(gls.NewVBO(positions).
		AddAttrib(gls.VertexPosition).
		AddAttrib(gls.VertexNormal),
	)

	return geom
}

// Create the four arrows shown on top of the Gopher
func (g *GokobanGame) CreateArrowNode() {

	g.arrowNode = core.NewNode()
	arrowGeom := NewArrowGeometry(0.5)
	arrowMaterial := material.NewStandard(&math32.Color{0.628, 0.882, 0.1})
	arrowMaterial.SetSide(material.SideDouble)

	arrowMesh := graphic.NewMesh(arrowGeom, arrowMaterial)
	arrowMesh.SetScale(0.2, 0.2, 1)
	arrowMesh.SetPosition(0, 0.6, 0)
	arrowMesh.SetRotationX(-math32.Pi / 2)
	g.arrowNode.Add(arrowMesh)

	arrowMeshLeft := graphic.NewMesh(arrowGeom, arrowMaterial)
	arrowMeshLeft.SetScale(0.1, 0.1, 1)
	arrowMeshLeft.SetPosition(0, 0.6, 0)
	arrowMeshLeft.SetRotationX(-math32.Pi / 2)
	arrowMeshLeft.SetRotationY(-math32.Pi / 2)
	g.arrowNode.Add(arrowMeshLeft)

	arrowMeshRight := graphic.NewMesh(arrowGeom, arrowMaterial)
	arrowMeshRight.SetScale(0.1, 0.1, 1)
	arrowMeshRight.SetPosition(0, 0.6, 0)
	arrowMeshRight.SetRotationX(-math32.Pi / 2)
	arrowMeshRight.SetRotationY(math32.Pi / 2)
	g.arrowNode.Add(arrowMeshRight)

	arrowMeshBack := graphic.NewMesh(arrowGeom, arrowMaterial)
	arrowMeshBack.SetScale(0.1, 0.1, 1)
	arrowMeshBack.SetPosition(0, 0.6, 0)
	arrowMeshBack.SetRotationX(-math32.Pi / 2)
	arrowMeshBack.SetRotationY(2 * math32.Pi / 2)
	g.arrowNode.Add(arrowMeshBack)

	arrowMaterialB := material.NewStandard(math32.NewColor("black"))
	arrowMaterialB.SetSide(material.SideDouble)
	arrowGeomB := NewArrowGeometry(0.5)
	positions := arrowGeomB.VBO(gls.VertexPosition)
	buffer := math32.NewArrayF32(0, 20)
	buffer.Append(
		-0.1, 0.35, 0, 0, 0, 1,
		-0.1, -0.35, 0, 0, 0, 1,
		0.9, -0.35, 0, 0, 0, 1,
		0.9, 0.35, 0, 0, 0, 1,
		0.9, 0.95, 0, 0, 0, 1,
		0.9, -0.95, 0, 0, 0, 1,
		2.2, 0, 0, 0, 0, 1,
	)
	positions.SetBuffer(buffer)

	arrowMeshB := graphic.NewMesh(arrowGeomB, arrowMaterialB)
	arrowMeshB.SetScale(0.175, 0.175, 1)
	arrowMeshB.SetPosition(0.034, 0.599, 0)
	arrowMeshB.SetRotationX(-math32.Pi / 2)
	g.arrowNode.Add(arrowMeshB)

	arrowMeshLeftB := graphic.NewMesh(arrowGeomB, arrowMaterialB)
	arrowMeshLeftB.SetScale(0.1, 0.1, 1)
	arrowMeshLeftB.SetPosition(0, 0.599, 0)
	arrowMeshLeftB.SetRotationX(-math32.Pi / 2)
	arrowMeshLeftB.SetRotationY(-math32.Pi / 2)
	g.arrowNode.Add(arrowMeshLeftB)

	arrowMeshRightB := graphic.NewMesh(arrowGeomB, arrowMaterialB)
	arrowMeshRightB.SetScale(0.1, 0.1, 1)
	arrowMeshRightB.SetPosition(0, 0.599, 0)
	arrowMeshRightB.SetRotationX(-math32.Pi / 2)
	arrowMeshRightB.SetRotationY(math32.Pi / 2)
	g.arrowNode.Add(arrowMeshRightB)

	arrowMeshBackB := graphic.NewMesh(arrowGeomB, arrowMaterialB)
	arrowMeshBackB.SetScale(0.1, 0.1, 1)
	arrowMeshBackB.SetPosition(0, 0.599, 0)
	arrowMeshBackB.SetRotationX(-math32.Pi / 2)
	arrowMeshBackB.SetRotationY(2 * math32.Pi / 2)
	g.arrowNode.Add(arrowMeshBackB)

	arrowMeshBB := graphic.NewMesh(arrowGeom, arrowMaterial)
	arrowMeshBB.SetScale(0.2, 0.2, 1)
	arrowMeshBB.SetPosition(0, 0.598, 0)
	arrowMeshBB.SetRotationX(-math32.Pi / 2)
	g.arrowNode.Add(arrowMeshBB)

	arrowMeshLeftBB := graphic.NewMesh(arrowGeom, arrowMaterial)
	arrowMeshLeftBB.SetScale(0.1, 0.1, 1)
	arrowMeshLeftBB.SetPosition(0, 0.598, 0)
	arrowMeshLeftBB.SetRotationX(-math32.Pi / 2)
	arrowMeshLeftBB.SetRotationY(-math32.Pi / 2)
	g.arrowNode.Add(arrowMeshLeftBB)

	arrowMeshRightBB := graphic.NewMesh(arrowGeom, arrowMaterial)
	arrowMeshRightBB.SetScale(0.1, 0.1, 1)
	arrowMeshRightBB.SetPosition(0, 0.598, 0)
	arrowMeshRightBB.SetRotationX(-math32.Pi / 2)
	arrowMeshRightBB.SetRotationY(math32.Pi / 2)
	g.arrowNode.Add(arrowMeshRightBB)

	arrowMeshBackBB := graphic.NewMesh(arrowGeom, arrowMaterial)
	arrowMeshBackBB.SetScale(0.1, 0.1, 1)
	arrowMeshBackBB.SetPosition(0, 0.598, 0)
	arrowMeshBackBB.SetRotationX(-math32.Pi / 2)
	arrowMeshBackBB.SetRotationY(2 * math32.Pi / 2)
	g.arrowNode.Add(arrowMeshBackBB)
}

func (g *GokobanGame) UpdateMusicButton(on bool) {
	if on {
		g.musicButton.SetImage(gui.ButtonNormal, g.dataDir + "/gui/music_normal.png")
		g.musicButton.SetImage(gui.ButtonOver, g.dataDir + "/gui/music_hover.png")
		g.musicButton.SetImage(gui.ButtonPressed, g.dataDir + "/gui/music_click.png")
		g.musicSlider.SetEnabled(true)
		g.musicSlider.SetValue(g.musicSlider.Value())
	} else {
		g.musicButton.SetImage(gui.ButtonNormal, g.dataDir + "/gui/music_normal_off.png")
		g.musicButton.SetImage(gui.ButtonOver, g.dataDir + "/gui/music_hover_off.png")
		g.musicButton.SetImage(gui.ButtonPressed, g.dataDir + "/gui/music_click_off.png")
		g.musicSlider.SetEnabled(false)
		g.SetMusicVolume(0)
	}
}

func (g *GokobanGame) UpdateSfxButton(on bool) {
	if on {
		g.sfxButton.SetImage(gui.ButtonNormal, g.dataDir + "/gui/sound_normal.png")
		g.sfxButton.SetImage(gui.ButtonOver, g.dataDir + "/gui/sound_hover.png")
		g.sfxButton.SetImage(gui.ButtonPressed, g.dataDir + "/gui/sound_click.png")
		g.sfxSlider.SetEnabled(true)
		g.sfxSlider.SetValue(g.sfxSlider.Value())
	} else {
		g.sfxButton.SetImage(gui.ButtonNormal, g.dataDir + "/gui/sound_normal_off.png")
		g.sfxButton.SetImage(gui.ButtonOver, g.dataDir + "/gui/sound_hover_off.png")
		g.sfxButton.SetImage(gui.ButtonPressed, g.dataDir + "/gui/sound_click_off.png")
		g.sfxSlider.SetEnabled(false)
		g.SetSfxVolume(0)
	}
}

// SetupGui creates all user interface elements
func (g *GokobanGame) SetupGui(width, height int) {
	log.Debug("Creating GUI...")

	transparent := math32.Color4{0, 0, 0, 0}
	blackTextColor := math32.Color4{0.3, 0.3, 0.3, 1.0}
	creditsColor := math32.Color{0.6, 0.6, 0.6}
	sliderColor := math32.Color4{0.628, 0.882, 0.1, 1}
	sliderColorOff := math32.Color4{0.82, 0.48, 0.48, 1}
	sliderColorOver := math32.Color4{0.728, 0.982, 0.2, 1}
	sliderBorderColor := math32.Color4{0.71, 0.482, 0.26, 1}

	sliderBorder := gui.RectBounds{3, 3, 3, 3}
	//zeroBorder := gui.RectBounds{0, 0, 0, 0}

	s := gui.StyleDefault()
	s.ImageButton = gui.ImageButtonStyles{}
	s.ImageButton.Normal = gui.ImageButtonStyle{}
	s.ImageButton.Normal.BgColor = transparent
	s.ImageButton.Normal.FgColor = blackTextColor
	s.ImageButton.Over = s.ImageButton.Normal
	s.ImageButton.Focus = s.ImageButton.Normal
	s.ImageButton.Pressed = s.ImageButton.Normal
	s.ImageButton.Disabled = s.ImageButton.Normal

	s.Slider = gui.SliderStyles{}
	s.Slider.Normal = gui.SliderStyle{}
	s.Slider.Normal.Border = sliderBorder
	s.Slider.Normal.BorderColor = sliderBorderColor
	s.Slider.Normal.BgColor = math32.Color4{0.2, 0.2, 0.2, 1}
	s.Slider.Normal.FgColor = sliderColor
	s.Slider.Over = s.Slider.Normal
	s.Slider.Over.BgColor = math32.Color4{0.3, 0.3, 0.3, 1}
	s.Slider.Over.FgColor = sliderColorOver
	s.Slider.Focus = s.Slider.Over
	s.Slider.Disabled = s.Slider.Normal
	s.Slider.Disabled.FgColor = sliderColorOff

	var err error

	hoverSound := func(evname string, ev interface{}) {
		g.PlaySound(g.hoverPlayer, nil)
	}

	// Menu
	g.menu = gui.NewPanel(100, 100)
	g.menu.SetColor4(&math32.Color4{0.1, 0.1, 0.1, 0.6})
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
	g.prevButton, err = gui.NewImageButton(g.dataDir + "/gui/left_normal.png")
	g.prevButton.SetImage(gui.ButtonOver, g.dataDir + "/gui/left_hover.png")
	g.prevButton.SetImage(gui.ButtonPressed, g.dataDir + "/gui/left_click.png")
	g.prevButton.SetImage(gui.ButtonDisabled, g.dataDir + "/gui/left_disabled2.png")
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
	g.levelLabel, err = gui.NewImageButton(g.dataDir + "/gui/panel.png")
	g.levelLabel.SetImage(gui.ButtonDisabled, g.dataDir + "/gui/panel.png")
	g.levelLabel.SetColor(&math32.Color{0.8, 0.8, 0.8})
	g.levelLabel.SetText("Level")
	g.levelLabel.SetFontSize(35)
	g.levelLabel.SetEnabled(false)
	header.Add(g.levelLabel)

	spacer2 := gui.NewPanel(0, 0)
	spacer2.SetLayoutParams(&params)
	header.Add(spacer2)

	// Next Level Button
	g.nextButton, err = gui.NewImageButton(g.dataDir + "/gui/right_normal.png")
	g.nextButton.SetImage(gui.ButtonOver, g.dataDir + "/gui/right_hover.png")
	g.nextButton.SetImage(gui.ButtonPressed, g.dataDir + "/gui/right_click.png")
	g.nextButton.SetImage(gui.ButtonDisabled, g.dataDir + "/gui/right_disabled2.png")
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
	g.restartButton, err = gui.NewImageButton(g.dataDir + "/gui/restart_normal.png")
	g.restartButton.SetImage(gui.ButtonOver, g.dataDir + "/gui/restart_hover.png")
	g.restartButton.SetImage(gui.ButtonPressed, g.dataDir + "/gui/restart_click.png")
	g.restartButton.SetImage(gui.ButtonDisabled, g.dataDir + "/gui/restart_disabled2.png")
	if err != nil {
		panic(err)
	}
	g.restartButton.Subscribe(gui.OnMouseUp, func(evname string, ev interface{}) {
		g.RestartLevel(true)
	})
	g.restartButton.Subscribe(gui.OnCursorEnter, hoverSound)
	footer.Add(g.restartButton)

	spacer3 := gui.NewPanel(0, 0)
	spacer3.SetLayoutParams(&params)
	footer.Add(spacer3)

	// Restart Level Button
	g.menuButton, err = gui.NewImageButton(g.dataDir + "/gui/menu_normal.png")
	g.menuButton.SetImage(gui.ButtonOver, g.dataDir + "/gui/menu_hover.png")
	g.menuButton.SetImage(gui.ButtonPressed, g.dataDir + "/gui/menu_click.png")
	g.menuButton.SetImage(gui.ButtonDisabled, g.dataDir + "/gui/menu_disabled2.png")
	if err != nil {
		panic(err)
	}
	g.menuButton.Subscribe(gui.OnMouseUp, func(evname string, ev interface{}) {
		g.PlaySound(g.clickPlayer, nil)
		g.ToggleMenu()
	})
	g.menuButton.Subscribe(gui.OnCursorEnter, hoverSound)
	footer.Add(g.menuButton)

	g.controls.SetVisible(false)
	g.root.Add(g.controls)

	// Title
	g.titleImage, err = gui.NewImageButton(g.dataDir + "/gui/title3.png")
	g.titleImage.SetImage(gui.ButtonDisabled, g.dataDir + "/gui/title3.png")
	g.titleImage.SetEnabled(false)
	g.root.Subscribe(gui.OnResize, func(evname string, ev interface{}) {
		g.titleImage.SetPositionX((g.root.ContentWidth() - g.titleImage.ContentWidth()) / 2)
	})
	g.menu.Add(g.titleImage)

	// Loading Text
	g.loadingLabel = gui.NewImageLabel("Loading...")
	g.loadingLabel.SetColor(&math32.Color{1, 1, 1})
	g.loadingLabel.SetFontSize(40)
	g.root.Subscribe(gui.OnResize, func(evname string, ev interface{}) {
		g.loadingLabel.SetPositionX((g.root.ContentWidth() - g.loadingLabel.ContentWidth()) / 2)
		g.loadingLabel.SetPositionY((g.root.ContentHeight() - g.loadingLabel.ContentHeight()) / 2)
	})
	g.root.Add(g.loadingLabel)

	// Instructions
	g.instructions1 = gui.NewImageLabel(INSTRUCTIONS_LINE1)
	g.instructions1.SetColor(&creditsColor)
	g.instructions1.SetFontSize(28)
	g.root.Subscribe(gui.OnResize, func(evname string, ev interface{}) {
		g.instructions1.SetWidth(g.root.ContentWidth())
		g.instructions1.SetPositionY(4 * g.instructions1.ContentHeight())
	})
	g.controls.Add(g.instructions1)

	g.instructions2 = gui.NewImageLabel(INSTRUCTIONS_LINE2)
	g.instructions2.SetColor(&creditsColor)
	g.instructions2.SetFontSize(28)
	g.root.Subscribe(gui.OnResize, func(evname string, ev interface{}) {
		g.instructions2.SetWidth(g.root.ContentWidth())
		g.instructions2.SetPositionY(5 * g.instructions2.ContentHeight())
	})
	g.controls.Add(g.instructions2)

	g.instructions3 = gui.NewImageLabel(INSTRUCTIONS_LINE3)
	g.instructions3.SetColor(&creditsColor)
	g.instructions3.SetFontSize(28)
	g.root.Subscribe(gui.OnResize, func(evname string, ev interface{}) {
		g.instructions3.SetWidth(g.root.ContentWidth())
		g.instructions3.SetPositionY(g.root.ContentHeight() - 2*g.instructions3.ContentHeight())
	})
	g.controls.Add(g.instructions3)

	buttonInstructionsPad := float32(24)

	g.instructionsRestart = gui.NewImageLabel("Restart Level (R)")
	g.instructionsRestart.SetColor(&creditsColor)
	g.instructionsRestart.SetFontSize(20)
	g.root.Subscribe(gui.OnResize, func(evname string, ev interface{}) {
		g.instructionsRestart.SetPosition(buttonInstructionsPad, g.root.ContentHeight()-6*g.instructionsRestart.ContentHeight())
	})
	g.controls.Add(g.instructionsRestart)

	g.instructionsMenu = gui.NewImageLabel("Show Menu (Esc)")
	g.instructionsMenu.SetColor(&creditsColor)
	g.instructionsMenu.SetFontSize(20)
	g.root.Subscribe(gui.OnResize, func(evname string, ev interface{}) {
		g.instructionsMenu.SetPosition(g.root.ContentWidth()-g.instructionsMenu.ContentWidth()-buttonInstructionsPad, g.root.ContentHeight()-6*g.instructionsMenu.ContentHeight())
	})
	g.controls.Add(g.instructionsMenu)

	// Main panel
	g.main = gui.NewPanel(600, 300)
	mainLayout := gui.NewVBoxLayout()
	mainLayout.SetAlignV(gui.AlignHeight)
	g.main.SetLayout(mainLayout)
	g.main.SetBorders(2, 2, 2, 2)
	g.main.SetBordersColor4(&sliderBorderColor)
	g.main.SetColor4(&math32.Color4{0.2, 0.2, 0.2, 0.6})
	g.root.Subscribe(gui.OnResize, func(evname string, ev interface{}) {
		g.main.SetPositionX((g.root.Width() - g.main.Width()) / 2)
		g.main.SetPositionY((g.root.Height()-g.main.Height())/2 + 50)
	})

	topRow := gui.NewPanel(g.main.ContentWidth(), 100)
	topRowLayout := gui.NewHBoxLayout()
	topRowLayout.SetAlignH(gui.AlignWidth)
	topRow.SetLayout(topRowLayout)
	alignCenterVerical := gui.HBoxLayoutParams{Expand: 0, AlignV: gui.AlignCenter}

	// Music Control
	musicControl := gui.NewPanel(130, 100)
	musicControl.SetLayout(topRowLayout)

	g.musicButton, err = gui.NewImageButton(g.dataDir + "/gui/music_normal.png")
	g.musicButton.SetImage(gui.ButtonOver, g.dataDir + "/gui/music_hover.png")
	g.musicButton.SetImage(gui.ButtonPressed, g.dataDir + "/gui/music_click.png")
	g.musicButton.SetImage(gui.ButtonDisabled, g.dataDir + "/gui/music_disabled2.png")
	if err != nil {
		panic(err)
	}
	g.musicButton.Subscribe(gui.OnMouseUp, func(evname string, ev interface{}) {
		g.PlaySound(g.clickPlayer, nil)
		g.userData.MusicOn = !g.userData.MusicOn
		g.UpdateMusicButton(g.userData.MusicOn)
	})
	g.musicButton.Subscribe(gui.OnCursorEnter, hoverSound)
	musicControl.Add(g.musicButton)

	// Music Volume Slider
	g.musicSlider = gui.NewVSlider(20, 80)
	g.musicSlider.SetValue(g.userData.MusicVol)
	g.musicSlider.Subscribe(gui.OnChange, func(evname string, ev interface{}) {
		g.SetMusicVolume(g.musicSlider.Value())
	})
	g.musicSlider.Subscribe(gui.OnCursorEnter, hoverSound)
	g.musicSlider.SetLayoutParams(&alignCenterVerical)
	musicControl.Add(g.musicSlider)

	topRow.Add(musicControl)

	// Sound Effects Control
	sfxControl := gui.NewPanel(130, 100)
	sfxControl.SetLayout(topRowLayout)

	g.sfxButton, err = gui.NewImageButton(g.dataDir + "/gui/sound_normal.png")
	g.sfxButton.SetImage(gui.ButtonOver, g.dataDir + "/gui/sound_hover.png")
	g.sfxButton.SetImage(gui.ButtonPressed, g.dataDir + "/gui/sound_click.png")
	g.sfxButton.SetImage(gui.ButtonDisabled, g.dataDir + "/gui/sound_disabled2.png")
	if err != nil {
		panic(err)
	}
	g.sfxButton.Subscribe(gui.OnMouseUp, func(evname string, ev interface{}) {
		g.PlaySound(g.clickPlayer, nil)
		g.userData.SfxOn = !g.userData.SfxOn
		g.UpdateSfxButton(g.userData.SfxOn)
	})
	g.sfxButton.Subscribe(gui.OnCursorEnter, hoverSound)
	sfxControl.Add(g.sfxButton)

	// Sound Effects Volume Slider
	g.sfxSlider = gui.NewVSlider(20, 80)
	g.sfxSlider.SetValue(g.userData.SfxVol)
	g.sfxSlider.Subscribe(gui.OnChange, func(evname string, ev interface{}) {
		g.SetSfxVolume(3 * g.sfxSlider.Value())
	})
	g.sfxSlider.Subscribe(gui.OnCursorEnter, hoverSound)
	g.sfxSlider.SetLayoutParams(&alignCenterVerical)
	sfxControl.Add(g.sfxSlider)

	topRow.Add(sfxControl)

	// FullScreen Button
	g.fullScreenButton, err = gui.NewImageButton(g.dataDir + "/gui/screen_normal.png")
	g.fullScreenButton.SetImage(gui.ButtonOver, g.dataDir + "/gui/screen_hover.png")
	g.fullScreenButton.SetImage(gui.ButtonPressed, g.dataDir + "/gui/screen_click.png")
	g.fullScreenButton.SetImage(gui.ButtonDisabled, g.dataDir + "/gui/screen_disabled2.png")
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
	g.quitButton, err = gui.NewImageButton(g.dataDir + "/gui/quit_normal.png")
	g.quitButton.SetImage(gui.ButtonOver, g.dataDir + "/gui/quit_hover.png")
	g.quitButton.SetImage(gui.ButtonPressed, g.dataDir + "/gui/quit_click.png")
	g.quitButton.SetImage(gui.ButtonDisabled, g.dataDir + "/gui/quit_disabled2.png")
	if err != nil {
		panic(err)
	}
	g.quitButton.Subscribe(gui.OnMouseUp, func(evname string, ev interface{}) {
		g.Quit()
	})
	g.quitButton.Subscribe(gui.OnCursorEnter, hoverSound)
	buttonRow.Add(g.quitButton)

	// Play Button
	g.playButton, err = gui.NewImageButton(g.dataDir + "/gui/play_normal.png")
	g.playButton.SetImage(gui.ButtonOver, g.dataDir + "/gui/play_hover.png")
	g.playButton.SetImage(gui.ButtonPressed, g.dataDir + "/gui/play_click.png")
	g.playButton.SetImage(gui.ButtonDisabled, g.dataDir + "/gui/play_disabled2.png")
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
	g3n.SetImageFromFile(g.dataDir + "/img/g3n.png")
	g.root.Subscribe(gui.OnResize, func(evname string, ev interface{}) {
		g3n.SetPositionX(g.root.ContentWidth() - g3n.Width())
		g3n.SetPositionY(g.root.ContentHeight() - 1.3*g3n.Height())
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
	log.Debug("%s", vorbis.VersionString())
	return nil
}

func main() {
	// OpenGL functions must be executed in the same thread where
	// the context was created (by window.New())
	runtime.LockOSThread()

	// Parse command line flags
	showLog := flag.Bool("debug", false, "display the debug log")
	flag.Parse()

	// Create logger
	log = logger.New("Gokoban", nil)
	log.AddWriter(logger.NewConsole(false))
	log.SetFormat(logger.FTIME | logger.FMICROS)
	if *showLog == true {
		log.SetLevel(logger.DEBUG)
	} else {
		log.SetLevel(logger.INFO)
	}
	log.Info("Initializing Gokoban")

	// Create GokobanGame struct
	g := new(GokobanGame)

	// Manually scan the $GOPATH directories to find the data directory
	rawPaths := os.Getenv("GOPATH")
	paths := strings.Split(rawPaths, ":")
	for _, j := range paths {
		// Checks data path
		path := filepath.Join(j, "src", "github.com", "danaugrs", "gokoban")
		if _, err := os.Stat(path); err == nil {
			g.dataDir = path
		}
	}

	// Load user data from file
	g.userData = NewUserData(g.dataDir)

	// Get the window manager
	var err error
	g.wmgr, err = window.Manager("glfw")
	if err != nil {
		panic(err)
	}

	// Create window and OpenGL context
	g.win, err = g.wmgr.CreateWindow(1200, 900, "Gokoban", g.userData.FullScreen)
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
	width, height := g.win.Size()
	g.gs.Viewport(0, 0, int32(width), int32(height))

	// Creates GUI root panel
	g.root = gui.NewRoot(g.gs, g.win)
	g.root.SetSize(float32(width), float32(height))

	// Subscribe to window resize events. When the window is resized:
	// - Update the viewport size
	// - Update the root panel size
	// - Update the camera aspect ratio
	g.win.Subscribe(window.OnWindowSize, func(evname string, ev interface{}) {
		width, height := g.win.Size()
		g.gs.Viewport(0, 0, int32(width), int32(height))
		g.root.SetSize(float32(width), float32(height))
		aspect := float32(width) / float32(height)
		g.camera.SetAspect(aspect)
	})

	// Subscribe window to events
	g.win.Subscribe(window.OnKeyDown, g.onKey)
	g.win.Subscribe(window.OnMouseUp, g.onMouse)
	g.win.Subscribe(window.OnMouseDown, g.onMouse)

	// Creates a renderer and adds default shaders
	g.renderer = renderer.NewRenderer(g.gs)
	//g.renderer.SetSortObjects(false)
	err = g.renderer.AddDefaultShaders()
	if err != nil {
		panic(err)
	}
	g.renderer.SetGui(g.root)

	// Adds a perspective camera to the scene
	// The camera aspect ratio should be updated if the window is resized.
	aspect := float32(width) / float32(height)
	g.camera = camera.NewPerspective(65, aspect, 0.01, 1000)
	g.camera.SetPosition(0, 4, 5)
	g.camera.LookAt(&math32.Vector3{0, 0, 0})

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
	g.stepDelta = math32.NewVector2(0, 0)
	g.renderer.SetScene(g.scene)

	// Add white ambient light to the scene
	ambLight := light.NewAmbient(&math32.Color{1.0, 1.0, 1.0}, 0.4)
	g.scene.Add(ambLight)

	g.levelStyle = NewStandardStyle(g.dataDir)

	g.SetupGui(width, height)
	g.RenderFrame()

	// Try to open audio libraries
	err = loadAudioLibs()
	if err != nil {
		log.Error("%s", err)
		g.UpdateMusicButton(false)
		g.UpdateSfxButton(false)
		g.musicButton.SetEnabled(false)
		g.sfxButton.SetEnabled(false)
	} else {
		g.audioAvailable = true
		g.LoadAudio()
		g.UpdateMusicButton(g.userData.MusicOn)
		g.UpdateSfxButton(g.userData.SfxOn)

		// Queue the music!
		g.musicPlayerMenu.Play()
	}

	g.LoadSkyBox()
	g.LoadGopher()
	g.CreateArrowNode()
	g.LoadLevels()

	g.win.Subscribe(window.OnCursor, g.onCursor)

	if g.userData.LastUnlockedLevel == len(g.levels) {
		g.titleImage.SetImage(gui.ButtonDisabled, g.dataDir + "/gui/title3_completed.png")
	}

	// Done Loading - hide the loading label, show the menu, and initialize the level
	g.loadingLabel.SetVisible(false)
	g.menu.Add(g.main)
	g.InitLevel(g.userData.LastLevel)
	g.gopherLocked = true

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

	// Process GUI timers
	g.root.TimerManager.ProcessTimers()

	// Render the scene/gui using the specified camera
	rendered, err := g.renderer.Render(g.camera)
	if err != nil {
		panic(err)
	}

	// Check I/O events
	g.wmgr.PollEvents()

	// Update window if necessary
	if rendered {
		g.win.SwapBuffers()
	}
}
