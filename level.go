package main

import (
	"github.com/g3n/engine/camera"
	"github.com/g3n/engine/core"
	"github.com/g3n/engine/light"
	"github.com/g3n/engine/math32"
	"github.com/g3n/engine/window"

	"strings"
)

type CELL_TYPE string

const (
	START          CELL_TYPE = "s"
	BLOCK          CELL_TYPE = "]"
	BOX            CELL_TYPE = "x"
	PAD            CELL_TYPE = "o"
	ELEVATOR       CELL_TYPE = "e"
	ELEVATOR_SHAFT CELL_TYPE = "-"
	NONE           CELL_TYPE = "."
)

type GridCell struct {
	loc GridLoc
	obj IMapObj
}

type GridLoc struct {
	z, x, y int
}

func (l *GridLoc) Equals(v math32.Vector3) bool {
	return l.z == int(v.Z) && l.x == int(v.X) && l.y == int(v.Y)
}

func (l *GridLoc) Vec3() *math32.Vector3 {
	return math32.NewVector3(float32(l.x), float32(l.y), float32(l.z))
}

//func (l *GridLoc) FromVec3(v *math32.Vector3) {
//	l.z = int(v.Z)
//	l.x = int(v.X)
//	l.y = int(v.Y)
//}

// LevelData contains all the logical information about a level
type LevelData struct {
	grid       [][][]GridCell
	gopherInit GridLoc
	boxesInit  []GridLoc
	pads       []GridLoc
	center     math32.Vector3
}

func (ld *LevelData) Get(loc GridLoc) IMapObj {
	return ld.grid[loc.z][loc.x][loc.y].obj
}

func (ld *LevelData) Set(loc GridLoc, gc IMapObj) {
	ld.grid[loc.z][loc.x][loc.y].obj = gc
}

func (ld *LevelData) initLoc(loc GridLoc) {
	ld.grid[loc.z][loc.x][loc.y].loc = loc
}

func (ld *LevelData) IsPad(pl GridLoc) bool {
	for _, a := range ld.pads {
		if a == pl {
			return true
		}
	}
	return false
}

func ParseLevel(data string) (*LevelData, error) {

	ld := new(LevelData)

	// Pad row-wise
	rows := strings.Split(data, "\n")
	for i, row := range rows {
		rows[i] = ". " + row + " ."
	}

	// Pad column-wise
	ncols := len(strings.Fields(rows[0]))
	padding_row := strings.Repeat(". ", ncols)
	rows = append([]string{padding_row}, rows...)
	rows = append(rows, padding_row)
	nrows := len(rows)

	// Calculate max number of floors
	cells := strings.Fields(data)
	nfloors := 1
	for _, c := range cells {
		if len(c) > nfloors {
			nfloors = len(c)
		}
	}
	ADD_TO_NFLOORS := 2
	nfloors += ADD_TO_NFLOORS // necessary

	// Initialize slices
	ld.grid = make([][][]GridCell, nrows)
	for i, _ := range ld.grid {
		ld.grid[i] = make([][]GridCell, ncols)
		for j, _ := range ld.grid[i] {
			ld.grid[i][j] = make([]GridCell, nfloors)
		}
	}

	// Calculate center of level
	ld.center.SetZ(float32(nrows)/2 - 0.5)
	ld.center.SetY(float32(nfloors-ADD_TO_NFLOORS)/2 - 0.5)
	ld.center.SetX(float32(ncols)/2 - 0.5)

	for i, row := range rows {
		cells := strings.Fields(row)
		for j, cell := range cells {
			for k, c := range cell {
				loc := GridLoc{i, j, k}
				ld.initLoc(loc)
				cc := CELL_TYPE(c)
				if cc != NONE {
					switch cc {
					case START:
						ld.gopherInit = loc
						ld.Set(loc, NewGopher(loc))
					case BLOCK:
						ld.Set(loc, NewBlock(loc))
					case BOX:
						box := NewBox(loc)
						ld.boxesInit = append(ld.boxesInit, loc)
						ld.Set(loc, box)
					case PAD:
						ld.pads = append(ld.pads, loc)
						ld.Set(loc, NewPad(loc))
					case ELEVATOR:
						// Calculate number of floors
						var high int
						for high = k; (high+1) < len(cell) && string(cell[high+1]) == string(ELEVATOR_SHAFT); high++ {
						}
						elev := NewElevator(loc, k, high)
						ld.Set(loc, elev)
					}
				}
			}
		}
	}

	return ld, nil
}

