package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(updateCmd)
}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "更新一个项目",

	//todo 更具skill的配置文件，更新项目中的skill链接
	//需要对比SHA值，判断skill是否有更新
	//更新方式为直接覆盖原skill文件
	//gskill update更新所有的skill
	//gskill update <skill_name> 只更新指定的skill
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("更新项目命令被调用")
	},
}
