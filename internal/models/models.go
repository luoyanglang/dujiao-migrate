package models

import "database/sql"

// Category 分类
type Category struct {
	ID     int
	Name   string
	Ord    int
	IsOpen int
}

// Product 商品
type Product struct {
	ID            int
	GroupID       int
	Name          string
	Description   sql.NullString
	Keywords      sql.NullString
	Picture       sql.NullString
	ActualPrice   float64
	InStock       int
	Ord           int
	Type          int
	Content       sql.NullString
	OtherIpuCnf   sql.NullString
	IsOpen        int
}

// Card 卡密
type Card struct {
	Carmi string
}

// Stats 统计信息
type Stats struct {
	Categories CategoryStats
	Products   ProductStats
	Cards      CardStats
}

// CategoryStats 分类统计
type CategoryStats struct {
	Success int
	Skipped int
	Failed  int
}

// ProductStats 商品统计
type ProductStats struct {
	Success int
	Skipped int
	Failed  int
}

// CardStats 卡密统计
type CardStats struct {
	Success int
	Failed  int
}
