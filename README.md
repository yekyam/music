# music

Simple music player command line utility.

## Demo

## Features

Does what it says on the label.

- Add songs to a library.
- Randomly shuffles songs from the library.
- Repeat tracks
- Repeat entire library
- Skip
- Pause
- Delete
- Rename
- Auto download song from a link (must have yt-dlp installed)

## Pros

- Small
- Easy to use
- Works 

## Cons

- Have to manually add songs to library
- Formatting gets messed up sometimes idk y

## Usage

To use the program, you pass in a `mode` flag with the given `args`

Example
```
$ ./music -add -name "Goodbye Yellow Brick Road" -location "https://www.youtube.com/watch?v=wy709iNG6i8"
trying to grab song from url, pls be patient
Added `Goodbye Yellow Brick Road` to library at file: `./library/Goodbye_Yellow_Brick_Road.mp3`
```
## Modes
### list
Lists all of the songs in the library, then exits

#### Arguments
None

#### Example Usage
```
$ ./music -list
    -Goodbye Yellow Brick Road
    -Rocket Man
```

### delete
Deletes the specified song in the library, then exits

#### Arguments
- `-name` The name of the song to delete

#### Example Usage
```
$ ./music -delete -name "Goodbye Yellow Brick Road"                                                                                    288.05s (main|💩) 00:15
Are you sure you want to delete `Goodbye Yellow Brick Road` ?
y
$ ./music -list
    -Rocket Man
```

### play
Plays all the song in the library **in a random order**. Displays player controls.

#### Arguments
None

#### Example Usage
```
$ ./music -play
Controls:
        -<enter or spacebar> to pause/play
        -<right arrow> to skip
        -<left arrow> to go back
        -<up arrow> to restart song
        -<L key> to switch loop modes: OFF for no loop, SONG to loop song, and LIB to loop all songs
Like Him   0% |                                                                                 | (0/278, 0 it/hr) [0s:0s]
```

### add
Adds a song to the library.
#### Arguments
- `-name` The name of the song to add
- `-location` The location, can either be a file path or a URL (must have yt-dlp and ffmpeg installed and in your path)

#### Example Usage
```
$ ./music -add -name "Goodbye Yellow Brick Road" -location "https://www.youtube.com/watch?v=wy709iNG6i8"
```

### rename
Renames a song with a new name.
#### Arguments
- `-name` The name of the song to rename
- `-rename_to` The new name of the song
#### Example Usage
```
$ ./music -rename -name "Goodbye Yellow Brick Road" -rename_to "GYBR"
```



