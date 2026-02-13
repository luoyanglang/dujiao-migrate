package migrator

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/luoyanglang/dujiao-migrate/internal/api"
	"github.com/luoyanglang/dujiao-migrate/internal/config"
	"github.com/luoyanglang/dujiao-migrate/internal/database"
	"github.com/luoyanglang/dujiao-migrate/internal/models"
	"github.com/luoyanglang/dujiao-migrate/internal/utils"
)

// Migrator 迁移器
type Migrator struct {
	cfg    *config.Config
	db     *sql.DB
	client *api.Client
	stats  models.Stats
}

// New 创建迁移器
func New(cfg *config.Config) (*Migrator, error) {
	db, err := database.Connect(cfg.OldDB)
	if err != nil {
		return nil, fmt.Errorf("连接老版数据库失败: %w", err)
	}
	log.Println("✓ 老版数据库连接成功")

	client := api.NewClient(cfg.NewAPI.BaseURL, cfg.Options.RetryTimes, cfg.Options.RetryDelay)

	if err := client.Login(cfg.NewAPI.Username, cfg.NewAPI.Password); err != nil {
		db.Close()
		return nil, fmt.Errorf("登录新版后台失败: %w", err)
	}
	log.Println("✓ 新版后台登录成功")

	return &Migrator{
		cfg:    cfg,
		db:     db,
		client: client,
	}, nil
}

// Close 关闭连接
func (m *Migrator) Close() {
	if m.db != nil {
		m.db.Close()
	}
}

// Run 执行迁移
func (m *Migrator) Run() error {
	log.Println(strings.Repeat("=", 50))
	log.Println("独角数卡 数据迁移工具 v1.0.0")
	log.Println("作者: 狼哥")
	log.Println("Telegram: @luoyanglang")
	log.Println("仓库: github.com/luoyanglang/dujiao-migrate")
	log.Println("协议: GPL-3.0")
	log.Println(strings.Repeat("=", 50))

	categoryMap, err := m.migrateCategories()
	if err != nil {
		return fmt.Errorf("迁移分类失败: %w", err)
	}

	productMap, err := m.migrateProducts(categoryMap)
	if err != nil {
		return fmt.Errorf("迁移商品失败: %w", err)
	}

	if m.cfg.Options.MigrateCards {
		if err := m.migrateCards(productMap); err != nil {
			return fmt.Errorf("迁移卡密失败: %w", err)
		}
	}

	m.printSummary()
	return nil
}

