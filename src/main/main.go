package main

import (
	"gors"
	"flag"
)

func main() {
	storageDir := flag.String("storage", "storage", "Storage Root Directory")
	storageMode := flag.String("mode", gors.HOME, "Storage Mode")
	chown := flag.Bool("chown", false, "Chown new files?")
	resourcesPath := flag.String("resources", "src", "Path for templates and css")
	port := flag.Int("port", 8888, "Server Port")
	flag.Parse()
	gors.StartServer(*storageDir, gors.StorageMode(*storageMode), *chown, *resourcesPath, *port);
}
