// Copyright 2017 Daniel Salvadori. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package components

import (
	"strconv"
	"fmt"
	"github.com/g3n/engine/geometry"
	"io/ioutil"
	"github.com/g3n/engine/loader/obj"
	"github.com/g3n/engine/material"
	"github.com/g3n/engine/graphic"
	"github.com/g3n/engine/audio"
	"github.com/g3n/engine/audio/al"
	"github.com/g3n/engine/audio/vorbis"
	"github.com/g3n/engine/camera"
	"github.com/g3n/engine/camera/control"
	"github.com/g3n/engine/core"
	"github.com/g3n/engine/gls"
	"github.com/g3n/engine/gui"
	"github.com/g3n/engine/math32"
	"github.com/g3n/engine/renderer"
	"github.com/g3n/engine/window"
	"github.com/g3n/engine/util/logger"

)

var log *logger.Logger

const CREDITS_LINE1 string = "Open source game by Daniel Salvadori (github.com/danaugrs/gokoban). Written in Go and powered by g3n (github.com/g3n/engine)."
const CREDITS_LINE2 string = "Music by Eric Matyas (www.soundimage.org)."

const INSTRUCTIONS_LINE1 string = "Click and drag to look around. Use the mouse wheel to zoom."
const INSTRUCTIONS_LINE2 string = "Use WASD or the arrow keys to move the gopher relative to the camera."
const INSTRUCTIONS_LINE3 string = "Push the box on top the yellow pad, Gopher!"
const INSTRUCTIONS_LINE3_COMPLETE string = "Well done! Proceed to the next level by clicking on the top right corner."

func InitLogger(l *logger.Logger) {
	log = l
}
type GokobanGame struct {
	Wmgr         window.IWindowManager
	Win          window.IWindow
	Gs           *gls.GLS
	Renderer     *renderer.Renderer
	Scene        *core.Node
	Camera       *camera.Perspective
	OrbitControl *control.OrbitControl
	DataDir      string

	UserData *UserData

	Root     *gui.Root
	Menu     *gui.Panel
	Main     *gui.Panel
	controls *gui.Panel

	StepDelta     *math32.Vector2
	musicCheckbox *gui.CheckRadio
	musicSlider   *gui.Slider

	sfxCheckbox *gui.CheckRadio
	sfxSlider   *gui.Slider

	LoadingLabel        *gui.ImageLabel
	instructions1       *gui.ImageLabel
	instructions2       *gui.ImageLabel
	instructions3       *gui.ImageLabel
	instructionsRestart *gui.ImageLabel
	instructionsMenu    *gui.ImageLabel

	levelLabel       *gui.ImageButton
	TitleImage       *gui.ImageButton
	nextButton       *gui.ImageButton
	prevButton       *gui.ImageButton
	restartButton    *gui.ImageButton
	MenuButton       *gui.ImageButton
	quitButton       *gui.ImageButton
	playButton       *gui.ImageButton
	SfxButton        *gui.ImageButton
	MusicButton      *gui.ImageButton
	fullScreenButton *gui.ImageButton

	LevelScene *core.Node
	LevelStyle *LevelStyle
	Levels     []*Level
	levelsRaw  []string
	level      *Level
	leveln     int

	GopherLocked   bool
	gopherNode     *core.Node
	arrowNode      *core.Node
	steps          int
	AudioAvailable bool

	// Sound/music players
	musicPlayer           *audio.Player
	MusicPlayerMenu       *audio.Player
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

	// If the Menu is not visible then "free" the gopher
	// The Menu would be visible if the user fell or dropped a box and then opened the Menu before the fall ended
	// If the Menu is visible then we want to keep the gopher locked
	if !g.Menu.Visible() {
		g.GopherLocked = false
	}

	g.Levels[g.leveln].Restart(playSound)
}

