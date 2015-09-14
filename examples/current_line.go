package main

import (
	"log"

	"github.com/juanolon/go-nvim"
)

func main() {

	n, err := nvim.Dial(nvim.CONN_NET, "/var/folders/x6/h18jf2xj10bgb1_jlf_fy0rr0000gn/T/nvimaGKXbS/0")
	if err != nil {
		log.Fatal("error connecting: ", err)
	}
	line := n.GetCurrentLine()
	log.Println(line)
}
