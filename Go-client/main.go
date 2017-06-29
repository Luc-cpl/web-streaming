package main

import (
	"fmt"
	"os/exec"
)

func main() {
	cmd := exec.Command(`ffplay -f dshow -i video="Lenovo EasyCamera" -framerate 24 -video_size 320X240`)
	//var buffer bytes.Buffer
	// cmd.Stdout = &buffer
	// if cmd.Run() != nil {
	// 	fmt.Println("could not generate frame")
	// }

	cmd.Start()
	for {
		out, err := cmd.Output()
		if err != nil {
			fmt.Println(err)
			break
		}
		fmt.Println(out)
	}
}
