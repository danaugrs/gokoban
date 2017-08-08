# Gokoban - 3D Puzzle Game
_Awarded 1st place in the [2017 Gopher Game Jam](https://itch.io/jam/gopher-jam)_

Gokoban is a 3D puzzle game written in Go. You control the Go gopher, and your objective in each level is to push the boxes until they are all on top of the yellow pads. There are elevators that help you reach high places and move boxes up and down. Levels are read from text files in [`/levels`](levels) so you can easily modify them and even create new ones.

It was created using [G3N](https://github.com/g3n/engine) for the [2017 Gopher Game Jam](https://itch.io/jam/gopher-jam) on [itch.io](https://itch.io).

### [:white_check_mark: Download Windows 64-bit Precompiled Binary](https://github.com/danaugrs/gokoban/archive/win64-bin.zip)

![Gokoban Screenshots](img/screenshots.gif)

## Building from source

First make sure you have the [G3N external dependencies](https://github.com/g3n/engine#dependencies) in place.

The following command will download and install Gokoban, G3N, and all of G3N's Go package dependencies (make sure your GOPATH is set correctly):

`go get -u github.com/danaugrs/gokoban`

If you are on Windows, you'll need the audio DLLs mentioned in the [G3N readme](https://github.com/g3n/engine#dependencies).
You may also need `vcruntime140.dll`. All the necessary DLLs are provided here under [`dist/win`](dist/win) - you just need to "add" them to your PATH, or copy them to the same folder that your Gokoban executable is in. Alternatively you can build them yourself by following [these instructions](https://github.com/g3n/windows_audio_dlls). You can obtain `vcruntime140.dll` by downloading a [Microsoft Visual C++ Redistributable](https://support.microsoft.com/en-us/help/2977003/the-latest-supported-visual-c-downloads).

## Support

I hope you enjoy playing and learning from Gokoban as much as I enjoyed writing it.

If you come across any issues, please [report them](https://github.com/danaugrs/gokoban/issues).