// Level stores all the operational data for a level
type Level struct {
	game   *GokobanGame
	scene  *core.Node
	camera *camera.Perspective

	data  *LevelData
	style *LevelStyle

	gopher    *Gopher
	boxes     []*Box
	elevators []*Elevator

	gopherNodeTranslate *core.Node
	gopherNodeRotate    *core.Node
	toAnimate           []*Animation
	animating           bool
	resetAnim           bool
}

// NewLevel returns a pointer to a new Level object
func NewLevel(g *GokobanGame, ld *LevelData, ls *LevelStyle, cam *camera.Perspective) *Level {

	l := new(Level)
	l.game = g
	l.data = ld
	l.style = ls
	l.camera = cam
	l.animating = false

	l.scene = core.NewNode()
	l.scene.SetPosition(-ld.center.X, -ld.center.Y, -ld.center.Z)

	l.gopherNodeTranslate = core.NewNode()
	l.scene.Add(l.gopherNodeTranslate)

	l.gopherNodeRotate = core.NewNode()
	l.gopherNodeTranslate.Add(l.gopherNodeRotate)

	log.Debug("Starting NewLevel loop")
	for i, row := range ld.grid {
		for j, cell := range row {
			for k, c := range cell {
				if c.obj != nil {
					switch obj := c.obj.(type) {
					case *Gopher:
						l.gopher = obj
						l.gopherNodeTranslate.SetPositionVec(c.loc.Vec3())
						obj.SetNode(l.gopherNodeTranslate)

					case *Block:
						mesh := ls.makeBlock()
						obj.SetMesh(mesh)
						l.scene.Add(mesh)

					case *Box:
						l.boxes = append(l.boxes, obj)

						mesh := ls.makeRedBox()
						light := light.NewPoint(l.style.boxLightColorOff, 1.0)

						obj.SetMeshAndLight(mesh, light)
						l.scene.Add(mesh)

					case *Pad:
						light := light.NewPoint(&math32.Color{1, 1, 0}, 1.0)
						padPos := c.loc.Vec3()
						//padPos.Y -= 0.2
						light.SetPositionVec(padPos)
						l.scene.Add(light)

						// Remove pad as logical object from map
						ld.Set(c.loc, nil)

						// if block below, change texture
						if b, ok := ld.grid[i][j][k-1].obj.(*Block); ok {
							b.mesh.AddGroupMaterial(ls.padMaterial, 2)
						}
						// TODO (maybe)
						// else if blocks around, use texture on all existing sides
						// else if no blocks around create transparent small cube mesh indicating objective

					case *Elevator:
						l.elevators = append(l.elevators, obj)

						mesh := ls.makeElevator()
						obj.SetMesh(mesh)
						l.scene.Add(mesh)

						light := light.NewPoint(&math32.Color{0, 0, 1}, 1.0)
						mesh.Add(light)
					}
				}
			}
		}
	}

	// Add a single point light above the level
	light := light.NewPoint(&math32.Color{1, 1, 1}, 8.0)
	light.SetPosition(l.data.center.X, l.data.center.Y*2+2, l.data.center.Z)
	l.scene.Add(light)

	return l

}

