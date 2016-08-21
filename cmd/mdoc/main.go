package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/pnelson/mdoc"
)

var (
	addr  = flag.String("addr", ":3000", "address to listen on")
	theme = flag.String("theme", defaultTheme, "rendering theme")
	help  = flag.Bool("help", false, "display this usage information")
)

var defaultTheme = filepath.Join(os.Getenv("GOPATH"), "src/github.com/pnelson/mdoc/contrib/themes/default")

func main() {
	flag.Parse()
	args := flag.Args()
	if *help {
		usage(os.Stdout)
		return
	}
	if len(args) > 1 {
		usage(os.Stderr)
		os.Exit(1)
	}
	dir := "."
	if len(args) == 1 {
		dir = args[0]
	}
	m := mdoc.New(dir, mdoc.Theme(*theme))
	err := http.ListenAndServe(":3000", m)
	if err != nil {
		log.Fatal(err)
	}
}

func usage(w io.Writer) {
	fmt.Fprintln(w, "usage: mdoc [-addr=<addr>] [-theme=<theme>] [<dir>]")
	flag.PrintDefaults()
}