// NextLevel loads the next level if exists
func (g *GokobanGame) NextLevel() {
	log.Debug("Next Level")

	if g.leveln < len(g.Levels)-1 {
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

// ToggleFullScreen toggles whether is game is fullscreen or Windowed
func (g *GokobanGame) ToggleFullScreen() {
	log.Debug("Toggle FullScreen")

	g.Win.SetFullScreen(!g.Win.FullScreen())
}

// ToggleMenu switched the Menu, title, and credits overlay for the in-level corner buttons
func (g *GokobanGame) ToggleMenu() {
	log.Debug("Toggle Menu")

	if g.Menu.Visible() {

		// Dispatch OnMouseUp to clear the orbit control if user had mouse button pressed when they pressed Esc to hide Menu
		g.Win.Dispatch(gui.OnMouseUp, &window.MouseEvent{})

		// Dispatch OnCursorLeave to sliders in case user had cursor over sliders when they pressed Esc to hide Menu
		g.sfxSlider.Dispatch(gui.OnCursorLeave, &window.MouseEvent{})
		g.musicSlider.Dispatch(gui.OnCursorLeave, &window.MouseEvent{})

		g.Menu.SetVisible(false)
		g.controls.SetVisible(true)
		g.OrbitControl.Enabled = true
		g.GopherLocked = false
		if g.AudioAvailable {
			g.MusicPlayerMenu.Stop()
			g.musicPlayer.Play()
		}
	} else {
		g.Menu.SetVisible(true)
		g.controls.SetVisible(false)
		g.OrbitControl.Enabled = false
		g.GopherLocked = true
		if g.AudioAvailable {
			g.musicPlayer.Stop()
			g.MusicPlayerMenu.Play()
		}
	}
}

// Quit saves the user data and quits the game
func (g *GokobanGame) Quit() {
	log.Debug("Quit")

	// Copy settings into user data and save
	g.UserData.SfxVol = g.sfxSlider.Value()
	g.UserData.MusicVol = g.musicSlider.Value()
	g.UserData.FullScreen = g.Win.FullScreen()
	g.UserData.Save(g.DataDir)

	// Close the Window
	g.Win.SetShouldClose(true)
}

// OnKey handles keyboard events for the game
func (g *GokobanGame) OnKey(evname string, ev interface{}) {

	kev := ev.(*window.KeyEvent)
	switch kev.Keycode {
	case window.KeyEscape:
		g.ToggleMenu()
	case window.KeyF:
		g.ToggleFullScreen()
	case window.KeyR:
		if !g.Menu.Visible() && g.steps > 0 {
			g.RestartLevel(true)
		}
	}
}

// OnMouse handles mouse events for the game
func (g *GokobanGame) OnMouse(evname string, ev interface{}) {
	mev := ev.(*window.MouseEvent)

	if g.GopherLocked == false && g.leveln > 0 {
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

// OnCursor handles cursor movement for the game
func (g *GokobanGame) OnCursor(evname string, ev interface{}) {

	// Calculate direction of potential movement based on camera angle
	var dir math32.Vector3
	g.Camera.WorldDirection(&dir)
	g.StepDelta.Set(0, 0)

	if math32.Abs(dir.Z) > math32.Abs(dir.X) {
		if dir.Z > 0 {
			g.arrowNode.SetRotationY(3 * math32.Pi / 2)
			g.StepDelta.Y = 1
		} else {
			g.arrowNode.SetRotationY(1 * math32.Pi / 2)
			g.StepDelta.Y = -1
		}
	} else {
		if dir.X > 0 {
			g.arrowNode.SetRotationY(4 * math32.Pi / 2)
			g.StepDelta.X = 1
		} else {
			g.arrowNode.SetRotationY(2 * math32.Pi / 2)
			g.StepDelta.X = -1
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

	if g.UserData.LastUnlockedLevel == g.leveln {
		g.UserData.LastUnlockedLevel++
		g.UserData.Save(g.DataDir) // Save in case game crashes
		if g.UserData.LastUnlockedLevel < len(g.Levels) {
			g.nextButton.SetEnabled(true)
		}
		if g.UserData.LastUnlockedLevel == len(g.Levels) {
			g.GameCompleted()
		}
	}
}

// GameCompleted stops the music, plays the the Winning sound, and changes the title image to say "Completed"
func (g *GokobanGame) GameCompleted() {
	log.Debug("Game Completed")

	if g.AudioAvailable {
		g.musicPlayer.Stop()
		g.PlaySound(g.gameCompletePlayer, nil)
	}
	g.TitleImage.SetImage(gui.ButtonDisabled, g.DataDir + "/gui/title3_completed.png")
}

// InitLevel initializes the level associated to the provided index
func (g *GokobanGame) InitLevel(n int) {
	log.Debug("Initializing Level %v", n+1)

	// Always enable the button to return to the previous level except when we are in the very first level
	g.prevButton.SetEnabled(n != 0)

	// The button to go to the next level has 3 different states: disabled, locked and enabled
	// If this is the very last level - disable it completely
	if n == len(g.Levels)-1 {
		g.nextButton.SetImage(gui.ButtonDisabled, g.DataDir + "/gui/right_disabled2.png")
		g.nextButton.SetEnabled(false)
	} else {
		g.nextButton.SetImage(gui.ButtonDisabled, g.DataDir + "/gui/right_disabled_locked.png")
		// check last completed level
		if g.UserData.LastUnlockedLevel == n {
			g.nextButton.SetEnabled(false)
		} else {
			g.nextButton.SetEnabled(true)
		}
	}

	// Remove level.Scene from LevelScene and unsubscribe from events
	if len(g.LevelScene.Children()) > 0 {
		g.LevelScene.Remove(g.level.scene)
		g.Win.UnsubscribeID(window.OnKeyDown, g.leveln)
	}

	// Update current level index and level reference
	g.leveln = n
	g.UserData.LastLevel = n
	g.level = g.Levels[g.leveln]

	g.RestartLevel(false)
	g.level.gopherNodeRotate.Add(g.gopherNode)
	g.level.gopherNodeTranslate.Add(g.arrowNode)
	g.levelLabel.SetText("Level " + strconv.Itoa(n+1))
	g.LevelScene.Add(g.level.scene)
	g.Win.SubscribeID(window.OnKeyDown, g.leveln, g.level.onKey)

}

// LoadLevels reads and parses the level files inside ./levels, building an array of Level objects
func (g *GokobanGame) LoadLevels() {
	log.Debug("Load Levels")

	files, _ := ioutil.ReadDir(g.DataDir + "/levels")
	g.Levels = make([]*Level, len(files)-1)

	for i, f := range files {

		// Skip README.md
		if f.Name() == "README.md" {
			continue
		}

		log.Debug("Reading level file: %v as level %v", f.Name(), i+1)

		// Read level text file
		b, err := ioutil.ReadFile(g.DataDir + "/levels/" + f.Name())
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
		g.Levels[i] = NewLevel(g, ld, g.LevelStyle, g.Camera)
	}
}

// SetSfxVolume sets the volume of all sound effects
func (g *GokobanGame) SetSfxVolume(vol float32) {
	log.Debug("Set Sfx Volume %v", vol)

	if g.AudioAvailable {
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

	if g.AudioAvailable {
		g.musicPlayer.SetGain(vol)
		g.MusicPlayerMenu.SetGain(vol)
	}
}

// LoadAudio loads music and sound effects
func (g *GokobanGame) LoadAudio() {
	log.Debug("Load Audio")

	// Create listener and add it to the current camera
	listener := audio.NewListener()
	cdir := g.Camera.Direction()
	listener.SetDirectionVec(&cdir)
	g.Camera.GetCamera().Add(listener)

	// Helper function to create player and handle errors
	createPlayer := func(fname string) *audio.Player {
		log.Debug("Loading " + fname)
		p, err := audio.NewPlayer(fname)
		if err != nil {
			log.Error("Failed to create player for: %v", fname)
		}
		return p
	}

	g.musicPlayer = createPlayer(g.DataDir + "/audio/music/Lost-Jungle_Looping.ogg")
	g.musicPlayer.SetLooping(true)

	g.MusicPlayerMenu = createPlayer(g.DataDir + "/audio/music/Spooky-Island.ogg")
	g.MusicPlayerMenu.SetLooping(true)

	rFactor := float32(0.2)

	g.clickPlayer = createPlayer(g.DataDir + "/audio/sfx/button_click.ogg")
	g.clickPlayer.SetRolloffFactor(rFactor)

	g.hoverPlayer = createPlayer(g.DataDir + "/audio/sfx/button_hover.ogg")
	g.hoverPlayer.SetRolloffFactor(rFactor)

	g.walkPlayer = createPlayer(g.DataDir + "/audio/sfx/gopher_walk.ogg")
	g.walkPlayer.SetRolloffFactor(rFactor)

	g.bumpPlayer = createPlayer(g.DataDir + "/audio/sfx/gopher_bump.ogg")
	g.bumpPlayer.SetRolloffFactor(rFactor)

	g.gopherFallStartPlayer = createPlayer(g.DataDir + "/audio/sfx/gopher_fall_start.ogg")
	g.gopherFallStartPlayer.SetRolloffFactor(rFactor)

	g.gopherFallEndPlayer = createPlayer(g.DataDir + "/audio/sfx/gopher_fall_end.ogg")
	g.gopherFallEndPlayer.SetRolloffFactor(rFactor)

	g.gopherHurtPlayer = createPlayer(g.DataDir + "/audio/sfx/gopher_hurt.ogg")
	g.gopherHurtPlayer.SetRolloffFactor(rFactor)

	g.boxPushPlayer = createPlayer(g.DataDir + "/audio/sfx/box_push.ogg")
	g.boxPushPlayer.SetRolloffFactor(rFactor)

	g.boxOnPadPlayer = createPlayer(g.DataDir + "/audio/sfx/box_on.ogg")
	g.boxOnPadPlayer.SetRolloffFactor(rFactor)

	g.boxOffPadPlayer = createPlayer(g.DataDir + "/audio/sfx/box_off.ogg")
	g.boxOffPadPlayer.SetRolloffFactor(rFactor)

	g.boxFallStartPlayer = createPlayer(g.DataDir + "/audio/sfx/box_fall_start.ogg")
	g.boxFallStartPlayer.SetRolloffFactor(rFactor)

	g.boxFallEndPlayer = createPlayer(g.DataDir + "/audio/sfx/box_fall_end.ogg")
	g.boxFallEndPlayer.SetRolloffFactor(rFactor)

	g.elevatorUpPlayer = createPlayer(g.DataDir + "/audio/sfx/elevator_up.ogg")
	g.elevatorUpPlayer.SetLooping(true)
	g.elevatorUpPlayer.SetRolloffFactor(rFactor)

	g.elevatorDownPlayer = createPlayer(g.DataDir + "/audio/sfx/elevator_down.ogg")
	g.elevatorDownPlayer.SetLooping(true)
	g.elevatorDownPlayer.SetRolloffFactor(rFactor)

	g.levelDonePlayer = createPlayer(g.DataDir + "/audio/sfx/level_done.ogg")
	g.levelDonePlayer.SetRolloffFactor(rFactor)

	g.levelRestartPlayer = createPlayer(g.DataDir + "/audio/sfx/level_restart.ogg")
	g.levelRestartPlayer.SetRolloffFactor(rFactor)

	g.levelFailPlayer = createPlayer(g.DataDir + "/audio/sfx/level_fail.ogg")
	g.levelFailPlayer.SetRolloffFactor(rFactor)

	g.gameCompletePlayer = createPlayer(g.DataDir + "/audio/sfx/game_complete.ogg")
	g.gameCompletePlayer.SetRolloffFactor(rFactor)
}

// LoadSkybox loads the space skybox and adds it to the Scene
func (g *GokobanGame) LoadSkyBox() {
	log.Debug("Creating Skybox...")

	// Load skybox textures
	skyboxData := graphic.SkyboxData{
		g.DataDir + "/img/skybox/", "jpg",
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
		sbmat := skybox.Materials()[i].IMaterial().(*material.Standard)
		sbmat.SetUseLights(material.UseLightNone)
		sbmat.SetEmissiveColor(&math32.Color{brightness, brightness, brightness})
	}
	g.Scene.Add(skybox)

	log.Debug("Done creating skybox")
}

// LoadGopher loads the gopher model and adds to it the sound players associated to it
func (g *GokobanGame) LoadGopher() {
	log.Debug("Decoding gopher model...")

	// Decode model in OBJ format
	dec, err := obj.Decode(g.DataDir + "/gopher/gopher.obj", g.DataDir + "/gopher/gopher.mtl")
	if err != nil {
		panic(err.Error())
	}

	// Create a new node with all the objects in the decoded file and adds it to the Scene
	gopherTop, err := dec.NewGroup()
	if err != nil {
		panic(err.Error())
	}

	g.gopherNode = core.NewNode()
	g.gopherNode.Add(gopherTop)

	log.Debug("Done decoding gopher model")

	// Add gopher-related sound players to gopher node for correct 3D sound positioning
	if g.AudioAvailable {
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
		g.MusicButton.SetImage(gui.ButtonNormal, g.DataDir + "/gui/music_normal.png")
		g.MusicButton.SetImage(gui.ButtonOver, g.DataDir + "/gui/music_hover.png")
		g.MusicButton.SetImage(gui.ButtonPressed, g.DataDir + "/gui/music_click.png")
		g.musicSlider.SetEnabled(true)
		g.musicSlider.SetValue(g.musicSlider.Value())
	} else {
		g.MusicButton.SetImage(gui.ButtonNormal, g.DataDir + "/gui/music_normal_off.png")
		g.MusicButton.SetImage(gui.ButtonOver, g.DataDir + "/gui/music_hover_off.png")
		g.MusicButton.SetImage(gui.ButtonPressed, g.DataDir + "/gui/music_click_off.png")
		g.musicSlider.SetEnabled(false)
		g.SetMusicVolume(0)
	}
}

func (g *GokobanGame) UpdateSfxButton(on bool) {
	if on {
		g.SfxButton.SetImage(gui.ButtonNormal, g.DataDir + "/gui/sound_normal.png")
		g.SfxButton.SetImage(gui.ButtonOver, g.DataDir + "/gui/sound_hover.png")
		g.SfxButton.SetImage(gui.ButtonPressed, g.DataDir + "/gui/sound_click.png")
		g.sfxSlider.SetEnabled(true)
		g.sfxSlider.SetValue(g.sfxSlider.Value())
	} else {
		g.SfxButton.SetImage(gui.ButtonNormal, g.DataDir + "/gui/sound_normal_off.png")
		g.SfxButton.SetImage(gui.ButtonOver, g.DataDir + "/gui/sound_hover_off.png")
		g.SfxButton.SetImage(gui.ButtonPressed, g.DataDir + "/gui/sound_click_off.png")
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
	g.Menu = gui.NewPanel(100, 100)
	g.Menu.SetColor4(&math32.Color4{0.1, 0.1, 0.1, 0.6})
	g.Root.Subscribe(gui.OnResize, func(evname string, ev interface{}) {
		g.Menu.SetWidth(g.Root.ContentWidth())
		g.Menu.SetHeight(g.Root.ContentHeight())
	})

	// Controls
	g.controls = gui.NewPanel(100, 100)
	g.Root.Subscribe(gui.OnResize, func(evname string, ev interface{}) {
		g.controls.SetWidth(g.Root.ContentWidth())
		g.controls.SetHeight(g.Root.ContentHeight())
	})

	// Header panel
	header := gui.NewPanel(0, 0)
	header.SetPosition(0, 0)
	header.SetLayout(gui.NewHBoxLayout())
	header.SetPaddings(20, 20, 20, 20)
	header.SetSize(float32(width), 160)
	g.Root.Subscribe(gui.OnResize, func(evname string, ev interface{}) {
		header.SetWidth(g.Root.ContentWidth())
	})
	g.controls.Add(header)

	// Previous Level Button
	g.prevButton, err = gui.NewImageButton(g.DataDir + "/gui/left_normal.png")
	g.prevButton.SetImage(gui.ButtonOver, g.DataDir + "/gui/left_hover.png")
	g.prevButton.SetImage(gui.ButtonPressed, g.DataDir + "/gui/left_click.png")
	g.prevButton.SetImage(gui.ButtonDisabled, g.DataDir + "/gui/left_disabled2.png")
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
	g.levelLabel, err = gui.NewImageButton(g.DataDir + "/gui/panel.png")
	g.levelLabel.SetImage(gui.ButtonDisabled, g.DataDir + "/gui/panel.png")
	g.levelLabel.SetColor(&math32.Color{0.8, 0.8, 0.8})
	g.levelLabel.SetText("Level")
	g.levelLabel.SetFontSize(35)
	g.levelLabel.SetEnabled(false)
	header.Add(g.levelLabel)

	spacer2 := gui.NewPanel(0, 0)
	spacer2.SetLayoutParams(&params)
	header.Add(spacer2)

	// Next Level Button
	g.nextButton, err = gui.NewImageButton(g.DataDir + "/gui/right_normal.png")
	g.nextButton.SetImage(gui.ButtonOver, g.DataDir + "/gui/right_hover.png")
	g.nextButton.SetImage(gui.ButtonPressed, g.DataDir + "/gui/right_click.png")
	g.nextButton.SetImage(gui.ButtonDisabled, g.DataDir + "/gui/right_disabled2.png")
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
	footer.SetSize(g.Root.ContentHeight(), float32(footer_height))
	g.Root.Subscribe(gui.OnResize, func(evname string, ev interface{}) {
		footer.SetWidth(g.Root.ContentWidth())
		footer.SetPositionY(g.Root.ContentHeight() - float32(footer_height))
	})
	g.controls.Add(footer)

	// Restart Level Button
	g.restartButton, err = gui.NewImageButton(g.DataDir + "/gui/restart_normal.png")
	g.restartButton.SetImage(gui.ButtonOver, g.DataDir + "/gui/restart_hover.png")
	g.restartButton.SetImage(gui.ButtonPressed, g.DataDir + "/gui/restart_click.png")
	g.restartButton.SetImage(gui.ButtonDisabled, g.DataDir + "/gui/restart_disabled2.png")
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
	g.MenuButton, err = gui.NewImageButton(g.DataDir + "/gui/menu_normal.png")
	g.MenuButton.SetImage(gui.ButtonOver, g.DataDir + "/gui/menu_hover.png")
	g.MenuButton.SetImage(gui.ButtonPressed, g.DataDir + "/gui/menu_click.png")
	g.MenuButton.SetImage(gui.ButtonDisabled, g.DataDir + "/gui/menu_disabled2.png")
	if err != nil {
		panic(err)
	}
	g.MenuButton.Subscribe(gui.OnMouseUp, func(evname string, ev interface{}) {
		g.PlaySound(g.clickPlayer, nil)
		g.ToggleMenu()
	})
	g.MenuButton.Subscribe(gui.OnCursorEnter, hoverSound)
	footer.Add(g.MenuButton)

	g.controls.SetVisible(false)
	g.Root.Add(g.controls)

	// Title
	g.TitleImage, err = gui.NewImageButton(g.DataDir + "/gui/title3.png")
	g.TitleImage.SetImage(gui.ButtonDisabled, g.DataDir + "/gui/title3.png")
	g.TitleImage.SetEnabled(false)
	g.Root.Subscribe(gui.OnResize, func(evname string, ev interface{}) {
		g.TitleImage.SetPositionX((g.Root.ContentWidth() - g.TitleImage.ContentWidth()) / 2)
	})
	g.Menu.Add(g.TitleImage)

	// Loading Text
	g.LoadingLabel = gui.NewImageLabel("Loading...")
	g.LoadingLabel.SetColor(&math32.Color{1, 1, 1})
	g.LoadingLabel.SetFontSize(40)
	g.Root.Subscribe(gui.OnResize, func(evname string, ev interface{}) {
		g.LoadingLabel.SetPositionX((g.Root.ContentWidth() - g.LoadingLabel.ContentWidth()) / 2)
		g.LoadingLabel.SetPositionY((g.Root.ContentHeight() - g.LoadingLabel.ContentHeight()) / 2)
	})
	g.Root.Add(g.LoadingLabel)

	// Instructions
	g.instructions1 = gui.NewImageLabel(INSTRUCTIONS_LINE1)
	g.instructions1.SetColor(&creditsColor)
	g.instructions1.SetFontSize(28)
	g.Root.Subscribe(gui.OnResize, func(evname string, ev interface{}) {
		g.instructions1.SetWidth(g.Root.ContentWidth())
		g.instructions1.SetPositionY(4 * g.instructions1.ContentHeight())
	})
	g.controls.Add(g.instructions1)

	g.instructions2 = gui.NewImageLabel(INSTRUCTIONS_LINE2)
	g.instructions2.SetColor(&creditsColor)
	g.instructions2.SetFontSize(28)
	g.Root.Subscribe(gui.OnResize, func(evname string, ev interface{}) {
		g.instructions2.SetWidth(g.Root.ContentWidth())
		g.instructions2.SetPositionY(5 * g.instructions2.ContentHeight())
	})
	g.controls.Add(g.instructions2)

	g.instructions3 = gui.NewImageLabel(INSTRUCTIONS_LINE3)
	g.instructions3.SetColor(&creditsColor)
	g.instructions3.SetFontSize(28)
	g.Root.Subscribe(gui.OnResize, func(evname string, ev interface{}) {
		g.instructions3.SetWidth(g.Root.ContentWidth())
		g.instructions3.SetPositionY(g.Root.ContentHeight() - 2*g.instructions3.ContentHeight())
	})
	g.controls.Add(g.instructions3)

	buttonInstructionsPad := float32(24)

	g.instructionsRestart = gui.NewImageLabel("Restart Level (R)")
	g.instructionsRestart.SetColor(&creditsColor)
	g.instructionsRestart.SetFontSize(20)
	g.Root.Subscribe(gui.OnResize, func(evname string, ev interface{}) {
		g.instructionsRestart.SetPosition(buttonInstructionsPad, g.Root.ContentHeight()-6*g.instructionsRestart.ContentHeight())
	})
	g.controls.Add(g.instructionsRestart)

	g.instructionsMenu = gui.NewImageLabel("Show Menu (Esc)")
	g.instructionsMenu.SetColor(&creditsColor)
	g.instructionsMenu.SetFontSize(20)
	g.Root.Subscribe(gui.OnResize, func(evname string, ev interface{}) {
		g.instructionsMenu.SetPosition(g.Root.ContentWidth()-g.instructionsMenu.ContentWidth()-buttonInstructionsPad, g.Root.ContentHeight()-6*g.instructionsMenu.ContentHeight())
	})
	g.controls.Add(g.instructionsMenu)

	// Main panel
	g.Main = gui.NewPanel(600, 300)
	mainLayout := gui.NewVBoxLayout()
	mainLayout.SetAlignV(gui.AlignHeight)
	g.Main.SetLayout(mainLayout)
	g.Main.SetBorders(2, 2, 2, 2)
	g.Main.SetBordersColor4(&sliderBorderColor)
	g.Main.SetColor4(&math32.Color4{0.2, 0.2, 0.2, 0.6})
	g.Root.Subscribe(gui.OnResize, func(evname string, ev interface{}) {
		g.Main.SetPositionX((g.Root.Width() - g.Main.Width()) / 2)
		g.Main.SetPositionY((g.Root.Height()-g.Main.Height())/2 + 50)
	})

	topRow := gui.NewPanel(g.Main.ContentWidth(), 100)
	topRowLayout := gui.NewHBoxLayout()
	topRowLayout.SetAlignH(gui.AlignWidth)
	topRow.SetLayout(topRowLayout)
	alignCenterVerical := gui.HBoxLayoutParams{Expand: 0, AlignV: gui.AlignCenter}

	// Music Control
	musicControl := gui.NewPanel(130, 100)
	musicControl.SetLayout(topRowLayout)

	g.MusicButton, err = gui.NewImageButton(g.DataDir + "/gui/music_normal.png")
	g.MusicButton.SetImage(gui.ButtonOver, g.DataDir + "/gui/music_hover.png")
	g.MusicButton.SetImage(gui.ButtonPressed, g.DataDir + "/gui/music_click.png")
	g.MusicButton.SetImage(gui.ButtonDisabled, g.DataDir + "/gui/music_disabled2.png")
	if err != nil {
		panic(err)
	}
	g.MusicButton.Subscribe(gui.OnMouseUp, func(evname string, ev interface{}) {
		g.PlaySound(g.clickPlayer, nil)
		g.UserData.MusicOn = !g.UserData.MusicOn
		g.UpdateMusicButton(g.UserData.MusicOn)
	})
	g.MusicButton.Subscribe(gui.OnCursorEnter, hoverSound)
	musicControl.Add(g.MusicButton)

	// Music Volume Slider
	g.musicSlider = gui.NewVSlider(20, 80)
	g.musicSlider.SetValue(g.UserData.MusicVol)
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

	g.SfxButton, err = gui.NewImageButton(g.DataDir + "/gui/sound_normal.png")
	g.SfxButton.SetImage(gui.ButtonOver, g.DataDir + "/gui/sound_hover.png")
	g.SfxButton.SetImage(gui.ButtonPressed, g.DataDir + "/gui/sound_click.png")
	g.SfxButton.SetImage(gui.ButtonDisabled, g.DataDir + "/gui/sound_disabled2.png")
	if err != nil {
		panic(err)
	}
	g.SfxButton.Subscribe(gui.OnMouseUp, func(evname string, ev interface{}) {
		g.PlaySound(g.clickPlayer, nil)
		g.UserData.SfxOn = !g.UserData.SfxOn
		g.UpdateSfxButton(g.UserData.SfxOn)
	})
	g.SfxButton.Subscribe(gui.OnCursorEnter, hoverSound)
	sfxControl.Add(g.SfxButton)

	// Sound Effects Volume Slider
	g.sfxSlider = gui.NewVSlider(20, 80)
	g.sfxSlider.SetValue(g.UserData.SfxVol)
	g.sfxSlider.Subscribe(gui.OnChange, func(evname string, ev interface{}) {
		g.SetSfxVolume(3 * g.sfxSlider.Value())
	})
	g.sfxSlider.Subscribe(gui.OnCursorEnter, hoverSound)
	g.sfxSlider.SetLayoutParams(&alignCenterVerical)
	sfxControl.Add(g.sfxSlider)

	topRow.Add(sfxControl)

	// FullScreen Button
	g.fullScreenButton, err = gui.NewImageButton(g.DataDir + "/gui/screen_normal.png")
	g.fullScreenButton.SetImage(gui.ButtonOver, g.DataDir + "/gui/screen_hover.png")
	g.fullScreenButton.SetImage(gui.ButtonPressed, g.DataDir + "/gui/screen_click.png")
	g.fullScreenButton.SetImage(gui.ButtonDisabled, g.DataDir + "/gui/screen_disabled2.png")
	if err != nil {
		panic(err)
	}
	g.fullScreenButton.Subscribe(gui.OnMouseUp, func(evname string, ev interface{}) {
		g.PlaySound(g.clickPlayer, nil)
		g.ToggleFullScreen()
	})
	g.fullScreenButton.Subscribe(gui.OnCursorEnter, hoverSound)
	topRow.Add(g.fullScreenButton)

	g.Main.Add(topRow)

	buttonRow := gui.NewPanel(g.Main.ContentWidth(), 100)
	buttonRowLayout := gui.NewHBoxLayout()
	buttonRowLayout.SetAlignH(gui.AlignWidth)
	buttonRow.SetLayout(buttonRowLayout)

	// Quit Button
	g.quitButton, err = gui.NewImageButton(g.DataDir + "/gui/quit_normal.png")
	g.quitButton.SetImage(gui.ButtonOver, g.DataDir + "/gui/quit_hover.png")
	g.quitButton.SetImage(gui.ButtonPressed, g.DataDir + "/gui/quit_click.png")
	g.quitButton.SetImage(gui.ButtonDisabled, g.DataDir + "/gui/quit_disabled2.png")
	if err != nil {
		panic(err)
	}
	g.quitButton.Subscribe(gui.OnMouseUp, func(evname string, ev interface{}) {
		g.Quit()
	})
	g.quitButton.Subscribe(gui.OnCursorEnter, hoverSound)
	buttonRow.Add(g.quitButton)

	// Play Button
	g.playButton, err = gui.NewImageButton(g.DataDir + "/gui/play_normal.png")
	g.playButton.SetImage(gui.ButtonOver, g.DataDir + "/gui/play_hover.png")
	g.playButton.SetImage(gui.ButtonPressed, g.DataDir + "/gui/play_click.png")
	g.playButton.SetImage(gui.ButtonDisabled, g.DataDir + "/gui/play_disabled2.png")
	if err != nil {
		panic(err)
	}
	g.playButton.Subscribe(gui.OnMouseUp, func(evname string, ev interface{}) {
		g.PlaySound(g.clickPlayer, nil)
		g.ToggleMenu()
	})
	g.playButton.Subscribe(gui.OnCursorEnter, hoverSound)
	buttonRow.Add(g.playButton)

	g.Main.Add(buttonRow)

	// Add credits labels
	lCredits1 := gui.NewImageLabel(CREDITS_LINE1)
	lCredits1.SetColor(&creditsColor)
	lCredits1.SetFontSize(20)
	g.Root.Subscribe(gui.OnResize, func(evname string, ev interface{}) {
		lCredits1.SetWidth(g.Root.ContentWidth())
		lCredits1.SetPositionY(g.Root.ContentHeight() - 2*lCredits1.ContentHeight())
	})
	g.Menu.Add(lCredits1)

	lCredits2 := gui.NewImageLabel(CREDITS_LINE2)
	lCredits2.SetColor(&creditsColor)
	lCredits2.SetFontSize(20)
	g.Root.Subscribe(gui.OnResize, func(evname string, ev interface{}) {
		lCredits2.SetWidth(g.Root.ContentWidth())
		lCredits2.SetPositionY(g.Root.ContentHeight() - lCredits2.ContentHeight())
	})
	g.Menu.Add(lCredits2)

	g3n := gui.NewImageLabel("")
	g3n.SetSize(57, 50)
	g3n.SetImageFromFile(g.DataDir + "/img/g3n.png")
	g.Root.Subscribe(gui.OnResize, func(evname string, ev interface{}) {
		g3n.SetPositionX(g.Root.ContentWidth() - g3n.Width())
		g3n.SetPositionY(g.Root.ContentHeight() - 1.3*g3n.Height())
	})
	g.Menu.Add(g3n)

	g.Root.Add(g.Menu)

	// Dispatch a fake OnResize event to update all subscribed elements
	g.Root.Dispatch(gui.OnResize, nil)

	log.Debug("Done creating GUI.")
}

// PlaySound attaches the specified player to the specified node and plays the sound
func (g *GokobanGame) PlaySound(player *audio.Player, node *core.Node) {
	if g.AudioAvailable {
		if node != nil {
			node.Add(player)
		}
		player.Stop()
		player.Play()
	}
}

// LoadAudioLibs
func LoadAudioLibs() error {

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


// RenderFrame renders a frame of the Scene with the GUI overlaid
func (g *GokobanGame) RenderFrame() {

	// Process GUI timers
	g.Root.TimerManager.ProcessTimers()

	// Render the Scene/gui using the specified camera
	rendered, err := g.Renderer.Render(g.Camera)
	if err != nil {
		panic(err)
	}

	// Check I/O events
	g.Wmgr.PollEvents()

	// Update Window if necessary
	if rendered {
		g.Win.SwapBuffers()
	}
}