// migrateCategories 迁移分类
func (m *Migrator) migrateCategories() (map[int]map[string]interface{}, error) {
	log.Println("\n=== 迁移分类 ===")

	where := "deleted_at IS NULL"
	if m.cfg.Options.OnlyActive {
		where += " AND is_open = 1"
	}

	query := fmt.Sprintf("SELECT id, gp_name, ord, is_open FROM goods_group WHERE %s ORDER BY ord DESC", where)
	rows, err := m.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []models.Category
	for rows.Next() {
		var cat models.Category
		if err := rows.Scan(&cat.ID, &cat.Name, &cat.Ord, &cat.IsOpen); err != nil {
			return nil, err
		}
		categories = append(categories, cat)
	}

	if len(categories) == 0 {
		log.Println("没有需要迁移的分类")
		return make(map[int]map[string]interface{}), nil
	}

	// 获取已存在的分类
	existingItems := make(map[string]int)
	if m.cfg.Options.SkipExisting {
		existingItems, err = m.getExistingItems("/categories")
		if err != nil {
			log.Printf("警告: 获取已存在分类失败: %v", err)
		}
	}

	maxOrd := 0
	for _, cat := range categories {
		if cat.Ord > maxOrd {
			maxOrd = cat.Ord
		}
	}

	categoryMap := make(map[int]map[string]interface{})
	usedSlugs := make(map[string]bool)
	for slug := range existingItems {
		usedSlugs[slug] = true
	}

	for _, cat := range categories {
		slug := utils.Slugify(cat.Name)
		baseSlug := slug

		// 检查是否已存在（跳过）
		if existingID, exists := existingItems[baseSlug]; exists {
			categoryMap[cat.ID] = map[string]interface{}{
				"new_id": existingID,
				"slug":   baseSlug,
			}
			log.Printf("  ⊘ %s 跳过: 已存在 (ID:%d)", cat.Name, existingID)
			m.stats.Categories.Skipped++
			continue
		}

		slug = utils.EnsureUniqueSlug(slug, usedSlugs)

		payload := map[string]interface{}{
			"id": 0,
			"name": map[string]string{
				"zh-CN": cat.Name,
				"zh-TW": "",
				"en-US": "",
			},
			"slug":       slug,
			"sort_order": maxOrd - cat.Ord + 1,
		}

		newID, err := m.createWithSlugRetry("/categories", payload, baseSlug, usedSlugs)
		if err != nil {
			log.Printf("  ✗ %s 失败: %v", cat.Name, err)
			m.stats.Categories.Failed++
			continue
		}

		categoryMap[cat.ID] = map[string]interface{}{
			"new_id": newID,
			"slug":   payload["slug"],
		}
		log.Printf("  ✓ %s (老ID:%d -> 新ID:%d)", cat.Name, cat.ID, newID)
		m.stats.Categories.Success++
	}

	return categoryMap, nil
}

