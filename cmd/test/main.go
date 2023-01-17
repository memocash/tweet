package test

import (
	"github.com/spf13/cobra"
	"time"
)

var testCmd = &cobra.Command{
	Use:   "test",
	Short: "Debugging (currently for database debugging)",
	Run: func(c *cobra.Command, args []string) {
		//create a goroutine that prints out "hi" every second, using a timer channel
		timer := time.NewTicker(time.Second)
		go func() {
			for {
				select {
				case <-timer.C:
					println("hi")
				}
			}
		}()
	},
}
func GetCommand() *cobra.Command {
	return testCmd
}

