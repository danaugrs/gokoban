// Copyright 2017 Daniel Salvadori. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"github.com/g3n/engine/geometry"
	"github.com/g3n/engine/graphic"
	"github.com/g3n/engine/material"
	"github.com/g3n/engine/math32"
	"github.com/g3n/engine/texture"
)

// LevelStyle contains all the level styling information and functions
type LevelStyle struct {
	boxLightColorOn  *math32.Color
	boxLightColorOff *math32.Color

	blockMaterial    *material.Phong
	boxMaterialRed   *material.Phong
	boxMaterialGreen *material.Phong
	padMaterial      *material.Phong
	elevatorMaterial *material.Phong

	makeBlock    func() *graphic.Mesh
	makeRedBox   func() *graphic.Mesh
	makeGreenBox func() *graphic.Mesh
	makeElevator func() *graphic.Mesh
}

// NewStandardStyle returns a pointer to a LevelStyle object with standard values
func NewStandardStyle() *LevelStyle {

	s := new(LevelStyle)

	s.boxLightColorOn = &math32.Color{0, 1, 0}  // green
	s.boxLightColorOff = &math32.Color{1, 0, 0} // red

	// Helper function to load texture and handle errors
	newTexture := func(path string) *texture.Texture2D {
		tex, err := texture.NewTexture2DFromImage(path)
		if err != nil {
			log.Fatal("Error loading texture: %s", err)
		}
		return tex
	}

	// Load textures and create materials

	s.blockMaterial = material.NewPhong(math32.NewColor("white"))
	s.blockMaterial.AddTexture(newTexture("img/floor.png"))

	s.padMaterial = material.NewPhong(math32.NewColor("white"))
	s.padMaterial.AddTexture(newTexture("img/pad.png"))
	s.padMaterial.SetTransparent(true) // Makes this material be displayed in front of blockMaterial

	s.boxMaterialRed = material.NewPhong(math32.NewColor("white"))
	s.boxMaterialRed.AddTexture(newTexture("img/crate_red.png"))

	s.boxMaterialGreen = material.NewPhong(math32.NewColor("white"))
	s.boxMaterialGreen.AddTexture(newTexture("img/crate_green2.png"))

	s.elevatorMaterial = material.NewPhong(math32.NewColor("white"))
	s.elevatorMaterial.AddTexture(newTexture("img/metal_diffuse.png"))

	// Create functions that return a cube mesh using the provided material, reusing the same cube geometry

	sharedCubeGeom := geometry.NewCube(1)
	makeCubeWithMaterial := func(mat *material.Phong) func() *graphic.Mesh {
		return func() *graphic.Mesh { return graphic.NewMesh(sharedCubeGeom, mat) }
	}

	s.makeBlock = makeCubeWithMaterial(s.blockMaterial)
	s.makeRedBox = makeCubeWithMaterial(s.boxMaterialRed)
	s.makeGreenBox = makeCubeWithMaterial(s.boxMaterialGreen)
	s.makeElevator = makeCubeWithMaterial(s.elevatorMaterial)

	return s
}
