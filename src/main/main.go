package main

import (
	"gors"
	"flag"
)

func main() {
	storageDir := flag.String("storage", "storage", "Storage Root Directory")
	storageMode := flag.String("mode", gors.HOME, "Storage Mode")
	chown := flag.String("chown", "", "Chown files to provided user name or use authenticated user name (*)")
	resourcesPath := flag.String("resources", "src", "Path for templates and css")
	port := flag.Int("port", 8888, "Server Port")
	externalBaseUrl := flag.String("url", "", "External Base URL")
	flag.Parse()
	gors.StartServer(*storageDir, gors.StorageMode(*storageMode), *chown, *resourcesPath, *port, *externalBaseUrl);
}
