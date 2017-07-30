Feel free to modify the level files and even create your own!

`s` - The start position of the gopher. There must be one (and only one) of this present.  
`]` - A block.  
`x` - A box.  
`o` - A pad, or "objective" - the position where a box will be activated if placed there. Should be on top of a block e.g. `]o`.  
`e` - An elevator. Should be accompanied by hyphens indicating the elevator's range of motion e.g. `e--` for a 2-story elevator.  
`-` - Indicates the elevator shaft i.e. how far up the elevator goes.  
`.` - Used as a spacer e.g. `].]` will create a floor, a space, and a ceiling. Necessary if a vertical column has absolutely nothing on it.  

In the level files spaces separate vertical columns, which are represented as space-less character sequences.
