package main

import (
	"gors"
	"flag"
)

func main() {
	storageDir := flag.String("storage", "storage", "Storage Root Directory")
	port := flag.Int("port", 8888, "Server Port")
	flag.Parse()
	gors.StartServer(*storageDir,*port);
}