// Restart restarts the level
func (l *Level) Restart(playSound bool) {

	log.Debug("Restart")

	l.animating = false
	l.resetAnim = true

	l.game.restartButton.SetEnabled(false)

	// Stop all sounds
	if l.game.audioAvailable {
		l.game.walkPlayer.Stop()
		l.game.bumpPlayer.Stop()
		l.game.gopherFallEndPlayer.Stop()
		l.game.gopherFallStartPlayer.Stop()
		l.game.boxPushPlayer.Stop()
		l.game.boxOnPadPlayer.Stop()
		l.game.boxOffPadPlayer.Stop()
		l.game.boxFallEndPlayer.Stop()
		l.game.boxFallStartPlayer.Stop()
		l.game.elevatorUpPlayer.Stop()
		l.game.elevatorDownPlayer.Stop()
		l.game.levelDonePlayer.Stop()
		l.game.levelFailPlayer.Stop()
	}

	if playSound && l.game.steps != 0 {
		l.game.PlaySound(l.game.levelRestartPlayer, nil)
	}

	l.game.steps = 0

	l.SetPosition(l.gopher, l.data.gopherInit)

	for i, box := range l.boxes {
		l.boxOffPad(box, false)
		l.SetPosition(box, l.data.boxesInit[i])
	}

	for _, elev := range l.elevators {
		lowLoc := elev.Location()
		lowLoc.y = elev.low
		l.SetPosition(elev, lowLoc)
	}
}

// SetPosition moves an object in the data grid along with its node to the desired position
func (l *Level) SetPosition(obj IMapObj, dest GridLoc) {
	l.data.Set(obj.Location(), nil)
	obj.SetLocation(dest)
	l.data.Set(obj.Location(), obj)
	obj.Node().SetPositionVec(dest.Vec3())
}

// onKey handles keyboard events for the level
func (l *Level) onKey(evname string, ev interface{}) {

	if !l.game.gopherLocked {

		xd := int(l.game.stepDelta.X)
		zd := int(l.game.stepDelta.Y)

		kev := ev.(*window.KeyEvent)
		switch kev.Keycode {
		case window.KeyW, window.KeyUp:
			log.Debug("Up")
			l.step(zd, xd)
		case window.KeyS, window.KeyDown:
			log.Debug("Down")
			l.step(-zd, -xd)
		case window.KeyA, window.KeyLeft:
			log.Debug("Left")
			l.step(-xd, zd)
		case window.KeyD, window.KeyRight:
			log.Debug("Right")
			l.step(xd, -zd)
		}
	}
}

// Update updates all ongoing animations for the level
func (l *Level) Update(timeDelta float64) {

	if l.resetAnim {
		l.resetAnim = false
		l.toAnimate = make([]*Animation, 0)
	}

	newToAnimate := l.toAnimate
	l.toAnimate = make([]*Animation, 0)

	for _, anim := range newToAnimate {
		if !l.resetAnim {
			still_animating := anim.Update(timeDelta)
			if still_animating {
				// copy to new slice
				l.toAnimate = append(l.toAnimate, anim)
			}
		}
	}

}

// animate queues a movement animation for an object and also moves the object in the grid
func (l *Level) animate(obj IMapObj, dest GridLoc, delete bool, cb func(interface{})) {

	log.Debug("Queueing animation %+v %+v", obj, dest)

	// Queue animation
	anim := NewAnimation(obj.Node(), dest.Vec3(), cb, obj)
	l.toAnimate = append(l.toAnimate, anim)

	// Move in matrix
	oloc := obj.Location()
	l.data.Set(oloc, nil)
	if delete == false {
		l.data.Set(dest, obj)
		obj.SetLocation(dest)
	}
}

// getCellRelativeToLoc returns the object and location
// relative to the provided location using the provided deltas
func (l *Level) getCellRelativeToLoc(p GridLoc, zd, xd, yd int) (IMapObj, GridLoc) {
	cell_loc := p
	cell_loc.x += xd
	cell_loc.y += yd
	cell_loc.z += zd
	return l.data.Get(cell_loc), cell_loc
}

