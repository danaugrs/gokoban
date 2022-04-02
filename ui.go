// Copyright 2017 Daniel Salvadori. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"github.com/g3n/engine/camera"
	"github.com/g3n/engine/gui"
	"github.com/g3n/engine/math32"
	"github.com/g3n/engine/window"
)

var creditsColor = math32.Color{0.6, 0.6, 0.6}
var transparent = math32.Color4{0, 0, 0, 0}
var blackTextColor = math32.Color4{0.3, 0.3, 0.3, 1.0}
var sliderColor = math32.Color4{0.628, 0.882, 0.1, 1}
var sliderColorOff = math32.Color4{0.82, 0.48, 0.48, 1}
var sliderColorOver = math32.Color4{0.728, 0.982, 0.2, 1}
var sliderBorderColor = math32.Color4{0.71, 0.482, 0.26, 1}
var sliderBorder = gui.RectBounds{3, 3, 3, 3}
var gameScreenPadding float32 = 20.0

type UI struct {
	gui.Panel
	game *Gokoban

	inMenu bool

	// Main menu
	menuScreen       *gui.Panel
	titleImage       *gui.ImageButton
	loadingLabel     *gui.Label
	menuPanel        *gui.Panel
	quitButton       *gui.ImageButton
	playButton       *gui.ImageButton
	sfxButton        *gui.ImageButton
	sfxSlider        *gui.Slider
	musicButton      *gui.ImageButton
	musicSlider      *gui.Slider
	fullScreenButton *gui.ImageButton

	// In-game controls and HUD
	gameScreen          *gui.Panel
	gameScreenHeader    *gui.Panel
	gameScreenFooter    *gui.Panel
	levelLabelImage     *gui.ImageLabel
	levelLabelText      *gui.Label
	nextButton          *gui.ImageButton
	prevButton          *gui.ImageButton
	restartButton       *gui.ImageButton
	menuButton          *gui.ImageButton
	instructions1       *gui.ImageLabel
	instructions2       *gui.ImageLabel
	instructions3       *gui.ImageLabel
	instructionsRestart *gui.ImageLabel
	instructionsMenu    *gui.ImageLabel
}

// NewUI creates a ui panel with a loading label and title
func NewUI(game *Gokoban) *UI {
	ui := new(UI)
	ui.game = game

	var err error

	ui.Panel.Initialize(ui, 1280, 920)
	ui.SetEnabled(false)
	ui.inMenu = true

	ui.loadingLabel = gui.NewLabel("Loading...")
	ui.loadingLabel.SetColor(&math32.Color{1, 1, 1})
	ui.loadingLabel.SetFontSize(42)
	ui.loadingLabel.SetPositionX(((ui.ContentWidth() - ui.loadingLabel.ContentWidth()) / 2)+ 0.5)
	ui.loadingLabel.SetPositionY(((ui.ContentHeight() - ui.loadingLabel.ContentHeight()) / 2)+ 0.5)
	ui.Add(ui.loadingLabel)

	// Title
	ui.titleImage, err = gui.NewImageButton("./gui/title3.png")
	if err != nil {
		panic(err)
	}
	ui.titleImage.SetImage(gui.ButtonDisabled, "./gui/title3.png")
	ui.titleImage.SetEnabled(false)
	ui.titleImage.SetZLayerDelta(1)
	ui.titleImage.SetPositionX((ui.ContentWidth() - ui.titleImage.ContentWidth()) / 2)
	ui.Add(ui.titleImage)

	return ui
}

// Init sets up the styling and creates the menu and game screens
func (ui *UI) Init() {

	// Modify ImageButton style
	s := gui.StyleDefault()
	s.ImageButton = gui.ImageButtonStyles{}
	s.ImageButton.Normal = gui.ImageButtonStyle{}
	s.ImageButton.Normal.BgColor = transparent
	s.ImageButton.Normal.FgColor = blackTextColor
	s.ImageButton.Over = s.ImageButton.Normal
	s.ImageButton.Focus = s.ImageButton.Normal
	s.ImageButton.Pressed = s.ImageButton.Normal
	s.ImageButton.Disabled = s.ImageButton.Normal

	// Modify Slider style
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

	// Create menu and game screens
	ui.CreateMenuScreen()
	ui.CreateGameScreen()

	// Show menu screen
	ui.Add(ui.menuScreen)

	// Move title image to menu
	ui.menuScreen.Add(ui.titleImage)
}

