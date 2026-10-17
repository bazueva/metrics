package main

type service struct{}

func (service) Exit(code int) {}

func main() {
	os := service{}
	os.Exit(1)
}