// getCellRelativeTo returns the object and location
// relative to the location of the provided object using the provided deltas
func (l *Level) getCellRelativeTo(c IMapObj, zd, xd, yd int) (IMapObj, GridLoc) {
	cell_loc := c.Location()
	cell_loc.x += xd
	cell_loc.y += yd
	cell_loc.z += zd
	return l.data.Get(cell_loc), cell_loc
}

// step processes a gopher step to the provided direction
func (l *Level) step(zd, xd int) {

	// Only process step if not already animating another
	// TODO else - add to queue?
	if !l.animating {

		l.game.restartButton.SetEnabled(true)

		// Rotate gopher
		if xd > 0 {
			l.gopherNodeRotate.SetRotationY(0)
		}
		if xd < 0 {
			l.gopherNodeRotate.SetRotationY(math32.Pi)
		}
		if zd > 0 {
			l.gopherNodeRotate.SetRotationY(math32.Pi * 3 / 2)
		}
		if zd < 0 {
			l.gopherNodeRotate.SetRotationY(math32.Pi / 2)
		}

		// Check if can move
		c, cl := l.getCellRelativeTo(l.gopher, zd, xd, 0)

		if c != nil {
			if c.IsPushable() {
				// Check if box can be pushed (if there is space behind it)
				cn, cnl := l.getCellRelativeTo(l.gopher, 2*zd, 2*xd, 0)
				if cn == nil {
					l.pushBox(c, cnl)
					l.moveGopherTo(cl)
				} else {
					l.wallBump()
				}
			} else {
				l.wallBump()
			}
		} else {
			l.moveGopherTo(cl)
		}
	}
}

func (l *Level) wallBump() {
	log.Debug("Hit wall")
	l.game.PlaySound(l.game.bumpPlayer, nil)
}

// moveGopherTo moves the gopher and sets up the appropriate callbacks
func (l *Level) moveGopherTo(pos GridLoc) {

	l.game.steps++
	l.animating = true

	floor, _ := l.getCellRelativeToLoc(pos, 0, 0, -1)
	if floor == nil {
		l.game.PlaySound(l.game.gopherFallStartPlayer, nil)
	} else {
		l.game.PlaySound(l.game.walkPlayer, nil)
	}

	oldloc := l.gopher.Location()
	l.animate(l.gopher, pos, false, func(obj interface{}) {
		l.moveAwayFrom(oldloc)
		l.afterMove(obj)
	})
}

// levelComplete returns true if all the boxes are on pads
func (l *Level) levelComplete() bool {
	for _, p := range l.data.pads {
		if _, ok := l.data.Get(p).(*Box); !ok {
			return false
		}
	}
	return true
}

// boxOnPad handles what happens when a box enters a pad
func (l *Level) boxOnPad(box *Box, playSound bool) {

	log.Debug("Box on pad")
	if box.light.Color() == *l.style.boxLightColorOff {
		if playSound {
			l.game.PlaySound(l.game.boxOnPadPlayer, box.Node())
		}
		log.Debug("...replacing mesh and changing light color")
		l.scene.Remove(box.mesh)
		newMesh := l.style.makeGreenBox()
		box.light.SetColor(l.style.boxLightColorOn)
		box.SetMeshAndLight(newMesh, box.light)
		l.scene.Add(newMesh)
		if l.levelComplete() {
			l.game.PlaySound(l.game.levelDonePlayer, nil)
			l.game.LevelComplete()
		}
	}
}

// boxOffPad handles what happens when a box leaves a pad
func (l *Level) boxOffPad(box *Box, playSound bool) {

	log.Debug("Box off pad")
	if box.light.Color() == *l.style.boxLightColorOn {
		if playSound {
			l.game.PlaySound(l.game.boxOffPadPlayer, box.Node())
		}
		log.Debug("...replacing mesh and changing light color")
		l.scene.Remove(box.mesh)
		newMesh := l.style.makeRedBox()
		box.light.SetColor(l.style.boxLightColorOff)
		box.SetMeshAndLight(newMesh, box.light)
		l.scene.Add(newMesh)
	}
}

