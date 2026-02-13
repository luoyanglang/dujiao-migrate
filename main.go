package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/luoyanglang/dujiao-migrate/internal/config"
	"github.com/luoyanglang/dujiao-migrate/internal/migrator"
)

const version = "1.0.0"

func main() {
	// 命令行参数
	configFile := flag.String("config", "", "配置文件路径")
	generateConfig := flag.Bool("generate-config", false, "生成示例配置文件")
	showVersion := flag.Bool("version", false, "显示版本信息")

	// 老版数据库参数
	oldHost := flag.String("old-host", "", "老版数据库主机")
	oldPort := flag.Int("old-port", 0, "老版数据库端口")
	oldUser := flag.String("old-user", "", "老版数据库用户名")
	oldPassword := flag.String("old-password", "", "老版数据库密码")
	oldDatabase := flag.String("old-database", "", "老版数据库名")
	oldDriver := flag.String("old-driver", "mysql", "老版数据库驱动 (mysql/postgres/sqlite)")

	// 新版 API 参数
	newAPI := flag.String("new-api", "", "新版 API 地址")
	newUser := flag.String("new-user", "", "新版管理员用户名")
	newPassword := flag.String("new-password", "", "新版管理员密码")

	// 选项
	noSkip := flag.Bool("no-skip", false, "不跳过已存在的数据")
	noCards := flag.Bool("no-cards", false, "不迁移卡密")
	oldSitePath := flag.String("old-site-path", "", "老版站点路径（用于图片迁移）")

	flag.Parse()

	// 显示版本
	if *showVersion {
		fmt.Printf("独角数卡迁移工具 v%s\n", version)
		return
	}

	// 生成示例配置
	if *generateConfig {
		config.GenerateSampleConfig()
		return
	}

	// 加载配置
	cfg, err := config.LoadConfig(*configFile, &config.CLIArgs{
		OldHost:     *oldHost,
		OldPort:     *oldPort,
		OldUser:     *oldUser,
		OldPassword: *oldPassword,
		OldDatabase: *oldDatabase,
		OldDriver:   *oldDriver,
		NewAPI:      *newAPI,
		NewUser:     *newUser,
		NewPassword: *newPassword,
		NoSkip:      *noSkip,
		NoCards:     *noCards,
		OldSitePath: *oldSitePath,
	})
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 创建迁移器
	m, err := migrator.New(cfg)
	if err != nil {
		log.Fatalf("创建迁移器失败: %v", err)
	}
	defer m.Close()

	// 执行迁移
	if err := m.Run(); err != nil {
		log.Fatalf("迁移失败: %v", err)
	}

	log.Println("迁移完成！")
}
