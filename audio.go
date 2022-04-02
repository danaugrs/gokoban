// Copyright 2017 Daniel Salvadori. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"github.com/g3n/engine/audio"
)

// Audio contains all sounds and music used by the game
type Audio struct {
	// Music
	musicGame *audio.Player
	musicMenu *audio.Player

	// Gameplay sound effects
	levelDone    *audio.Player
	levelRestart *audio.Player
	levelFail    *audio.Player
	gameComplete *audio.Player

	// User interface sound effects
	click *audio.Player
	hover *audio.Player

	// Gopher sound effects
	gopherWalk      *audio.Player
	gopherBump      *audio.Player
	gopherHurt      *audio.Player
	gopherFallStart *audio.Player
	gopherFallEnd   *audio.Player

	// Box sound effects
	boxPush      *audio.Player
	boxOnPad     *audio.Player
	boxOffPad    *audio.Player
	boxFallStart *audio.Player
	boxFallEnd   *audio.Player

	// Elevator sound effects
	elevatorUp   *audio.Player
	elevatorDown *audio.Player
}

// NewAudio creates and returns a new Audio instance with sounds and music ready to go
func NewAudio() *Audio {
	a := new(Audio)

	// Helper function to create player and handle errors
	createPlayer := func(fileName string) *audio.Player {
		log.Debug("Creating sound player for: " + fileName)
		p, err := audio.NewPlayer(fileName)
		if err != nil {
			log.Error("Failed to create sound player: %v", err)
		}
		return p
	}

	// Music
	a.musicGame = createPlayer("./audio/music/Lost-Jungle_Looping.ogg")
	a.musicGame.SetLooping(true)
	a.musicMenu = createPlayer("./audio/music/Spooky-Island.ogg")
	a.musicMenu.SetLooping(true)

	// Gameplay sound effects
	a.levelDone = createPlayer("./audio/sfx/level_done.ogg")
	a.levelDone.SetRolloffFactor(0)
	a.levelRestart = createPlayer("./audio/sfx/level_restart.ogg")
	a.levelRestart.SetRolloffFactor(0)
	a.levelFail = createPlayer("./audio/sfx/level_fail.ogg")
	a.levelFail.SetRolloffFactor(0)
	a.gameComplete = createPlayer("./audio/sfx/game_complete.ogg")
	a.gameComplete.SetRolloffFactor(0)

	// User interface sound effects
	a.click = createPlayer("./audio/sfx/button_click.ogg")
	a.click.SetRolloffFactor(0)
	a.hover = createPlayer("./audio/sfx/button_hover.ogg")
	a.hover.SetRolloffFactor(0)

	// Gopher sound effects
	a.gopherWalk = createPlayer("./audio/sfx/gopher_walk.ogg")
	a.gopherWalk.SetRolloffFactor(0)
	a.gopherBump = createPlayer("./audio/sfx/gopher_bump.ogg")
	a.gopherBump.SetRolloffFactor(0)
	a.gopherHurt = createPlayer("./audio/sfx/gopher_hurt.ogg")
	a.gopherHurt.SetRolloffFactor(0)
	a.gopherFallStart = createPlayer("./audio/sfx/gopher_fall_start.ogg")
	a.gopherFallStart.SetRolloffFactor(0)
	a.gopherFallEnd = createPlayer("./audio/sfx/gopher_fall_end.ogg")
	a.gopherFallEnd.SetRolloffFactor(0)

	// Box sound effects
	a.boxPush = createPlayer("./audio/sfx/box_push.ogg")
	a.boxPush.SetRolloffFactor(0)
	a.boxOnPad = createPlayer("./audio/sfx/box_on.ogg")
	a.boxOnPad.SetRolloffFactor(0)
	a.boxOffPad = createPlayer("./audio/sfx/box_off.ogg")
	a.boxOffPad.SetRolloffFactor(0)
	a.boxFallStart = createPlayer("./audio/sfx/box_fall_start.ogg")
	a.boxFallStart.SetRolloffFactor(0)
	a.boxFallEnd = createPlayer("./audio/sfx/box_fall_end.ogg")
	a.boxFallEnd.SetRolloffFactor(0)

	// Elevator sound effects
	a.elevatorUp = createPlayer("./audio/sfx/elevator_up.ogg")
	a.elevatorUp.SetLooping(true)
	a.elevatorUp.SetRolloffFactor(0)
	a.elevatorDown = createPlayer("./audio/sfx/elevator_down.ogg")
	a.elevatorDown.SetLooping(true)
	a.elevatorDown.SetRolloffFactor(0)

	return a
}

// SetMusicVolume sets the volume of the music
func (a *Audio) SetMusicVolume(vol float32) {
	log.Debug("Set Music Volume %v", vol)

	a.musicGame.SetGain(vol)
	a.musicMenu.SetGain(vol)
}

// SetSfxVolume sets the volume of all sound effects
func (a *Audio) SetSfxVolume(vol float32) {
	log.Debug("Set Sfx Volume %v", vol)

	// Gameplay sound effects
	a.levelDone.SetGain(vol)
	a.levelRestart.SetGain(vol)
	a.levelFail.SetGain(vol)
	a.gameComplete.SetGain(vol)

	// User interface sound effects
	a.click.SetGain(vol)
	a.hover.SetGain(vol)

	// Gopher sound effects
	a.gopherWalk.SetGain(vol)
	a.gopherBump.SetGain(vol)
	a.gopherHurt.SetGain(vol)
	a.gopherFallStart.SetGain(vol)
	a.gopherFallEnd.SetGain(vol)

	// Box sound effects
	a.boxPush.SetGain(vol)
	a.boxOnPad.SetGain(vol)
	a.boxOffPad.SetGain(vol)
	a.boxFallStart.SetGain(vol)
	a.boxFallEnd.SetGain(vol)

	// Elevator sound effects
	a.elevatorUp.SetGain(vol)
	a.elevatorDown.SetGain(vol)
}
