package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/stephenwu2020/ximalaya-dl/dl"
)

var (
	display bool
	output  string
)

var rootCmd = &cobra.Command{
	Use:     "ximalaya-dl url",
	Short:   "Ximalaya Fm downloader",
	Example: example(),
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 1 {
			cmd.Help()
			os.Exit(1)
		}
		rawurl := args[0]
		detail, err := dl.NewAlbumDetail(rawurl)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		if err := detail.Fetch(); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		if output != "" {
			detail.SetOutput(output)
		}

		detail.Display()

		if display {
			return
		}

		if err := detail.DownLoad(); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().BoolVarP(&display, "display", "d", false, "Just display album info.")
	rootCmd.Flags().StringVarP(&output, "output", "o", "", "output dir")
}

func example() string {
	str := `
  download album: ximalaya-dl https://www.ximalaya.com/xiangsheng/39725061	
  download audio: ximalaya-dl https://www.ximalaya.com/xiangsheng/39725061/322739646
	`
	return str
}