// Resize updates the UI elements based on the provided screen dimensions
func (ui *UI) Resize(width, height int) {

	// Update screen container panels
	ui.SetSize(float32(width), float32(height))
	ui.titleImage.SetPositionX((float32(width) - ui.titleImage.ContentWidth()) / 2)
	ui.menuScreen.SetSize(float32(width), float32(height))
	ui.gameScreen.SetSize(float32(width), float32(height))

	// Note: below we add a half-pixel in several places to prevent a visual bleeding artifact

	// Menu Screen
	ui.menuPanel.SetPositionX(math32.Round((float32(width)-ui.menuPanel.Width())/2) + 0.5)
	ui.menuPanel.SetPositionY(math32.Round((float32(height)-ui.menuPanel.Height())/1.6) + 0.5)

	// Game Screen
	// Note: for some reason calling SetPosition instead of SetPositionX and SetPositionY (separately) results in the same visual bleeding artifact
	ui.prevButton.SetPositionX(math32.Round(gameScreenPadding) + 0.5)
	ui.prevButton.SetPositionY(math32.Round(gameScreenPadding) + 0.5)
	ui.levelLabelImage.SetPositionX(math32.Round((float32(width)-ui.levelLabelImage.ContentWidth())/2) + 0.5)
	ui.levelLabelText.SetPositionX(math32.Round((float32(width) - ui.levelLabelText.ContentWidth()) / 2))
	ui.nextButton.SetPositionX(math32.Round(float32(width)-ui.prevButton.ContentWidth()-gameScreenPadding) + 0.5)
	ui.restartButton.SetPositionY(math32.Round(float32(height)-ui.restartButton.ContentHeight()-gameScreenPadding) + 0.5)
	ui.menuButton.SetPositionX(math32.Round(float32(width)-ui.menuButton.Width()-gameScreenPadding) + 0.5)
	ui.menuButton.SetPositionY(math32.Round(float32(height)-ui.menuButton.Height()-gameScreenPadding) + 0.5)
	ui.instructions1.SetWidth(float32(width))
	ui.instructions1.SetPositionY(4 * ui.instructions1.ContentHeight())
	ui.instructions2.SetWidth(float32(width))
	ui.instructions2.SetPositionY(5 * ui.instructions2.ContentHeight())
	ui.instructions3.SetWidth(float32(width))
	ui.instructions3.SetPositionY(float32(height) - 2*ui.instructions3.ContentHeight())
	buttonInstructionsPad := float32(24)
	ui.instructionsRestart.SetPositionX(buttonInstructionsPad)
	ui.instructionsRestart.SetPositionY(float32(height) - 6*ui.instructionsRestart.ContentHeight())
	ui.instructionsMenu.SetPositionX(float32(width) - ui.instructionsMenu.ContentWidth() - buttonInstructionsPad)
	ui.instructionsMenu.SetPositionY(float32(height) - 6*ui.instructionsMenu.ContentHeight())
}

// ToggleMenu switched the menu, title, and credits overlay for the in-level corner buttons
func (ui *UI) ToggleMenu() {
	if ui.inMenu {
		// Dispatch OnCursorLeave and OnMouseUp to sliders in case user had cursor over sliders when they pressed Esc to hide menu
		ui.sfxSlider.Dispatch(gui.OnCursorLeave, &window.MouseEvent{})
		ui.sfxSlider.Dispatch(gui.OnMouseUp, &window.MouseEvent{})
		ui.musicSlider.Dispatch(gui.OnCursorLeave, &window.MouseEvent{})
		ui.musicSlider.Dispatch(gui.OnMouseUp, &window.MouseEvent{})

		ui.Remove(ui.menuScreen)
		ui.Add(ui.gameScreen)
		
		ui.game.orbit.SetEnabled(camera.OrbitRot + camera.OrbitZoom)
		ui.game.gopherLocked = false
		ui.game.audio.musicMenu.Stop()
		ui.game.audio.musicGame.Play()
	} else {
		ui.Remove(ui.gameScreen)
		ui.Add(ui.menuScreen)

		ui.game.orbit.SetEnabled(camera.OrbitNone)
		ui.game.gopherLocked = true
		ui.game.audio.musicGame.Stop()
		ui.game.audio.musicMenu.Play()
	}
	ui.inMenu = !ui.inMenu
}

