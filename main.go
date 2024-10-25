package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"path/filepath"
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
	KEY_LEFT   = 68
	KEY_UP     = 65
	KEY_RIGHT  = 67
	KEY_CTRL_C = 3
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

// add_song adds a song object to the given library.
func add_song(library *Library, song_name string, location string) error {

	// should do some handling to download song if url
	// should also move the file to the current library

	_, _ = fmt.Println("Filename: " + filepath.Base(location))
	wd, _ := os.Getwd()
	new_path := wd + "/library/" + filepath.Base(location)
	fmt.Println(new_path)
	err := os.Rename(location, new_path)

	if err != nil {
		return err
	}

	library.Songs = append(library.Songs, Song{new_path, song_name})

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
func play_library(library Library) error {

	// reader := bufio.NewReader(os.Stdin)

	fmt.Println("Controls:\n\t-<enter or spacebar> to pause/play\n\t-<right arrow> to skip\n\t-<left arrow> to go back\n\t-<up arrow> to restart song")

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

	defer close(progress_bar_done)
	defer close(quit_progressbar)
	defer close(quit_program)
	defer close(quit_input)
	defer close(pause)
	defer close(skip)
	defer close(back)
	defer close(restart)

	first_run := true

	for i := 0; i < len(library.Songs); i++ {

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
					bar.Close()
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
				}

			}
		}()

		// handle input signals
	inner:
		for {
			select {
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

func main() {
	DEFAULT_PATH := "./library/library.json"

	library, err := get_library(DEFAULT_PATH)

	if err != nil {
		fmt.Println("couldn't load library; " + err.Error())
		return
	}

	// add_song(&library)

	song_name := flag.String("name", "", "The name of the song to add. Must be used with the location flag")
	song_location := flag.String("location", "", "The location of the song to add. Must be used with the name flag")
	play := flag.Bool("play", false, "If enabled, plays the music in the library in a random order")

	flag.Parse()

	if !*play && len(flag.Args()) == 0 {
		flag.Usage()
		os.Exit(1)
	}

	if len(*song_name) != 0 || len(*song_location) != 0 {

		if len(*song_name) == 0 {
			fmt.Println("Enter a song name")
			os.Exit(1)
		}

		if len(*song_location) == 0 {
			fmt.Println("Enter a song location")
			os.Exit(1)
		}

		err = add_song(&library, *song_name, *song_location)

		if err != nil {
			fmt.Println("error:( " + err.Error())
			os.Exit(1)
		}

		err = save_library(&library, DEFAULT_PATH)

		if err != nil {
			fmt.Println("error:( " + err.Error())
			os.Exit(1)
		}

	}

	if *play {

		err = play_library(library)

		if err != nil {
			fmt.Println(err.Error())
		}
	}

	os.Exit(0)
}
