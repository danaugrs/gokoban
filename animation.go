// Copyright 2017 Daniel Salvadori. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"github.com/g3n/engine/core"
	"github.com/g3n/engine/math32"
)

// Animation describes an ongoing constant-speed, linear animation
type Animation struct {
	node     *core.Node        // node to animate
	dest     *math32.Vector3   // the destination
	speed    float32           // how many "blocks" per second
	callback func(interface{}) // function to be called once animation is complete
	cb_arg   interface{}       // arguments stored and passed in to the callback function
}

// NewAnimation returns a pointer to a new Animation object
func NewAnimation(node *core.Node, dest *math32.Vector3, cb func(interface{}), cb_arg interface{}) *Animation {
	a := new(Animation)
	a.node = node
	a.dest = dest
	a.speed = 10
	a.callback = cb
	a.cb_arg = cb_arg
	return a
}

// Update moves the node towards its destination according to speed
// and calls the callback with the previously provided args once finished
func (a *Animation) Update(timeDelta float64) bool {
	pos := a.node.Position()
	delta := math32.NewVector3(0, 0, 0)
	delta.Add(a.dest)
	delta.Sub(&pos)
	dist := delta.Length()
	delta = delta.Normalize().MultiplyScalar(a.speed * float32(timeDelta))
	if dist > delta.Length() {
		a.node.SetPositionVec(pos.Sub(delta))
		return true
	} else {
		a.node.SetPositionVec(a.dest)
		if a.callback != nil {
			a.callback(a.cb_arg)
		}
		return false
	}
}
