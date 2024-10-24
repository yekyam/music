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

type Library struct {
	Songs []Song `json:"songs"`
}

type Song struct {
	Path      string `json:"path"`
	Song_name string `json:"song_name"`
}

const (
	KEY_ENTER  = 13
	KEY_SPACE  = 32
	KEY_LEFT   = 68
	KEY_UP     = 65
	KEY_RIGHT  = 67
	KEY_CTRL_C = 3
)

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

func play_library(library Library) error {

	// reader := bufio.NewReader(os.Stdin)

	fmt.Println("Controls:\n\t-<enter or spacebar> to pause/play\n\t-<right arrow> to skip\n\t-<left arrow> to go back\n\t-<up arrow> to restart song")

	rand.Shuffle(len(library.Songs), func(i, j int) { library.Songs[i], library.Songs[j] = library.Songs[j], library.Songs[i] })

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

		speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10))

		total_seconds := int64(format.SampleRate.D(streamer.Len()).Round(time.Second).Seconds())

		ctrl := &beep.Ctrl{Streamer: beep.Loop(1, streamer), Paused: false}
		speaker.Play(ctrl)

		fmt.Println("Now playing: " + v.Song_name + "\n")
		quit := make(chan bool)
		// pause := make(chan bool)
		// ready := make(chan bool)

		// goroutine for progress bar
		go func() {
			bar := progressbar.Default(total_seconds, v.Song_name)
			for {

				select {
				case <-quit:
					return
				case <-time.After(time.Second):
					if !ctrl.Paused {
						bar.Add(1)
					}
				}

			}
		}()

		for {

			command, err := get_key_press()

			if err != nil {
				return err
			}
			// fmt.Println(command)

			fmt.Println()
			if streamer.Position() == streamer.Len()-1 {
				fmt.Println("Done")
				break
			}

			// pause/unpause
			if command == KEY_ENTER || command == KEY_SPACE {
				speaker.Lock()
				ctrl.Paused = !ctrl.Paused

				if ctrl.Paused {
					fmt.Println("\npaused")
				} else {
					fmt.Println("\nunpaused")
				}

				speaker.Unlock()
			}

			// skip
			if command == KEY_RIGHT {
				fmt.Println()
				speaker.Clear()
				quit <- true
				break
			}

			// restart song
			if command == KEY_UP {
				fmt.Println()
				speaker.Clear()
				i--
				quit <- true
				break
			}

			// back
			if command == KEY_LEFT {

				fmt.Println()
				if i != 0 {
					speaker.Clear()
					i--
					i-- // two because on next iter, i will be inc'd by the outer loop
					quit <- true
					break
				}

			}

			// ctrl+c
			if command == KEY_CTRL_C {
				return nil
			}
		}
	}
	fmt.Println("Done")

	return nil
}

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
	} else {
		os.Exit(1)
	}
	os.Exit(0)

}
