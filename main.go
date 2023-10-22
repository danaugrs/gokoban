// Copyright 2017 Daniel Salvadori. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"github.com/g3n/engine/app"
	"github.com/g3n/engine/audio"
	"github.com/g3n/engine/camera"
	"github.com/g3n/engine/core"
	"github.com/g3n/engine/geometry"
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
	"io/ioutil"
	"strconv"
	"time"
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

type Gokoban struct {
	*app.Application

	scene  *core.Node
	camera *camera.Camera
	orbit  *camera.OrbitControl

	userData *UserData

	levelScene *core.Node
	levelStyle *LevelStyle
	levels     []*Level
	level      *Level
	leveln     int

	stepDelta    *math32.Vector2
	gopherLocked bool
	gopherNode   *core.Node
	arrowNode    *core.Node
	steps        int

	// User interface
	ui *UI

	// Sounds and music
	audio *Audio
}

// RestartLevel restarts the current level
func (g *Gokoban) RestartLevel(playSound bool) {
	log.Debug("Restart Level")

	firstLevel := g.leveln == 0

	if firstLevel {
		g.ui.instructions3.SetText(INSTRUCTIONS_LINE3)
	}

	g.ui.instructions1.SetVisible(firstLevel)
	g.ui.instructions2.SetVisible(firstLevel)
	g.ui.instructions3.SetVisible(firstLevel)
	g.ui.instructionsRestart.SetVisible(firstLevel)
	g.ui.instructionsMenu.SetVisible(firstLevel)
	g.arrowNode.SetVisible(firstLevel)

	g.levels[g.leveln].Restart(playSound)
	g.gopherLocked = false
}

// NextLevel loads the next level if exists
func (g *Gokoban) NextLevel() {
	log.Debug("Next Level")

	if g.leveln < len(g.levels)-1 {
		g.InitLevel(g.leveln + 1)
	}
}

// PreviousLevel loads the previous level if exists
func (g *Gokoban) PreviousLevel() {
	log.Debug("Previous Level")

	if g.leveln > 0 {
		g.InitLevel(g.leveln - 1)
	}
}

// ToggleFullScreen toggles whether is game is fullscreen or windowed
func (g *Gokoban) ToggleFullScreen() {
	log.Debug("Toggle FullScreen")

	g.IWindow.(*window.GlfwWindow).SetFullScreen(!g.IWindow.(*window.GlfwWindow).FullScreen())
}

// Quit saves the user data and quits the game
func (g *Gokoban) Quit() {
	log.Debug("Quit")

	// Copy settings into user data and save
	g.userData.SfxVol = g.ui.sfxSlider.Value()
	g.userData.MusicVol = g.ui.musicSlider.Value()
	g.userData.FullScreen = g.IWindow.(*window.GlfwWindow).FullScreen()
	g.userData.Save()

	// Close the window
	g.Exit()
}

// onKey handles keyboard events for the game
func (g *Gokoban) onKey(evname string, ev interface{}) {

	kev := ev.(*window.KeyEvent)
	switch kev.Key {
	case window.KeyEscape:
		g.ui.ToggleMenu()
	case window.KeyF:
		g.ToggleFullScreen()
	case window.KeyR:
		if !g.ui.inMenu && g.steps > 0 {
			g.RestartLevel(true)
		}
	}
}

// onMouse handles mouse events for the game
func (g *Gokoban) onMouse(evname string, ev interface{}) {
	mev := ev.(*window.MouseEvent)

	if !g.gopherLocked && g.leveln > 0 {
		// Mouse button pressed
		if evname == window.OnMouseDown {
			// Left button pressed
			if mev.Button == window.MouseButtonLeft {
				g.arrowNode.SetVisible(true)
			}
		} else if evname == window.OnMouseUp {
			g.arrowNode.SetVisible(false)
		}
	}
}

