package cmd

import (
	"os"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "pullameblophoto",
	Short: "Pull ameblo photo",
	Long:  `Extract image URL from Ameblo and download pictures.`,
}

func init() {
	cobra.OnInitialize()
}

func Run() {
	if err := rootCmd.Execute(); err != nil {
		color.Red(err.Error())
		os.Exit(-1)
	}
}