// CreateMenuScreen creates the menu screen widgets
func (ui *UI) CreateMenuScreen() {

	var err error

	// Create menu screen container panel
	ui.menuScreen = gui.NewPanel(100, 100)
	ui.menuScreen.SetColor4(&math32.Color4{0.1, 0.1, 0.1, 0.6})

	// Main menu panel
	ui.menuPanel = gui.NewPanel(600, 300)
	ui.menuPanel.SetZLayerDelta(2)
	mainLayout := gui.NewVBoxLayout()
	mainLayout.SetAlignV(gui.AlignHeight)
	ui.menuPanel.SetLayout(mainLayout)
	ui.menuPanel.SetBorders(2, 2, 2, 2)
	ui.menuPanel.SetBordersColor4(&sliderBorderColor)
	ui.menuPanel.SetColor4(&math32.Color4{0.2, 0.2, 0.2, 0.6})

	// Main menu's top row
	topRow := gui.NewPanel(ui.menuPanel.ContentWidth(), 100)
	topRowLayout := gui.NewHBoxLayout()
	topRowLayout.SetAlignH(gui.AlignWidth)
	topRow.SetLayout(topRowLayout)
	alignCenterVerical := gui.HBoxLayoutParams{Expand: 0, AlignV: gui.AlignCenter}

	// Music Control
	musicControl := gui.NewPanel(130, 100)
	musicControl.SetLayout(topRowLayout)
	ui.musicButton, err = gui.NewImageButton("./gui/music_normal.png")
	ui.musicButton.SetImage(gui.ButtonOver, "./gui/music_hover.png")
	ui.musicButton.SetImage(gui.ButtonPressed, "./gui/music_click.png")
	ui.musicButton.SetImage(gui.ButtonDisabled, "./gui/music_disabled2.png")
	if err != nil {
		panic(err)
	}
	ui.musicButton.Subscribe(gui.OnMouseUp, func(evname string, ev interface{}) {
		ui.game.audio.click.Play()
		ui.game.userData.MusicOn = !ui.game.userData.MusicOn
		ui.UpdateMusicButton(ui.game.userData.MusicOn)
	})
	ui.musicButton.Subscribe(gui.OnCursorEnter, func(evname string, ev interface{}) {
		ui.game.audio.hover.Play()
	})
	musicControl.Add(ui.musicButton)

	// Music Volume Slider
	ui.musicSlider = gui.NewVSlider(20, 80)
	ui.musicSlider.SetValue(ui.game.userData.MusicVol)
	ui.musicSlider.Subscribe(gui.OnChange, func(evname string, ev interface{}) {
		ui.game.audio.SetMusicVolume(ui.musicSlider.Value())
	})
	ui.musicSlider.Subscribe(gui.OnCursorEnter, func(evname string, ev interface{}) {
		ui.game.audio.hover.Play()
	})
	ui.musicSlider.SetLayoutParams(&alignCenterVerical)
	musicControl.Add(ui.musicSlider)

	topRow.Add(musicControl)

	// Sound Effects Control
	sfxControl := gui.NewPanel(130, 100)
	sfxControl.SetLayout(topRowLayout)

	ui.sfxButton, err = gui.NewImageButton("./gui/sound_normal.png")
	ui.sfxButton.SetImage(gui.ButtonOver, "./gui/sound_hover.png")
	ui.sfxButton.SetImage(gui.ButtonPressed, "./gui/sound_click.png")
	ui.sfxButton.SetImage(gui.ButtonDisabled, "./gui/sound_disabled2.png")
	if err != nil {
		panic(err)
	}
	ui.sfxButton.Subscribe(gui.OnMouseUp, func(evname string, ev interface{}) {
		ui.game.audio.click.Play()
		ui.game.userData.SfxOn = !ui.game.userData.SfxOn
		ui.UpdateSfxButton(ui.game.userData.SfxOn)
	})
	ui.sfxButton.Subscribe(gui.OnCursorEnter, func(evname string, ev interface{}) {
		ui.game.audio.hover.Play()
	})
	sfxControl.Add(ui.sfxButton)

	// Sound Effects Volume Slider
	ui.sfxSlider = gui.NewVSlider(20, 80)
	ui.sfxSlider.SetValue(ui.game.userData.SfxVol)
	ui.sfxSlider.Subscribe(gui.OnChange, func(evname string, ev interface{}) {
		ui.game.audio.SetSfxVolume(ui.sfxSlider.Value())
	})
	ui.sfxSlider.Subscribe(gui.OnCursorEnter, func(evname string, ev interface{}) {
		ui.game.audio.hover.Play()
	})
	ui.sfxSlider.SetLayoutParams(&alignCenterVerical)
	sfxControl.Add(ui.sfxSlider)

	topRow.Add(sfxControl)

	// FullScreen Button
	ui.fullScreenButton, err = gui.NewImageButton("./gui/screen_normal.png")
	ui.fullScreenButton.SetImage(gui.ButtonOver, "./gui/screen_hover.png")
	ui.fullScreenButton.SetImage(gui.ButtonPressed, "./gui/screen_click.png")
	ui.fullScreenButton.SetImage(gui.ButtonDisabled, "./gui/screen_disabled2.png")
	if err != nil {
		panic(err)
	}
	ui.fullScreenButton.Subscribe(gui.OnMouseUp, func(evname string, ev interface{}) {
		ui.game.audio.click.Play()
		ui.game.ToggleFullScreen()
	})
	ui.fullScreenButton.Subscribe(gui.OnCursorEnter, func(evname string, ev interface{}) {
		ui.game.audio.hover.Play()
	})
	topRow.Add(ui.fullScreenButton)

	ui.menuPanel.Add(topRow)

	buttonRow := gui.NewPanel(ui.menuPanel.ContentWidth(), 100)
	buttonRowLayout := gui.NewHBoxLayout()
	buttonRowLayout.SetAlignH(gui.AlignWidth)
	buttonRow.SetLayout(buttonRowLayout)

	// Quit Button
	ui.quitButton, err = gui.NewImageButton("./gui/quit_normal.png")
	ui.quitButton.SetImage(gui.ButtonOver, "./gui/quit_hover.png")
	ui.quitButton.SetImage(gui.ButtonPressed, "./gui/quit_click.png")
	ui.quitButton.SetImage(gui.ButtonDisabled, "./gui/quit_disabled2.png")
	if err != nil {
		panic(err)
	}
	ui.quitButton.Subscribe(gui.OnMouseUp, func(evname string, ev interface{}) {
		ui.game.Quit()
	})
	ui.quitButton.Subscribe(gui.OnCursorEnter, func(evname string, ev interface{}) {
		ui.game.audio.hover.Play()
	})
	buttonRow.Add(ui.quitButton)

	// Play Button
	ui.playButton, err = gui.NewImageButton("./gui/play_normal.png")
	ui.playButton.SetImage(gui.ButtonOver, "./gui/play_hover.png")
	ui.playButton.SetImage(gui.ButtonPressed, "./gui/play_click.png")
	ui.playButton.SetImage(gui.ButtonDisabled, "./gui/play_disabled2.png")
	if err != nil {
		panic(err)
	}
	ui.playButton.Subscribe(gui.OnMouseUp, func(evname string, ev interface{}) {
		ui.game.audio.click.Play()
		ui.ToggleMenu()
	})
	ui.playButton.Subscribe(gui.OnCursorEnter, func(evname string, ev interface{}) {
		ui.game.audio.hover.Play()
	})
	buttonRow.Add(ui.playButton)

	ui.menuPanel.Add(buttonRow)

	// Add credits labels
	lCredits1 := gui.NewImageLabel(CREDITS_LINE1)
	lCredits1.SetColor(&creditsColor)
	lCredits1.SetFontSize(20)
	ui.menuScreen.Subscribe(gui.OnResize, func(evname string, ev interface{}) {
		lCredits1.SetWidth(ui.menuScreen.ContentWidth())
		lCredits1.SetPositionY(ui.menuScreen.ContentHeight() - 2*lCredits1.ContentHeight())
	})
	ui.menuScreen.Add(lCredits1)

	lCredits2 := gui.NewImageLabel(CREDITS_LINE2)
	lCredits2.SetColor(&creditsColor)
	lCredits2.SetFontSize(20)
	ui.menuScreen.Subscribe(gui.OnResize, func(evname string, ev interface{}) {
		lCredits2.SetWidth(ui.menuScreen.ContentWidth())
		lCredits2.SetPositionY(ui.menuScreen.ContentHeight() - lCredits2.ContentHeight())
	})
	ui.menuScreen.Add(lCredits2)

	// G3N logo
	g3n := gui.NewImageLabel("")
	g3n.SetImageFromFile("./gui/g3n.png")
	g3n.SetSize(50, 87)
	ui.menuScreen.Subscribe(gui.OnResize, func(evname string, ev interface{}) {
		g3n.SetPositionX(math32.Round(ui.menuScreen.ContentWidth()-g3n.Width()) + 0.5)
		g3n.SetPositionY(math32.Round(ui.menuScreen.ContentHeight()-g3n.Height()) + 0.5)
	})
	ui.menuScreen.Add(g3n)

	ui.menuScreen.Add(ui.menuPanel)
}