// onCursor handles cursor movement for the game
func (g *Gokoban) onCursor(evname string, ev interface{}) {

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

// LevelComplete updates and saves user data, enables the next button if appropriate, and checks for game completion
func (g *Gokoban) LevelComplete() {
	log.Debug("Level Complete")

	if g.leveln == 0 {
		g.ui.instructions3.SetText(INSTRUCTIONS_LINE3_COMPLETE)
	}

	if g.userData.LastUnlockedLevel == g.leveln {
		g.userData.LastUnlockedLevel++
		g.userData.Save()
		if g.userData.LastUnlockedLevel < len(g.levels) {
			g.ui.nextButton.SetEnabled(true)
		}
		if g.userData.LastUnlockedLevel == len(g.levels) {
			g.GameCompleted()
		}
	}
}

// GameCompleted stops the music, plays the the winning sound, and changes the title image to say "Completed"
func (g *Gokoban) GameCompleted() {
	log.Debug("Game Completed")

	g.audio.musicGame.Stop()
	g.audio.gameComplete.Play()
	g.ui.titleImage.SetImage(gui.ButtonDisabled, "./gui/title3_completed.png")
}

// InitLevel initializes the level associated to the provided index
func (g *Gokoban) InitLevel(n int) {
	log.Debug("Initializing Level %v", n+1)

	// Always enable the button to return to the previous level except when we are in the very first level
	g.ui.prevButton.SetEnabled(n != 0)

	// The button to go to the next level has 3 different states: disabled, locked and enabled
	// If this is the very last level - disable it completely
	if n == len(g.levels)-1 {
		g.ui.nextButton.SetImage(gui.ButtonDisabled, "./gui/right_disabled2.png")
		g.ui.nextButton.SetEnabled(false)
	} else {
		g.ui.nextButton.SetImage(gui.ButtonDisabled, "./gui/right_disabled_locked.png")
		// check last completed level
		if g.userData.LastUnlockedLevel == n {
			g.ui.nextButton.SetEnabled(false)
		} else {
			g.ui.nextButton.SetEnabled(true)
		}
	}

	// Remove level.scene from levelScene and unsubscribe from events
	if len(g.levelScene.Children()) > 0 {
		g.levelScene.Remove(g.level.scene)
		g.UnsubscribeID(window.OnKeyDown, g.leveln)
	}

	// Update current level index and level reference
	g.leveln = n
	g.userData.LastLevel = n
	g.level = g.levels[g.leveln]

	g.RestartLevel(false)
	g.level.gopherNodeRotate.Add(g.gopherNode)
	g.level.gopherNodeTranslate.Add(g.arrowNode)

	// Update level text and resize GUI
	g.ui.levelLabelText.SetText("Level " + strconv.Itoa(n+1))
	width, height := g.GetFramebufferSize()
	g.ui.Resize(width, height)

	g.levelScene.Add(g.level.scene)
	g.SubscribeID(window.OnKeyDown, g.leveln, g.level.onKey)
}

// LoadLevels reads and parses the level files inside ./levels, building an array of Level objects
func (g *Gokoban) LoadLevels() {
	log.Debug("Load Levels")

	files, _ := ioutil.ReadDir("./levels")
	g.levels = make([]*Level, len(files)-1)

	for i, f := range files {

		// Skip README.md
		if f.Name() == "README.md" {
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
		g.levels[i] = NewLevel(g, ld, g.levelStyle)
	}
}

// LoadSkybox loads the space skybox and adds it to the scene
func (g *Gokoban) LoadSkyBox() {
	log.Debug("Creating Skybox...")

	// Load skybox textures
	skyboxData := graphic.SkyboxData{"./img/skybox/", "jpg", [6]string{"px", "nx", "py", "ny", "pz", "nz"}}
	skybox, err := graphic.NewSkybox(skyboxData)
	if err != nil {
		panic(err)
	}
	g.scene.Add(skybox)

	log.Debug("Done creating skybox")
}

// LoadGopher loads the gopher model and adds to it the sound players associated to it
func (g *Gokoban) LoadGopher() {
	log.Debug("Decoding gopher model...")

	// Decode model in OBJ format
	dec, err := obj.Decode("./gopher/gopher.obj", "./gopher/gopher.mtl")
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
	g.gopherNode.Add(g.audio.gopherWalk)
	g.gopherNode.Add(g.audio.gopherBump)
	g.gopherNode.Add(g.audio.gopherHurt)
	g.gopherNode.Add(g.audio.gopherFallStart)
	g.gopherNode.Add(g.audio.gopherFallEnd)
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
func (g *Gokoban) CreateArrowNode() {

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

// OnWindowResize is default handler for window resize events.
func (g *Gokoban) OnWindowResize() {

	// Get framebuffer size and set the viewport accordingly
	width, height := g.GetFramebufferSize()
	g.Gls().Viewport(0, 0, int32(width), int32(height))

	// Set camera aspect ratio
	g.camera.SetAspect(float32(width) / float32(height))

	// Update UI
	g.ui.Resize(width, height)
}

func main() {

	// Parse command line flags
	oDebug := flag.Bool("debug", false, "display the debug log and check OpenGL errors")
	flag.Parse()

	// Create logger
	log = logger.New("Gokoban", nil)
	log.AddWriter(logger.NewConsole(false))
	log.SetFormat(logger.FTIME | logger.FMICROS)
	if *oDebug == true {
		log.SetLevel(logger.DEBUG)
	} else {
		log.SetLevel(logger.INFO)
	}
	log.Info("Initializing Gokoban")

	// Create Gokoban instance and initialize the G3N application
	g := new(Gokoban)
	g.Application = app.App(1280, 920, "Gokoban")	

	// Log OpenGL version
	log.Debug("OpenGL version: %s", g.Gls().GetString(gls.VERSION))

	// Speed up a bit by not checking OpenGL errors (only if not debugging)
	if *oDebug == false {
		g.Gls().SetCheckErrors(false)
	}

	// Load or create user data file
	g.userData = LoadOrCreateUserData()

	// Change to full screen if user prefers it
	g.IWindow.(*window.GlfwWindow).SetFullScreen(g.userData.FullScreen)

	// Create main scene and child levelScene
	g.scene = core.NewNode()
	g.levelScene = core.NewNode()
	g.scene.Add(g.levelScene)
	
	// Set the scene to be managed by the gui manager
	gui.Manager().Set(g.scene)

	// Create camera
	width, height := g.GetFramebufferSize()
	g.Gls().Viewport(0, 0, int32(width), int32(height))
	aspect := float32(width) / float32(height)
	g.camera = camera.New(aspect)
	g.camera.SetPosition(0, 4, 5)
	g.camera.LookAt(&math32.Vector3{0, 0, 0}, &math32.Vector3{0, 1, 0})
	g.scene.Add(g.camera) // Add camera to scene

	// Create orbit control and set limits
	g.orbit = camera.NewOrbitControl(g.camera)
	g.orbit.SetEnabled(camera.OrbitNone)
	g.orbit.MaxPolarAngle = 2 * math32.Pi / 3
	g.orbit.MinDistance = 5
	g.orbit.MaxDistance = 15

	// Add white ambient light to the scene
	ambLight := light.NewAmbient(&math32.Color{1.0, 1.0, 1.0}, 0.4)
	g.scene.Add(ambLight)

	// Create initial user interface and show loading label
	g.ui = NewUI(g)
	g.scene.Add(g.ui)

	// Render frame to show game title and loading label
	g.Gls().Clear(gls.COLOR_BUFFER_BIT | gls.DEPTH_BUFFER_BIT | gls.STENCIL_BUFFER_BIT)
	g.Renderer().Render(g.scene, g.camera)
	g.IWindow.(*window.GlfwWindow).SwapBuffers()
	log.Info("Render Frame")

	// Initialize the rest of the UI
	g.ui.Init()

	// Set up sounds and music
	g.audio = NewAudio()

	// Create audio listener and add it to the current camera
	listener := audio.NewListener()
	cdir := g.camera.Direction()
	listener.SetDirectionVec(&cdir)
	g.camera.Add(listener)

	// Update settings based on loaded (or newly created) user data
	g.ui.UpdateMusicButton(g.userData.MusicOn)
	g.ui.UpdateSfxButton(g.userData.SfxOn)

	// Start the music!
	g.audio.musicMenu.Play()

	// Initialize step delta
	g.stepDelta = math32.NewVector2(0, 0)

	// Create level style
	g.levelStyle = NewStandardStyle()

	// Load skybox, gopher, create arrow node (above gopher's head)
	g.LoadSkyBox()
	g.LoadGopher()
	g.CreateArrowNode()

	// Load all levels
	g.LoadLevels()

	// Subscribe window to events
	g.Subscribe(window.OnKeyDown, g.onKey)
	g.Subscribe(window.OnMouseUp, g.onMouse)
	g.Subscribe(window.OnMouseDown, g.onMouse)
	g.Subscribe(window.OnCursor, g.onCursor)
	g.Subscribe(window.OnWindowSize, func(evname string, ev interface{}) { g.OnWindowResize() })

	// Check if user already completed all levels
	if g.userData.LastUnlockedLevel == len(g.levels) {
		g.ui.titleImage.SetImage(gui.ButtonDisabled, "./gui/title3_completed.png")
	}

	// Trigger window resize to recompute UI
	g.OnWindowResize()

	// Done Loading - hide the loading label, show the menu, and initialize the level
	g.ui.loadingLabel.SetVisible(false)
	g.InitLevel(g.userData.LastLevel)
	g.gopherLocked = true

	// Start the render loop
	log.Info("Starting Render Loop")
	g.Application.Run(g.Update)
}

// Update wlll called every frame
func (g *Gokoban) Update(rend *renderer.Renderer, deltaTime time.Duration) {

	// Update the current level if any
	if g.level != nil {
		g.level.Update(deltaTime.Seconds())
	}

	// Clear the color, depth, and stencil buffers
	g.Gls().Clear(gls.COLOR_BUFFER_BIT | gls.DEPTH_BUFFER_BIT | gls.STENCIL_BUFFER_BIT)

	// Render scene
	err := rend.Render(g.scene, g.camera)
	if err != nil {
		panic(err)
	}

	// Update GUI timers
	gui.Manager().TimerManager.ProcessTimers()
}