// migrateProducts 迁移商品
func (m *Migrator) migrateProducts(categoryMap map[int]map[string]interface{}) (map[int]map[string]interface{}, error) {
	log.Println("\n=== 迁移商品 ===")

	where := "deleted_at IS NULL"
	if m.cfg.Options.OnlyActive {
		where += " AND is_open = 1"
	}

	query := fmt.Sprintf(`
		SELECT id, group_id, gd_name, gd_description, gd_keywords, 
		       picture, actual_price, in_stock, ord, type, 
		       description, other_ipu_cnf, is_open
		FROM goods WHERE %s ORDER BY ord DESC
	`, where)

	rows, err := m.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var products []models.Product
	for rows.Next() {
		var prod models.Product
		if err := rows.Scan(
			&prod.ID, &prod.GroupID, &prod.Name, &prod.Description, &prod.Keywords,
			&prod.Picture, &prod.ActualPrice, &prod.InStock, &prod.Ord, &prod.Type,
			&prod.Content, &prod.OtherIpuCnf, &prod.IsOpen,
		); err != nil {
			return nil, err
		}
		products = append(products, prod)
	}

	if len(products) == 0 {
		log.Println("没有需要迁移的商品")
		return make(map[int]map[string]interface{}), nil
	}

	existingItems := make(map[string]int)
	if m.cfg.Options.SkipExisting {
		existingItems, err = m.getExistingItems("/products")
		if err != nil {
			log.Printf("警告: 获取已存在商品失败: %v", err)
		}
	}

	productMap := make(map[int]map[string]interface{})
	usedSlugs := make(map[string]bool)
	for slug := range existingItems {
		usedSlugs[slug] = true
	}

	for _, prod := range products {
		catInfo, exists := categoryMap[prod.GroupID]
		if !exists {
			log.Printf("  ⚠ %s 跳过: 分类未迁移", prod.Name)
			m.stats.Products.Skipped++
			continue
		}

		newCategoryID := toInt(catInfo["new_id"])
		slug := utils.Slugify(prod.Name)
		baseSlug := slug

		if existingID, exists := existingItems[baseSlug]; exists {
			productMap[prod.ID] = map[string]interface{}{
				"new_id": existingID,
				"slug":   baseSlug,
			}
			log.Printf("  ⊘ %s 跳过: 已存在 (ID:%d)", prod.Name, existingID)
			m.stats.Products.Skipped++
			continue
		}

		slug = utils.EnsureUniqueSlug(slug, usedSlugs)

		// 处理标签
		tags := []string{}
		if prod.Keywords.Valid {
			for _, tag := range strings.Split(prod.Keywords.String, ",") {
				tag = strings.TrimSpace(tag)
				if tag != "" {
					tags = append(tags, tag)
				}
			}
		}

		// 处理图片
		images := []string{}
		if prod.Picture.Valid && prod.Picture.String != "" {
			newURL := m.uploadImage(prod.Picture.String)
			if newURL != "" {
				images = append(images, newURL)
			}
		}

		// 处理发货类型
		fulfillmentType := "manual"
		if prod.Type == 1 {
			fulfillmentType = "auto"
		}

		// 处理手动发货表单
		manualFormSchema := map[string]interface{}{
			"fields": []interface{}{},
		}
		if prod.OtherIpuCnf.Valid && prod.Type == 2 {
			fields := []interface{}{}
			fieldIndex := 1
			for _, line := range strings.Split(prod.OtherIpuCnf.String, "\n") {
				line = strings.TrimSpace(line)
				if line == "" {
					continue
				}
				parts := strings.Split(line, "|")
				if len(parts) >= 2 {
					fieldType := "text"
					if len(parts) > 3 && parts[3] == "1" {
						fieldType = "textarea"
					}
					required := false
					if len(parts) > 2 && parts[2] == "1" {
						required = true
					}
					field := map[string]interface{}{
						"key":      fmt.Sprintf("field%d", fieldIndex),
						"type":     fieldType,
						"required": required,
						"label": map[string]string{
							"zh-CN": parts[1],
							"zh-TW": "",
							"en-US": "",
						},
					}
					fields = append(fields, field)
					fieldIndex++
				}
			}
			manualFormSchema["fields"] = fields
		}

		manualStockTotal := 0
		if prod.Type == 2 {
			manualStockTotal = prod.InStock
		}

		payload := map[string]interface{}{
			"slug":        slug,
			"category_id": newCategoryID,
			"title": map[string]string{
				"zh-CN": prod.Name,
				"zh-TW": "",
				"en-US": "",
			},
			"description": map[string]string{
				"zh-CN": nullStr(prod.Description),
				"zh-TW": "",
				"en-US": "",
			},
			"content": map[string]string{
				"zh-CN": nullStr(prod.Content),
				"zh-TW": "",
				"en-US": "",
			},
			"fulfillment_type":   fulfillmentType,
			"images":             images,
			"is_active":          true,
			"manual_form_schema": manualFormSchema,
			"manual_stock_total": manualStockTotal,
			"price_amount":       prod.ActualPrice,
			"price_currency":     "CNY",
			"purchase_type":      "guest",
			"sort_order":         prod.Ord,
			"tags":               tags,
		}

		newID, err := m.createWithSlugRetry("/products", payload, baseSlug, usedSlugs)
		if err != nil {
			log.Printf("  ✗ %s 失败: %v", prod.Name, err)
			m.stats.Products.Failed++
			continue
		}

		productMap[prod.ID] = map[string]interface{}{
			"new_id": newID,
			"slug":   payload["slug"],
		}
		log.Printf("  ✓ %s (老ID:%d -> 新ID:%d)", prod.Name, prod.ID, newID)
		m.stats.Products.Success++
	}

	return productMap, nil
}