// UpdateMusicButton updates the state of the music button, slider, and the appropriate audio setting
func (ui *UI) UpdateMusicButton(on bool) {
	if on {
		ui.musicButton.SetImage(gui.ButtonNormal, "./gui/music_normal.png")
		ui.musicButton.SetImage(gui.ButtonOver, "./gui/music_hover.png")
		ui.musicButton.SetImage(gui.ButtonPressed, "./gui/music_click.png")
		ui.musicSlider.SetEnabled(true)
		ui.musicSlider.SetValue(ui.musicSlider.Value())
	} else {
		ui.musicButton.SetImage(gui.ButtonNormal, "./gui/music_normal_off.png")
		ui.musicButton.SetImage(gui.ButtonOver, "./gui/music_hover_off.png")
		ui.musicButton.SetImage(gui.ButtonPressed, "./gui/music_click_off.png")
		ui.musicSlider.SetEnabled(false)
		ui.game.audio.SetMusicVolume(0)
	}
}

// UpdateSfxButton updates the state of the sfx button, slider, and the appropriate audio setting
func (ui *UI) UpdateSfxButton(on bool) {
	if on {
		ui.sfxButton.SetImage(gui.ButtonNormal, "./gui/sound_normal.png")
		ui.sfxButton.SetImage(gui.ButtonOver, "./gui/sound_hover.png")
		ui.sfxButton.SetImage(gui.ButtonPressed, "./gui/sound_click.png")
		ui.sfxSlider.SetEnabled(true)
		ui.sfxSlider.SetValue(ui.sfxSlider.Value())
	} else {
		ui.sfxButton.SetImage(gui.ButtonNormal, "./gui/sound_normal_off.png")
		ui.sfxButton.SetImage(gui.ButtonOver, "./gui/sound_hover_off.png")
		ui.sfxButton.SetImage(gui.ButtonPressed, "./gui/sound_click_off.png")
		ui.sfxSlider.SetEnabled(false)
		ui.game.audio.SetSfxVolume(0)
	}
}

