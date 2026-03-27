package main

var Version = ""

func main() {
	rootCmd.Version = Version
	Execute()
}