// migrateCards 迁移卡密
func (m *Migrator) migrateCards(productMap map[int]map[string]interface{}) error {
	log.Println("\n=== 迁移卡密 ===")

	for oldProductID, info := range productMap {
		newProductID := toInt(info["new_id"])

		query := "SELECT carmi FROM carmis WHERE goods_id = ? AND status = 1 AND deleted_at IS NULL"
		rows, err := m.db.Query(query, oldProductID)
		if err != nil {
			log.Printf("  ✗ 商品%d: 查询卡密失败: %v", newProductID, err)
			continue
		}

		var secrets []string
		for rows.Next() {
			var carmi string
			if err := rows.Scan(&carmi); err != nil {
				log.Printf("  ✗ 商品%d: 读取卡密失败: %v", newProductID, err)
				continue
			}
			secrets = append(secrets, carmi)
		}
		rows.Close()

		if len(secrets) == 0 {
			continue
		}

		batchSize := m.cfg.Options.BatchSize
		for i := 0; i < len(secrets); i += batchSize {
			end := i + batchSize
			if end > len(secrets) {
				end = len(secrets)
			}
			batch := secrets[i:end]

			batchNo := fmt.Sprintf("MIGRATE-%s-%d", time.Now().Format("20060102150405"), oldProductID)
			payload := map[string]interface{}{
				"product_id": newProductID,
				"secrets":    batch,
				"batch_no":   batchNo,
				"note":       fmt.Sprintf("从老版迁移 (原商品ID:%d)", oldProductID),
			}

			resp, err := m.client.Post("/card-secrets/batch", payload)
			if err != nil {
				log.Printf("  ✗ 商品%d: 导入失败: %v", newProductID, err)
				m.stats.Cards.Failed += len(batch)
				continue
			}

			if resp.StatusCode != 0 {
				log.Printf("  ✗ 商品%d: 导入失败: %s", newProductID, resp.Msg)
				m.stats.Cards.Failed += len(batch)
				continue
			}

			m.stats.Cards.Success += len(batch)
			log.Printf("  ✓ 商品%d: 导入 %d 条卡密", newProductID, len(batch))
		}
	}

	return nil
}

// createWithSlugRetry 创建资源，slug 冲突时自动加后缀重试
func (m *Migrator) createWithSlugRetry(endpoint string, payload map[string]interface{}, baseSlug string, usedSlugs map[string]bool) (int, error) {
	// 第一次尝试
	resp, err := m.client.Post(endpoint, payload)
	if err != nil {
		return 0, err
	}

	if resp.StatusCode == 0 {
		return extractID(resp)
	}

	// slug 冲突，自动加后缀重试
	for i := 1; i <= 9; i++ {
		retrySlug := fmt.Sprintf("%s-%d", baseSlug, i)
		payload["slug"] = retrySlug

		resp, err = m.client.Post(endpoint, payload)
		if err != nil {
			continue
		}

		if resp.StatusCode == 0 {
			usedSlugs[retrySlug] = true
			return extractID(resp)
		}
	}

	return 0, fmt.Errorf("%s", resp.Msg)
}

// getExistingItems 获取已存在的项目 {slug: id}
func (m *Migrator) getExistingItems(endpoint string) (map[string]int, error) {
	items := make(map[string]int)
	page := 1
	maxPages := 100

	for page <= maxPages {
		resp, err := m.client.Get(fmt.Sprintf("%s?page=%d&page_size=100", endpoint, page))
		if err != nil {
			return items, err
		}

		if resp.StatusCode != 0 {
			break
		}

		// API 返回格式可能是 {data: [...]} 或 {data: {data: [...]}}
		dataList := extractDataList(resp.Data)
		if len(dataList) == 0 {
			break
		}

		beforeCount := len(items)
		for _, item := range dataList {
			itemMap, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			if slug, ok := itemMap["slug"].(string); ok {
				if id, ok := itemMap["id"].(float64); ok {
					items[slug] = int(id)
				}
			}
		}

		if len(items) == beforeCount {
			break
		}

		page++
	}

	return items, nil
}