// CreateGameScreen creates the game screen widgets
func (ui *UI) CreateGameScreen() {

	var err error

	// Create game screen container panel
	ui.gameScreen = gui.NewPanel(100, 100)
	ui.gameScreen.SetEnabled(false)

	// Previous Level Button
	ui.prevButton, err = gui.NewImageButton("./gui/left_normal.png")
	ui.prevButton.SetImage(gui.ButtonOver, "./gui/left_hover.png")
	ui.prevButton.SetImage(gui.ButtonPressed, "./gui/left_click.png")
	ui.prevButton.SetImage(gui.ButtonDisabled, "./gui/left_disabled2.png")
	if err != nil {
		panic(err)
	}
	ui.prevButton.Subscribe(gui.OnMouseUp, func(evname string, ev interface{}) {
		if !ui.prevButton.Enabled() {
			return
		}
		ui.game.audio.click.Play()
		ui.game.PreviousLevel()
	})
	ui.prevButton.Subscribe(gui.OnCursorEnter, func(evname string, ev interface{}) {
		if !ui.prevButton.Enabled() {
			return
		}
		ui.game.audio.hover.Play()
	})
	ui.gameScreen.Add(ui.prevButton)

	// Level Label Image
	ui.levelLabelImage = gui.NewImageLabel("")
	ui.levelLabelImage.SetImageFromFile("./gui/panel.png")
	ui.levelLabelImage.SetHeight(92)
	ui.levelLabelImage.SetEnabled(false)
	ui.gameScreen.Add(ui.levelLabelImage)

	// Level Label Text
	ui.levelLabelText = gui.NewLabel("Level")
	ui.levelLabelText.SetFontSize(35)
	ui.levelLabelText.SetColor(&math32.Color{0, 0, 0})
	ui.levelLabelText.SetPositionY(24)
	ui.levelLabelText.SetEnabled(false)
	ui.gameScreen.Add(ui.levelLabelText)

	// Next Level Button
	ui.nextButton, err = gui.NewImageButton("./gui/right_normal.png")
	ui.nextButton.SetImage(gui.ButtonOver, "./gui/right_hover.png")
	ui.nextButton.SetImage(gui.ButtonPressed, "./gui/right_click.png")
	ui.nextButton.SetImage(gui.ButtonDisabled, "./gui/right_disabled2.png")
	if err != nil {
		panic(err)
	}
	ui.nextButton.Subscribe(gui.OnMouseUp, func(evname string, ev interface{}) {
		if !ui.nextButton.Enabled() {
			return
		}
		ui.game.audio.click.Play()
		ui.game.NextLevel()
	})
	ui.nextButton.Subscribe(gui.OnCursorEnter, func(evname string, ev interface{}) {
		if !ui.nextButton.Enabled() {
			return
		}
		ui.game.audio.hover.Play()
	})
	ui.nextButton.SetPositionY(gameScreenPadding + 0.5)
	ui.gameScreen.Add(ui.nextButton)

	// Restart Level Button
	ui.restartButton, err = gui.NewImageButton("./gui/restart_normal.png")
	ui.restartButton.SetImage(gui.ButtonOver, "./gui/restart_hover.png")
	ui.restartButton.SetImage(gui.ButtonPressed, "./gui/restart_click.png")
	ui.restartButton.SetImage(gui.ButtonDisabled, "./gui/restart_disabled2.png")
	if err != nil {
		panic(err)
	}
	ui.restartButton.Subscribe(gui.OnMouseUp, func(evname string, ev interface{}) {
		if !ui.restartButton.Enabled() {
			return
		}
		ui.game.RestartLevel(true)
	})
	ui.restartButton.Subscribe(gui.OnCursorEnter, func(evname string, ev interface{}) {
		if !ui.restartButton.Enabled() {
			return
		}
		ui.game.audio.hover.Play()
	})
	ui.restartButton.SetPositionX(gameScreenPadding + 0.5)
	ui.gameScreen.Add(ui.restartButton)

	// Show Menu Button
	ui.menuButton, err = gui.NewImageButton("./gui/menu_normal.png")
	ui.menuButton.SetImage(gui.ButtonOver, "./gui/menu_hover.png")
	ui.menuButton.SetImage(gui.ButtonPressed, "./gui/menu_click.png")
	ui.menuButton.SetImage(gui.ButtonDisabled, "./gui/menu_disabled2.png")
	if err != nil {
		panic(err)
	}
	ui.menuButton.Subscribe(gui.OnMouseUp, func(evname string, ev interface{}) {
		if !ui.menuButton.Enabled() {
			return
		}
		ui.game.audio.click.Play()
		ui.ToggleMenu()
	})
	ui.menuButton.Subscribe(gui.OnCursorEnter, func(evname string, ev interface{}) {
		if !ui.menuButton.Enabled() {
			return
		}
		ui.game.audio.hover.Play()
	})
	ui.gameScreen.Add(ui.menuButton)

	// Instructions
	ui.instructions1 = gui.NewImageLabel(INSTRUCTIONS_LINE1)
	ui.instructions1.SetColor(&creditsColor)
	ui.instructions1.SetFontSize(28)
	ui.instructions1.SetEnabled(false)
	ui.gameScreen.Add(ui.instructions1)

	ui.instructions2 = gui.NewImageLabel(INSTRUCTIONS_LINE2)
	ui.instructions2.SetColor(&creditsColor)
	ui.instructions2.SetFontSize(28)
	ui.instructions2.SetEnabled(false)
	ui.gameScreen.Add(ui.instructions2)

	ui.instructions3 = gui.NewImageLabel(INSTRUCTIONS_LINE3)
	ui.instructions3.SetColor(&creditsColor)
	ui.instructions3.SetFontSize(28)
	ui.instructions3.SetEnabled(false)
	ui.gameScreen.Add(ui.instructions3)

	ui.instructionsRestart = gui.NewImageLabel("Restart Level (R)")
	ui.instructionsRestart.SetColor(&creditsColor)
	ui.instructionsRestart.SetFontSize(20)
	ui.instructionsRestart.SetEnabled(false)
	ui.gameScreen.Add(ui.instructionsRestart)

	ui.instructionsMenu = gui.NewImageLabel("Show Menu (Esc)")
	ui.instructionsMenu.SetColor(&creditsColor)
	ui.instructionsMenu.SetFontSize(20)
	ui.instructionsMenu.SetEnabled(false)
	ui.gameScreen.Add(ui.instructionsMenu)
}
