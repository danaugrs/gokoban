// Copyright 2017 Daniel Salvadori. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"encoding/gob"
	"os"
)

// The filepath of the binary file used to store the UserData instance
const USER_DATA_FILENAME string = "/gokoban-user-data"

// UserData stores all the information that persists between game sessions
type UserData struct {
	MusicOn           bool
	MusicVol          float32
	SfxOn             bool
	SfxVol            float32
	LastLevel         int
	LastUnlockedLevel int
	FullScreen        bool
}

// NewUserData loads user data from file or creates a new object with default values if no file exists
func LoadOrCreateUserData() *UserData {
	ud := new(UserData)

	// Get user's cache directory
	dataDir, err := os.UserCacheDir()
	if err != nil {
		panic(err)
	}

	// Try to read existing file
	file, err := os.Open(dataDir + USER_DATA_FILENAME)
	if err == nil {
		decoder := gob.NewDecoder(file)
		err = decoder.Decode(ud)
		if err != nil {
			log.Debug("Error decoding gokoban-user-data: %v", err)
		}
	} else {
		log.Debug("Error opening gokoban-user-data: %v", err)
	}
	file.Close()

	// If there was an error either opening or decoding the file
	if err != nil {
		// Set default values
		ud.SfxOn = true
		ud.MusicOn = true
		ud.SfxVol = 0.8
		ud.MusicVol = 0.5
		ud.FullScreen = false
		ud.LastLevel = 0
		ud.LastUnlockedLevel = 0
		log.Debug("Creating new user data with default values: %+v", ud)
	} else {
		log.Debug("Loaded user data: %+v", ud)
	}

	return ud
}

// Save saves the current user data to the user data file, overwriting existing/old data
func (ud *UserData) Save() {
	log.Debug("Saving user data: %+v", ud)

	// Get user's cache directory
	dataDir, err := os.UserCacheDir()
	if err != nil {
		panic(err)
	}

	// Save user data to file
	file, err := os.Create(dataDir + USER_DATA_FILENAME)
	if err == nil {
		encoder := gob.NewEncoder(file)
		err = encoder.Encode(&ud)
		if err != nil {
			log.Debug("Error encoding gokoban-user-data: %v", err)
		}
	} else {
		log.Debug("Error creating gokoban-user-data: %v", err)
	}
	file.Close()
}
