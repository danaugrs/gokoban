// Copyright 2017 Daniel Salvadori. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"github.com/g3n/engine/camera"
	"github.com/g3n/engine/camera/control"
	"github.com/g3n/engine/core"
	"github.com/g3n/engine/gls"
	"github.com/g3n/engine/gui"
	"github.com/g3n/engine/light"
	"github.com/g3n/engine/math32"
	"github.com/g3n/engine/renderer"
	"github.com/g3n/engine/util/logger"
	"github.com/g3n/engine/window"

	"flag"
	"runtime"
	"time"
	"os"
//	"strings"
//	"path/filepath"
	"github.com/Nv7-Github/gokoban/components"
)

var log *logger.Logger

//      ____       _         _
//     / ___| ___ | | _____ | |__   __ _ _ __
//    | |  _ / _ \| |/ / _ \| '_ \ / _` | '_ \
//    | |_| | (_) |   < (_) | |_) | (_| | | | |
//     \____|\___/|_|\_\___/|_.__/ \__,_|_| |_|
//

func main() {
	// OpenGL functions must be executed in the same thread where
	// the context was created (by window.New())
	runtime.LockOSThread()

	// Parse command line flags
	showLog := flag.Bool("debug", false, "display the debug log")
	flag.Parse()

	// Create logger
	log = logger.New("Gokoban", nil)
	components.InitLogger(log)

	log.AddWriter(logger.NewConsole(false))
	log.SetFormat(logger.FTIME | logger.FMICROS)
	if *showLog == true {
		log.SetLevel(logger.DEBUG)
	} else {
		log.SetLevel(logger.INFO)
	}
	log.Info("Initializing Gokoban")

	// Create GokobanGame struct
	g := new(components.GokobanGame)

	// Manually scan the $GOPATH directories to find the data directory
	//rawPaths := os.Getenv("GOPATH")
// 	paths := strings.Split(rawPaths, ":")
// 	for _, j := range paths {
// 		// Checks data path
// //		path := filepath.Join(j, "src", "github.com", "danaugrs", "gokoban")
// 	}
	path := "."
	if _, err := os.Stat(path); err == nil {
		g.DataDir = path
	}

	// Load user data from file
	g.UserData = components.NewUserData(g.DataDir)

	// Get the window manager
	var err error
	g.Wmgr, err = window.Manager("glfw")
	if err != nil {
		panic(err)
	}

	// Create window and OpenGL context
	g.Win, err = g.Wmgr.CreateWindow(1200, 900, "Gokoban", g.UserData.FullScreen)
	if err != nil {
		panic(err)
	}

	// Create OpenGL state
	g.Gs, err = gls.New()
	if err != nil {
		panic(err)
	}

	// Speed up a bit by not checking OpenGL errors
	g.Gs.SetCheckErrors(false)

	// Sets window background color
	g.Gs.ClearColor(0.1, 0.1, 0.1, 1.0)

	// Sets the OpenGL viewport size the same as the window size
	// This normally should be updated if the window is resized.
	width, height := g.Win.Size()
	g.Gs.Viewport(0, 0, int32(width), int32(height))

	// Creates GUI Root panel
	g.Root = gui.NewRoot(g.Gs, g.Win)
	g.Root.SetSize(float32(width), float32(height))

	// Subscribe to window resize events. When the window is resized:
	// - Update the viewport size
	// - Update the Root panel size
	// - Update the camera aspect ratio
	g.Win.Subscribe(window.OnWindowSize, func(evname string, ev interface{}) {
		width, height := g.Win.Size()
		g.Gs.Viewport(0, 0, int32(width), int32(height))
		g.Root.SetSize(float32(width), float32(height))
		aspect := float32(width) / float32(height)
		g.Camera.SetAspect(aspect)
	})

	// Subscribe window to events
	g.Win.Subscribe(window.OnKeyDown, g.OnKey)
	g.Win.Subscribe(window.OnMouseUp, g.OnMouse)
	g.Win.Subscribe(window.OnMouseDown, g.OnMouse)

	// Creates a renderer and adds default shaders
	g.Renderer = renderer.NewRenderer(g.Gs)
	//g.Renderer.SetSortObjects(false)
	err = g.Renderer.AddDefaultShaders()
	if err != nil {
		panic(err)
	}
	g.Renderer.SetGui(g.Root)

	// Adds a perspective camera to the scene
	// The camera aspect ratio should be updated if the window is resized.
	aspect := float32(width) / float32(height)
	g.Camera = camera.NewPerspective(65, aspect, 0.01, 1000)
	g.Camera.SetPosition(0, 4, 5)
	g.Camera.LookAt(&math32.Vector3{0, 0, 0})

	// Create orbit control and set limits
	g.OrbitControl = control.NewOrbitControl(g.Camera, g.Win)
	g.OrbitControl.Enabled = false
	g.OrbitControl.EnablePan = false
	g.OrbitControl.MaxPolarAngle = 2 * math32.Pi / 3
	g.OrbitControl.MinDistance = 5
	g.OrbitControl.MaxDistance = 15

	// Create main scene and child LevelScene
	g.Scene = core.NewNode()
	g.LevelScene = core.NewNode()
	g.Scene.Add(g.Camera)
	g.Scene.Add(g.LevelScene)
	g.StepDelta = math32.NewVector2(0, 0)
	g.Renderer.SetScene(g.Scene)

	// Add white ambient light to the Scene
	ambLight := light.NewAmbient(&math32.Color{1.0, 1.0, 1.0}, 0.4)
	g.Scene.Add(ambLight)

	g.LevelStyle = components.NewStandardStyle(g.DataDir)

	g.SetupGui(width, height)
	g.RenderFrame()

	// Try to open audio libraries
	err = components.LoadAudioLibs()
	if err != nil {
		log.Error("%s", err)
		g.UpdateMusicButton(false)
		g.UpdateSfxButton(false)
		g.MusicButton.SetEnabled(false)
		g.SfxButton.SetEnabled(false)
	} else {
		g.AudioAvailable = true
		g.LoadAudio()
		g.UpdateMusicButton(g.UserData.MusicOn)
		g.UpdateSfxButton(g.UserData.SfxOn)

		// Queue the music!
		g.MusicPlayerMenu.Play()
	}

	g.LoadSkyBox()
	g.LoadGopher()
	g.CreateArrowNode()
	g.LoadLevels()

	g.Win.Subscribe(window.OnCursor, g.OnCursor)

	if g.UserData.LastUnlockedLevel == len(g.Levels) {
		g.TitleImage.SetImage(gui.ButtonDisabled, g.DataDir + "/gui/title3_completed.png")
	}

	// Done Loading - hide the loading label, show the Menu, and initialize the level
	g.LoadingLabel.SetVisible(false)
	g.Menu.Add(g.Main)
	g.InitLevel(g.UserData.LastLevel)
	g.GopherLocked = true

	now := time.Now()
	newNow := time.Now()
	log.Info("Starting Render Loop")

	// Start the render loop
	for !g.Win.ShouldClose() {

		newNow = time.Now()
		timeDelta := now.Sub(newNow)
		now = newNow

		g.Update(timeDelta.Seconds())
		g.RenderFrame()
	}
}
