package main

import (
	"fmt"
	"os"
	"bufio"
	"strings"
	"log"
	"strconv"
	"io"
	"errors"
)

type task struct {
	title string
	desc string
}

type state struct {
	tasks []task
	message string
	modifying bool
}

type storage struct {
	fd *os.File
}

func main() {
	store := newStorage()
	defer store.fd.Close()

	state := state{
		tasks: store.read(),
		message: "",
		modifying: false,
	}

	if len(os.Args) == 2 && os.Args[1] == "print" {
		if len(state.tasks) > 0 {
			b := strings.Builder{}
			drawTasks(&state, &b)
			fmt.Print(b.String())
		}
		os.Exit(0)
	}

	input := bufio.NewScanner(os.Stdin)
	running := true

	for running {
		draw(&state)

		cmd := readInput(input)
		if cmd == "" {
			state.message = ""
			continue
		}

		switch cmd {
		case "add", "a":
			state.message = "Enter title (empty to cancel):"
			draw(&state)

			title := readInput(input)
			if title != "" {
				state.message = "Enter description (optional):"
				draw(&state)
				desc := readInput(input)
				state.tasks = append(state.tasks, task{title, desc})
				store.write(state.tasks)
			}

			state.message = ""

		case "delete", "d":
			if len(state.tasks) == 0 {
				state.message = "You have no tasks. Use `add` command to create one"
				break
			}

			state.message = "Choose task to delete (empty to cancel):"
			state.modifying = true
			draw(&state)

			id := readTaskId(input, &state)

			if id != -1 {
				tmp := []task{}

				for i, t := range state.tasks {
					if i != id {
						tmp = append(tmp, t)
					}
				}

				state.tasks = tmp
				store.write(state.tasks)
			}

			state.modifying = false
			state.message = ""

		case "edit", "e":
			if len(state.tasks) == 0 {
				state.message = "You have no tasks. Use `add` command to create one"
				break
			}

			state.message = "Choose task to edit (empty to cancel):"
			state.modifying = true
			draw(&state)

			id := readTaskId(input, &state)

			if id != -1 {
				state.message = "Enter new title (empty to skip):"
				draw(&state)
				title := readInput(input)

				state.message = "Enter new description (empty to skip):"
				draw(&state)
				desc := readInput(input)

				changed := false

				if title != "" {
					state.tasks[id].title = title
					changed = true
				}

				if desc != "" {
					state.tasks[id].desc = desc
					changed = true
				}

				if changed {
					store.write(state.tasks)
				}
			}

			state.modifying = false
			state.message = ""

		case "exit":
			running = false

		case "help":
			state.message = "(a)dd\n(d)elete\n(e)dit\nhelp\nexit"

		default:
			state.message = fmt.Sprintf("Unknown command `%s`. Type `help` to see commands", cmd)
		}
	}
}

func fatal(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func newStorage() *storage {
	dirpath, err := os.UserHomeDir()
	fatal(err)

	dirpath = dirpath + "/.local/share/task-list"

	err = os.MkdirAll(dirpath, 0755)
	fatal(err)

	fd, err := os.OpenFile(dirpath + "/list", os.O_RDWR | os.O_CREATE, 0644)
	fatal(err)

	return &storage{fd}
}

func (store *storage) read() []task {
	bytes, err := io.ReadAll(store.fd)
	fatal(err)

	items := []task{}
	data := string(bytes)

	if len(data) > 0 {
		lines := strings.Split(data, "\n")
		for _, line := range lines[:len(lines) - 1] { // split leaves empty line after last \n
			if title, desc, found := strings.Cut(line, "<-$->"); found {
				if desc == "-$-" {
					desc = ""
				}
				items = append(items, task{title, desc})
			} else {
				fatal(errors.New("Error parsing task: " + line))
			}
		}
	}

	return items
}

func (store *storage) write(ts []task) {
	b := strings.Builder{}

	for _, t := range ts {
		desc := t.desc
		if desc == "" {
			desc = "-$-"
		}
		fmt.Fprintf(&b, "%s<-$->%s\n", t.title, desc)
	}

	err := store.fd.Truncate(0)
	fatal(err)

	_, err = store.fd.Seek(0, os.SEEK_SET)
	fatal(err)

	_, err = store.fd.WriteString(b.String())
	fatal(err)
}

func draw(s *state) {
	b := strings.Builder{}

	b.WriteString("\033c") // clear screen

	drawTasks(s, &b)
	fmt.Fprintf(&b, "%s\n", s.message)

	b.WriteString(">> ") // cursor

	fmt.Print(b.String())
}

func drawTasks(s *state, b *strings.Builder) {
	b.WriteString("┏━━ Task List ━━━\n\n")

	for i, t := range s.tasks {
		if s.modifying {
			fmt.Fprintf(b, "  [%d] %s\n", i + 1, t.title)
			if t.desc != "" {
				fmt.Fprintf(b, "        %s\n", t.desc)
			}
		} else {
			fmt.Fprintf(b, "  %s\n", t.title)
			if t.desc != "" {
				fmt.Fprintf(b, "    %s\n", t.desc)
			}
		}

		b.WriteString("\n")
	}
}

func readInput(s *bufio.Scanner) string {
	s.Scan()
	fatal(s.Err())
	return strings.TrimSpace(s.Text())
}

func readTaskId(input *bufio.Scanner, s *state) int {
	for {
		text := readInput(input)
		if text == "" {
			return -1
		}

		if val, err := strconv.Atoi(text); err == nil && (val >= 1 && val <= len(s.tasks)) {
			// NOTE: in ui we draw ids starting from 1
			return val - 1
		}

		draw(s)
	}
}