// afterFallSound
func (l *Level) afterFallSound(o interface{}, numFloors int) {

	obj := o.(IMapObj)

	floor, _ := l.getCellRelativeTo(obj, 0, 0, -1)
	if _, objIsGopher := obj.(*Gopher); objIsGopher && numFloors >= 1 {
		l.game.PlaySound(l.game.gopherFallEndPlayer, nil)
	} else if box, objIsBox := obj.(*Box); objIsBox {
		if _, floorIsGopher := floor.(*Gopher); floorIsGopher {
			l.game.PlaySound(l.game.gopherHurtPlayer, nil)
		} else { //if !l.data.IsPad(obj.Location()) {
			l.game.PlaySound(l.game.boxFallEndPlayer, box.Node())
		}
	}
}

// afterNewFloor
func (l *Level) afterNewFloor(obj interface{}) {

	log.Debug("afterNewFloor")
	l.animating = false
	o := obj.(IMapObj)

	box, isBox := obj.(*Box)

	floor, _ := l.getCellRelativeTo(o, 0, 0, -1)
	if elev, ok := floor.(*Elevator); ok {
		l.elevate(elev)
	} else if isBox && l.data.IsPad(box.loc) {
		l.boxOnPad(box, true)
	}
}

// fall
func (l *Level) fall(obj IMapObj, playSound bool) {

	log.Debug("fall")
	l.animating = true

	if playSound {
		if box, ok := obj.(*Box); ok {
			l.game.PlaySound(l.game.boxFallStartPlayer, box.Node())
		}
	}

	var cb func(obj interface{})
	posStart := obj.Location()
	pfall := l.posAfterFallFrom(posStart)
	floors := posStart.y - pfall.y
	del := false
	if pfall.y == 0 {
		log.Debug("...out of game")
		l.game.gopherLocked = true
		l.game.arrowNode.SetVisible(false)
		del = true
		pfall.y = -20
		l.game.PlaySound(l.game.levelFailPlayer, nil)
		cb = func(obj interface{}) {
			log.Debug("Done falling out of game")
			l.game.RestartLevel(true)
		}
	} else {
		log.Debug("...still in game")
		cb = func(obj interface{}) {
			log.Debug("Done falling in game")
			if playSound {
				l.afterFallSound(obj, floors)
			}
			l.afterNewFloor(obj)
		}
	}

	l.animate(obj, pfall, del, cb)
}

func (l *Level) posAfterFallFrom(pos GridLoc) GridLoc {
	pos.y--
	for ; pos.y >= 0 && l.data.Get(pos) == nil; pos.y-- {
	}
	pos.y++
	return pos
}

// afterMove
func (l *Level) afterMove(obj interface{}) {

	log.Debug("afterMove")
	l.animating = false
	o := obj.(IMapObj)

	floor, _ := l.getCellRelativeTo(o, 0, 0, -1)
	if floor == nil {
		l.fall(o, true)
	} else {
		l.afterNewFloor(o)
	}
}

// pushBox
func (l *Level) pushBox(box IMapObj, dest GridLoc) {

	log.Debug("pushBox")
	l.game.PlaySound(l.game.boxPushPlayer, box.Node())

	toMove := make([]IMapObj, 0)
	toFall := make([]IMapObj, 0)
	foundBarrier := false

	// Check if leaving pad
	if l.data.IsPad(box.Location()) && !l.data.IsPad(dest) {
		boxObj := box.(*Box)
		l.boxOffPad(boxObj, true)
	}

	// Iterate through piled boxes
	for box != nil && box.IsPushable() {

		if foundBarrier == false && l.data.Get(dest) == nil {
			toMove = append(toMove, box)
		} else {
			foundBarrier = true
			toFall = append(toFall, box)
		}

		// Update current box and destination
		box, _ = l.getCellRelativeTo(box, 0, 0, 1)
		dest.y++
	}

	// Move boxes toMove, adding a callback to the first one for boxes toFall
	for i, box := range toMove {
		cb := func(obj interface{}) {
			l.afterMove(obj)
		}
		if i == 0 {
			cb = func(obj interface{}) {
				l.afterMove(obj)
				for j, boxToFall := range toFall {
					l.fall(boxToFall, j == 0) // only play sound for the first one
				}
			}
		}
		l.animate(box, GridLoc{dest.z, dest.x, box.Location().y}, false, cb)
	}
}