// printSummary 打印统计信息
func (m *Migrator) printSummary() {
	log.Println("\n" + strings.Repeat("=", 50))
	log.Println("迁移统计")
	log.Println(strings.Repeat("=", 50))
	log.Printf("分类: 成功 %d, 跳过 %d, 失败 %d",
		m.stats.Categories.Success, m.stats.Categories.Skipped, m.stats.Categories.Failed)
	log.Printf("商品: 成功 %d, 跳过 %d, 失败 %d",
		m.stats.Products.Success, m.stats.Products.Skipped, m.stats.Products.Failed)
	log.Printf("卡密: 成功 %d, 失败 %d",
		m.stats.Cards.Success, m.stats.Cards.Failed)
	log.Println(strings.Repeat("=", 50))
}

// --- 辅助函数 ---

// extractID 从 API 响应中提取 ID
func extractID(resp *api.Response) (int, error) {
	dataMap, ok := resp.Data.(map[string]interface{})
	if !ok {
		return 0, fmt.Errorf("响应数据格式错误")
	}

	if id, ok := dataMap["id"].(float64); ok {
		return int(id), nil
	}

	return 0, fmt.Errorf("无法获取新 ID")
}

// extractDataList 从 API 响应中提取数据列表（兼容两种格式）
func extractDataList(data interface{}) []interface{} {
	// 格式1: data 直接是数组
	if list, ok := data.([]interface{}); ok {
		return list
	}

	// 格式2: data 是 map，里面有 data 数组
	if dataMap, ok := data.(map[string]interface{}); ok {
		if list, ok := dataMap["data"].([]interface{}); ok {
			return list
		}
	}

	return nil
}

// toInt 安全地将 interface{} 转为 int
func toInt(v interface{}) int {
	switch val := v.(type) {
	case int:
		return val
	case float64:
		return int(val)
	case int64:
		return int(val)
	default:
		return 0
	}
}

// nullStr 安全地获取 sql.NullString 的值
func nullStr(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return ""
}

// uploadImage 上传图片到新版 API，返回新 URL
// 支持本地文件路径和 HTTP URL
func (m *Migrator) uploadImage(picturePath string) string {
	if picturePath == "" {
		return ""
	}

	oldSitePath := m.cfg.Options.OldSitePath
	if oldSitePath == "" {
		// 没配置老版站点路径，直接返回原始 URL
		return picturePath
	}

	// 如果是完整 URL（http/https），尝试下载后上传
	if strings.HasPrefix(picturePath, "http://") || strings.HasPrefix(picturePath, "https://") {
		// 远程 URL 暂不处理，直接返回
		return picturePath
	}

	// 拼接本地文件路径
	// 老版图片一般在 public/ 目录下
	localPath := picturePath
	if !filepath.IsAbs(picturePath) {
		// 尝试多个可能的路径
		candidates := []string{
			filepath.Join(oldSitePath, "public", picturePath),
			filepath.Join(oldSitePath, picturePath),
			filepath.Join(oldSitePath, "public", "storage", picturePath),
		}
		found := false
		for _, p := range candidates {
			if _, err := os.Stat(p); err == nil {
				localPath = p
				found = true
				break
			}
		}
		if !found {
			log.Printf("    ⚠ 图片文件不存在: %s", picturePath)
			return picturePath
		}
	}

	// 检查文件是否存在
	if _, err := os.Stat(localPath); os.IsNotExist(err) {
		log.Printf("    ⚠ 图片文件不存在: %s", localPath)
		return picturePath
	}

	// 上传到新版 API
	resp, err := m.client.UploadFile(localPath)
	if err != nil {
		log.Printf("    ⚠ 图片上传失败: %v", err)
		return picturePath
	}

	if resp.StatusCode != 0 {
		log.Printf("    ⚠ 图片上传失败: %s", resp.Msg)
		return picturePath
	}

	// 解析返回的 URL
	dataMap, ok := resp.Data.(map[string]interface{})
	if !ok {
		log.Printf("    ⚠ 图片上传响应格式错误")
		return picturePath
	}

	if newURL, ok := dataMap["url"].(string); ok {
		log.Printf("    📷 图片上传成功: %s", newURL)
		return newURL
	}

	return picturePath
}
