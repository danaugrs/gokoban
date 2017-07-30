import shutil
from subprocess import call
from os.path import join, isfile
import argparse

WINDOWS = 'win'
WINDOWS_DEBUG = 'win-debug'
LINUX = 'linux'

parser = argparse.ArgumentParser()
parser.add_argument("os", help="the operating system being built for", choices=[WINDOWS, WINDOWS_DEBUG, LINUX])
parser.add_argument("version", help="the version of the game")
args = parser.parse_args()

# Build game
if args.os == WINDOWS:
	call(["go", "build", "-ldflags", "-H=windowsgui"], cwd="../")
elif args.os == LINUX or args.os == WINDOWS_DEBUG:
	call(["go", "build"], cwd="../")

# Set the directory/zipfile name
directory = "gokoban-" + args.os + "-" + args.version

# Ignore any .blend or .xcf files
ignore_func = lambda d, files: [f for f in files if isfile(join(d, f))  and (f.endswith('.xcf') or f.endswith('.blend'))]

# Copy necessary files
shutil.copytree('../levels', directory + "/levels")
shutil.copytree('../audio', directory + "/audio")
shutil.copytree('../img', directory + "/img", ignore=ignore_func)
shutil.copytree('../gui', directory + "/gui", ignore=ignore_func)
shutil.copytree('../gopher', directory + "/gopher", ignore=ignore_func)
shutil.copy('../LICENSE', directory)
shutil.copy('../README.md', directory)

# Move executable into directory to be archived
shutil.move('../gokoban.exe', directory)

# If windows, need to copy the sound library DLLs
if args.os == WINDOWS or args.os == WINDOWS_DEBUG:
	shutil.copy('win/libogg.dll', directory)
	shutil.copy('win/libvorbis.dll', directory)
	shutil.copy('win/libvorbisfile.dll', directory)
	shutil.copy('win/OpenAL32.dll', directory)
	shutil.copy('win/vcruntime140.dll', directory)

# Create zip archive and delete temporary directory
shutil.make_archive(directory, 'zip', directory)
shutil.rmtree(directory)