// moveAwayFrom
func (l *Level) moveAwayFrom(pos GridLoc) {

	log.Debug("moveAwayFrom %+v", pos)

	floor, _ := l.getCellRelativeToLoc(pos, 0, 0, -1)
	if elev, ok := floor.(*Elevator); ok {
		log.Debug("Stepped out from elevator %+v", pos)
		l.lowerElev(elev)
	}

	ceil, _ := l.getCellRelativeToLoc(pos, 0, 0, 1)
	if ceil != nil {
		if ceil.IsPushable() {
			log.Debug("Stepped out from under box(es) %+v", pos)
			box := ceil
			for box != nil && box.IsPushable() {
				box_above, _ := l.getCellRelativeTo(box, 0, 0, 1)
				l.fall(box, true) // TODO only true for the first one
				box = box_above
			}
		} else if elev, ok := ceil.(*Elevator); ok {
			cargo := l.getCargo(elev)
			if len(cargo) == 0 {
				l.lowerElev(elev)
			}
		}
	}
}

// lowerElev lowers the specified elevator as far as it can go
func (l *Level) lowerElev(elev *Elevator) {

	log.Debug("lowerElev %+v", elev)
	newloc := elev.loc
	for newloc.y = elev.loc.y - 1; newloc.y >= elev.low && l.data.Get(newloc) == nil; newloc.y-- {
	}
	newloc.y++
	if newloc.y != elev.loc.y {
		log.Debug("Lowering elevator")
		l.game.PlaySound(l.game.elevatorDownPlayer, elev.Node())
		l.animate(elev, newloc, false, func(obj interface{}) {
			if l.game.audioAvailable {
				l.game.elevatorDownPlayer.Stop()
			}
		})
	}
}

// getCargo returns the list of objects on top of the specified elevator
func (l *Level) getCargo(elev *Elevator) []IMapObj {

	cargo := make([]IMapObj, 0)
	for y := 1; ; y++ {
		c, _ := l.getCellRelativeTo(elev, 0, 0, y)
		if c == nil || !c.IsPushable() {
			break
		} else {
			cargo = append(cargo, c)
		}
	}

	return cargo
}

// elevate moves elevator and cargo up as far as it can go
func (l *Level) elevate(elev *Elevator) {

	log.Debug("elevate %+v", elev)

	var spaces_above_cargo int
	max_elevation := elev.high - elev.loc.y

	if max_elevation > 0 {
		l.game.PlaySound(l.game.elevatorUpPlayer, elev.Node())

		l.animating = true
		cargo := l.getCargo(elev)
		last := cargo[len(cargo)-1]

		// Iterate above cargo, and calculate how many floors we can move up
		for y := 1; spaces_above_cargo < max_elevation; y++ {
			c, _ := l.getCellRelativeTo(last, 0, 0, y)
			if c == nil {
				spaces_above_cargo++
			} else {
				break
			}
		}

		log.Debug("Elevating %v", spaces_above_cargo)
		// Move elevator and cargo up (need to move highest/last things first)
		for i := len(cargo) - 1; i >= 0; i-- {
			c := cargo[i]
			up := c.Location()
			up.y += spaces_above_cargo
			l.animate(c, up, false, nil)
		}

		up := elev.Location()
		up.y += spaces_above_cargo
		l.animate(elev, up, false, func(interface{}) {
			if l.game.audioAvailable {
				l.game.elevatorUpPlayer.Stop()
			}
			l.animating = false
		})

	}
}
