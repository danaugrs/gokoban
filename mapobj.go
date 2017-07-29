// Copyright 2017 Daniel Salvadori. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"github.com/g3n/engine/core"
	"github.com/g3n/engine/graphic"
	"github.com/g3n/engine/light"
)

// MapObj describes the bare minimum needed by a game object that occupies a grid cell
type MapObj struct {
	node     *core.Node
	loc      GridLoc
	pushable bool
}

func (mo *MapObj) Node() *core.Node {
	return mo.node
}

func (mo *MapObj) Location() GridLoc {
	return mo.loc
}

func (mo *MapObj) SetLocation(l GridLoc) {
	mo.loc = l
}

func (mo *MapObj) IsPushable() bool {
	return mo.pushable
}

type IMapObj interface {
	Node() *core.Node
	Location() GridLoc
	SetLocation(GridLoc)
	IsPushable() bool
}

// Box
type Box struct {
	MapObj
	mesh  *graphic.Mesh
	light *light.Point
}

func NewBox(loc GridLoc) *Box {
	b := new(Box)
	b.loc = loc
	b.pushable = true
	return b
}

func (b *Box) SetMeshAndLight(mesh *graphic.Mesh, light *light.Point) {
	b.mesh = mesh
	b.node = &mesh.Node
	mesh.SetPositionVec(b.loc.Vec3())
	b.light = light
	b.node.Add(light)
}

// Block
type Block struct {
	MapObj
	mesh *graphic.Mesh
}

func NewBlock(loc GridLoc) *Block {
	b := new(Block)
	b.loc = loc
	b.pushable = false
	return b
}

func (b *Block) SetMesh(mesh *graphic.Mesh) {
	b.mesh = mesh
	b.node = &mesh.Node
	mesh.SetPositionVec(b.loc.Vec3())
}

// Elevator
type Elevator struct {
	MapObj
	mesh *graphic.Mesh
	low  int
	high int
}

func NewElevator(loc GridLoc, low, high int) *Elevator {
	b := new(Elevator)
	b.loc = loc
	b.pushable = false
	b.low = low
	b.high = high
	return b
}

func (b *Elevator) SetMesh(mesh *graphic.Mesh) {
	b.mesh = mesh
	b.node = &mesh.Node
	mesh.SetPositionVec(b.loc.Vec3())
}

// Gopher
type Gopher struct {
	MapObj
}

func NewGopher(loc GridLoc) *Gopher {
	g := new(Gopher)
	g.loc = loc
	g.pushable = true
	return g
}

func (b *Gopher) SetNode(node *core.Node) {
	b.node = node
}

// Pad
type Pad struct {
	MapObj
	mesh *graphic.Mesh
}

func NewPad(loc GridLoc) *Pad {
	b := new(Pad)
	b.loc = loc
	b.pushable = false
	return b
}

func (b *Pad) SetMesh(mesh *graphic.Mesh) {
	b.mesh = mesh
	b.node = &mesh.Node
	mesh.SetPositionVec(b.loc.Vec3())
}
