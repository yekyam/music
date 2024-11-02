package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/term"

	"github.com/gopxl/beep"
	"github.com/gopxl/beep/mp3"
	"github.com/gopxl/beep/speaker"
	"github.com/schollz/progressbar/v3"
)

// Library holds a list of songs.
type Library struct {
	Songs []Song `json:"songs"`
}

// Song holds a path to a file and the name of the song.
type Song struct {
	Path      string `json:"path"`
	Song_name string `json:"song_name"`
}

// Constants for keyboard inputs, works on MacOS.
const (
	KEY_ENTER  = 13
	KEY_SPACE  = 32
	KEY_ESCAPE = 27
	KEY_LEFT   = 68
	KEY_UP     = 65
	KEY_RIGHT  = 67
	KEY_CTRL_C = 3
	KEY_L      = 108
)

// create_library_file creates a default JSON file at the given filename.
func create_library_file(filename string) error {

	file, err := os.Create(filename)

	if err != nil {
		return err
	}

	defer file.Close()

	empty_json := "{\"songs\":[]}"

	_, err = file.WriteString(empty_json)

	if err != nil {
		return err
	}

	return nil
}

// save_library saves the given library to the given file as JSON.
func save_library(library *Library, filename string) error {

	library_json, err := json.Marshal(library)

	if err != nil {
		return err
	}

	jsonFile, err := os.Create(filename)

	if err != nil {
		return err
	}

	defer jsonFile.Close()

	_, err = jsonFile.Write(library_json)

	if err != nil {
		return err
	}

	return nil
}

// load_library reads the given JSON file into a Library object.
func load_library(filename string) (Library, error) {
	jsonFile, err := os.Open(filename)

	if err != nil {
		return Library{}, err
	}

	defer jsonFile.Close()

	byteValue, _ := io.ReadAll(jsonFile)

	var library Library

	json.Unmarshal(byteValue, &library)

	return library, nil
}

func add_song_from_path(library *Library, song_name string, path string) error {

	_, _ = fmt.Println("Filename: " + filepath.Base(path))

	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	new_path := wd + "/library/" + filepath.Base(path)
	fmt.Println("Moved to: " + new_path)

	err = os.Rename(path, new_path)
	if err != nil {
		return err
	}

	library.Songs = append(library.Songs, Song{new_path, song_name})

	return nil

}

// add_song adds a song object to the given library.
func add_song(library *Library, song_name string, location string) error {

	// should do some handling to download song if url
	// should also move the file to the current library
	if strings.Contains(location, "https") {

		fmt.Println("trying to grab song from url, pls be patient")

		new_path := "./library/" + strings.Replace(song_name, " ", "_", -1) + ".mp3"

		args := []string{
			"yt-dlp",
			"--extract-audio",
			"--audio-format",
			"mp3",
			location,
			"-o",
			new_path,
		}
		cmd := exec.Command(args[0], args[1:]...)
		output, err := cmd.Output()

		if err != nil {
			fmt.Println(output)
			return err
		}

		library.Songs = append(library.Songs, Song{new_path, song_name})

		fmt.Println("Added `" + song_name + "` to library at file: `" + new_path + "`")

	} else {
		fmt.Println("trying to add song from file path")
		err := add_song_from_path(library, song_name, location)
		if err != nil {
			return err
		}
	}

	return nil
}

// get_key_press grabs whatever key was pressed in the terminal's raw mode.
// Returns the byte representation and any errors.
func get_key_press() (byte, error) {
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return 0, err
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	b := make([]byte, 1)
	_, err = os.Stdin.Read(b)
	if err != nil {
		fmt.Println(err)
		return 0, err
	}
	return b[0], nil
}

