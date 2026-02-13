package utils

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/mozillazg/go-pinyin"
)

var pinyinArgs pinyin.Args

func init() {
	pinyinArgs = pinyin.NewArgs()
	pinyinArgs.Style = pinyin.Normal // 不带声调
}

// Slugify 生成 slug，中文自动转拼音
func Slugify(text string) string {
	// 如果包含中文，先转拼音
	if ContainsChinese(text) {
		parts := pinyin.LazyPinyin(text, pinyinArgs)
		if len(parts) > 0 {
			text = strings.Join(parts, "-")
		}
	}

	// 替换非字母数字为连字符
	reg := regexp.MustCompile(`[^a-zA-Z0-9]+`)
	slug := reg.ReplaceAllString(text, "-")

	// 去除首尾连字符并转小写
	slug = strings.Trim(slug, "-")
	slug = strings.ToLower(slug)

	// 如果为空，生成时间戳 slug
	if slug == "" {
		slug = fmt.Sprintf("item-%s", time.Now().Format("20060102150405"))
	}

	// 限制长度
	if len(slug) > 50 {
		slug = slug[:50]
		// 避免截断在连字符处
		slug = strings.TrimRight(slug, "-")
	}

	return slug
}

// EnsureUniqueSlug 确保 slug 唯一
func EnsureUniqueSlug(slug string, usedSlugs map[string]bool) string {
	baseSlug := slug
	counter := 1

	for usedSlugs[slug] {
		slug = fmt.Sprintf("%s-%d", baseSlug, counter)
		counter++
	}

	usedSlugs[slug] = true
	return slug
}

// ContainsChinese 检测字符串是否包含中文
func ContainsChinese(s string) bool {
	for _, r := range s {
		if r >= 0x4e00 && r <= 0x9fff {
			return true
		}
	}
	return false
}