// play_library plays the songs in the library. Also displays a progress bar per song, and allows some user controls.
func play_library(library *Library) error {

	// reader := bufio.NewReader(os.Stdin)

	controls := "Controls:" +
		"\n\t-<enter or spacebar> to pause/play" +
		"\n\t-<right arrow> to skip" +
		"\n\t-<left arrow> to go back" +
		"\n\t-<up arrow> to restart song" +
		"\n\t-<L key> to switch loop modes: OFF for no loop, SONG to loop song, and LIB to loop all songs"

	fmt.Println(controls)

	rand.Shuffle(len(library.Songs), func(i, j int) { library.Songs[i], library.Songs[j] = library.Songs[j], library.Songs[i] })

	progress_bar_done := make(chan bool, 1)
	// restart_loop := make(chan bool)
	quit_program := make(chan bool, 1)
	quit_input := make(chan bool, 1)
	quit_progressbar := make(chan bool, 1)

	pause := make(chan bool, 1)
	skip := make(chan bool, 1)
	back := make(chan bool, 1)
	restart := make(chan bool, 1)
	loop := make(chan bool, 1)

	defer close(progress_bar_done)
	defer close(quit_progressbar)
	defer close(quit_program)
	defer close(quit_input)
	defer close(pause)
	defer close(skip)
	defer close(back)
	defer close(restart)
	defer close(loop)

	const (
		LOOP_OFF     = 0
		LOOP_SONG    = 1
		LOOP_LIBRARY = 2
		LOOP_MAX     = 3
	)

	first_run := true

	total_songs := len(library.Songs)
	loop_type := 0

outer:
	for i := 0; ; i++ {

		switch loop_type {
		case LOOP_OFF:
			if i == total_songs {
				break outer
			}
		case LOOP_SONG:
			i--
		case LOOP_LIBRARY:
			i = i % total_songs

		}

		v := library.Songs[i]

		file, err := os.Open(v.Path)

		if err != nil {
			return err
		}

		streamer, format, err := mp3.Decode(file)
		if err != nil {
			log.Fatal(err)
		}
		defer streamer.Close()

		if first_run {
			speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10))
			first_run = false
		}

		total_seconds := int64(format.SampleRate.D(streamer.Len()).Round(time.Second).Seconds())

		ctrl := &beep.Ctrl{Streamer: beep.Loop(1, streamer), Paused: false}
		speaker.Play(ctrl)

		// goroutine for progress bar
		go func() {
			bar := progressbar.Default(total_seconds, v.Song_name)
			for {

				select {
				case <-quit_progressbar:
					bar.Exit()
					return
				case <-time.After(time.Second):
					if bar.State().CurrentNum == bar.State().Max {
						progress_bar_done <- true
						return
					}
					if !ctrl.Paused {
						bar.Add(1)
					}
				}

			}
		}()

		//goroutine for signaling input
		go func() {
			for {

				select {
				case <-quit_input:
					return
				default:
					command, _ := get_key_press()

					// pause/unpause
					if command == KEY_ENTER || command == KEY_SPACE {
						pause <- true
					}

					// skip
					if command == KEY_RIGHT {
						skip <- true
					}

					// restart song
					if command == KEY_UP {
						restart <- true
					}

					// back
					if command == KEY_LEFT {
						back <- true
					}

					// ctrl+c
					if command == KEY_CTRL_C {
						quit_program <- true
					}

					if command == KEY_L {
						loop <- true
					}

				}

			}
		}()

		// handle input signals
	inner:
		for {
			select {

			case <-loop:
				loop_type = (loop_type + 1) % LOOP_MAX
				switch loop_type {
				case LOOP_OFF:
					fmt.Println("\n\nLoop type: OFF")
				case LOOP_SONG:
					fmt.Println("\n\nLoop type: SONG")
				case LOOP_LIBRARY:
					fmt.Println("\n\nLoop type: LIB")
				}

			case <-pause:

				speaker.Lock()
				ctrl.Paused = !ctrl.Paused

				speaker.Unlock()
			case <-skip:

				fmt.Println()
				speaker.Clear()
				streamer.Close()
				quit_input <- true
				quit_progressbar <- true

				break inner

			case <-restart:

				fmt.Println()
				speaker.Clear()
				streamer.Close()
				i--
				quit_input <- true
				quit_progressbar <- true

				break inner
			case <-back:

				if i != 0 {
					speaker.Clear()
					streamer.Close()
					i--
					i-- // two because on next iter, i will be inc'd by the outer loop
					quit_input <- true
					quit_progressbar <- true
					break inner
				}

			case <-progress_bar_done:
				speaker.Clear()
				streamer.Close()

				quit_input <- true
				break inner
			case <-quit_program:

				return nil
				// break inner

			}
		}
	}
	fmt.Println("Done")

	return nil
}

// get_library attempts to load the library at the filename. If the library doesn't exist, a default empty library is created.
func get_library(filename string) (Library, error) {

	library, err := load_library(filename)

	if err != nil {
		fmt.Println("Couldn't open file")

		if os.IsNotExist(err) {
			fmt.Println("Making new file")
			err := create_library_file(filename)

			if err != nil {
				fmt.Println("Couldn't init lib:(")
				return Library{}, err
			}
			fmt.Println("Created new file")
		}
		return Library{}, err
	}

	return library, nil
}

func ask_for_permsission(message string) (bool, error) {

	reader := bufio.NewReader(os.Stdin)

	fmt.Println(message)
	line, err := reader.ReadString('\n')

	if err != nil {
		fmt.Println("Couldn't read string!")
		return false, err
	}

	if strings.ToLower(line)[0] != 'y' {
		return false, nil
	}
	return true, nil
}

func handle_args(library *Library, DEFAULT_PATH string) {

	// modes
	play := flag.Bool("play", false, "Starts program in 'play' mode, "+
		"which plays songs in library and has controls")
	list := flag.Bool("list", false, "Starts program in 'list' mode, "+
		"which just outputs the songs in the library and returns")
	add := flag.Bool("add", false, "Starts program in 'add' mode, "+
		"which adds the song (specified by '-name (name)' and '-location (filepath)') to the library and returns")
	rename := flag.Bool("rename", false, "Starts program in 'rename' mode, "+
		"which renames the song to a new song name (specified by '-name (name)'' and '-rename_to (new name)')")
	delete := flag.Bool("delete", false, "Starts program in delete' mode")

	// args
	song_name := flag.String("name", "", "The `name` of the song to add. Must be used with the location flag")
	song_location := flag.String("location", "", "The `location` of the song to add. Must be used with the name flag")
	renamme_to := flag.String("rename_to", "", "If passed, renames the song in the library specified by -name to the `new name`")

	flag.Parse()

	if *list {
		for _, song := range library.Songs {
			fmt.Println("\t-" + song.Song_name)
		}
		os.Exit(0)
	} else if *rename {

		if len(*song_name) == 0 {
			fmt.Println("Enter a song name")
			os.Exit(1)
		}

		if len(*renamme_to) == 0 {
			fmt.Println("Enter a song location")
			os.Exit(1)
		}

		success := false

		for i, song := range library.Songs {

			if song.Song_name != *song_name {
				continue
			}

			ok, err := ask_for_permsission("Are you sure you want to rename `" + *song_name + "` to `" + *renamme_to + "`?")

			if err != nil {
				fmt.Println("Error; aborting")
				os.Exit(1)
			}

			if !ok {
				fmt.Println("Cancelled operation")
			}

			song.Song_name = *renamme_to
			library.Songs[i] = song
			err = save_library(library, DEFAULT_PATH)

			if err != nil {
				fmt.Println("Couldn't save library after renaming!")
				break
			}
			success = true

		}

		if !success {
			fmt.Println("Couldn't find song: " + *song_name)
			os.Exit(1)
		}
		os.Exit(0)
	} else if *add {

		if len(*song_name) == 0 {
			fmt.Println("Enter a song name")
			os.Exit(1)
		}

		if len(*song_location) == 0 {
			fmt.Println("Enter a song location")
			os.Exit(1)
		}

		err := add_song(library, *song_name, *song_location)

		if err != nil {
			fmt.Println("error:( " + err.Error())
			os.Exit(1)
		}

		err = save_library(library, DEFAULT_PATH)

		if err != nil {
			fmt.Println("error:( " + err.Error())
			os.Exit(1)
		}

	} else if *play {

		err := play_library(library)

		if err != nil {
			fmt.Println(err.Error())
		}
	} else if *delete {

		if len(*song_name) == 0 {
			fmt.Println("Enter a song name")
			os.Exit(1)
		}

		success := false

		for i, song := range library.Songs {

			if song.Song_name != *song_name {
				continue
			}

			ok, err := ask_for_permsission("Are you sure you want to delete `" + *song_name + "` ?")

			if err != nil {
				fmt.Println("Error; aborting")
				os.Exit(1)
			}

			if !ok {
				fmt.Println("Cancelled operation")
			}

			library.Songs = append(library.Songs[:i], library.Songs[i+1:]...)
			err = save_library(library, DEFAULT_PATH)

			if err != nil {
				fmt.Println("Couldn't save library after renaming!")
				break
			}
			success = true

		}

		if !success {
			fmt.Println("Couldn't find song: " + *song_name)
			os.Exit(1)
		}
		os.Exit(0)

	} else {
		fmt.Println("Here")
		flag.Usage()
		os.Exit(1)

	}

}

func main() {
	DEFAULT_PATH := "./library/library.json"

	library, err := get_library(DEFAULT_PATH)

	if err != nil {
		fmt.Println("couldn't load library; " + err.Error())
		return
	}

	// add_song(&library)
	// modes
	handle_args(&library, DEFAULT_PATH)

	os.Exit(0)
}
