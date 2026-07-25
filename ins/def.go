/*
 *@author  chengkenli
 *@project StarRocksProbe
 *@package app
 *@file    def
 *@date    2026/1/8 17:09
 */

package ins

import (
	"StarRocksProbe/util"
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

type availableDates struct {
	Date string `json:"date"`
	URL  string `json:"url"`
}

// 比较不同IP的Variable配置，返回不一致的配置项
func findInconsistentVariables(avgs []variable) []resultb {
	// 提取所有Variable键值对，按key分组记录不同IP的值
	keyMap := make(map[string]map[string]string) // key -> IP -> value

	for _, v := range avgs {
		for _, item := range v.Variable {
			// 解析键值对
			var key, value string
			for i, ch := range item {
				if ch == '=' {
					key = item[:i]
					value = item[i+1:]
					break
				}
			}
			if key == "" {
				continue // 忽略无效格式
			}
			if keyMap[key] == nil {
				keyMap[key] = make(map[string]string)
			}
			keyMap[key][v.IP] = value
		}
	}

	// 检查每个key在不同IP的值是否一致
	var results []resultb
	for key, ipValues := range keyMap {
		// 如果该key在所有IP中不是唯一值，则记录不一致
		uniqueValues := make(map[string]bool)
		for _, val := range ipValues {
			uniqueValues[val] = true
		}
		if len(uniqueValues) > 1 {
			// 对每个IP，记录该key的不一致项
			for ip, val := range ipValues {
				results = append(results, resultb{
					IP:       ip,
					Variable: fmt.Sprintf("%s=%s", key, val),
				})
			}
		}
	}
	return results
}

// 比较不同IP的Variable配置，返回不一致的配置项（包括缺失配置）
func findInconsistentVariables2(avgs []variable) []resultb {
	if len(avgs) == 0 {
		return nil
	}

	// 提取所有IP列表
	allIPs := make([]string, len(avgs))
	for i, v := range avgs {
		allIPs[i] = v.IP
	}

	// 提取所有Variable键值对，按key分组记录不同IP的值
	keyMap := make(map[string]map[string]string) // key -> IP -> value
	allKeys := make(map[string]bool)             // 收集所有出现过的key

	// 首先收集所有IP的所有配置项
	for _, v := range avgs {
		for _, item := range v.Variable {
			// 解析键值对
			var key, value string
			for i, ch := range item {
				if ch == '=' {
					key = item[:i]
					value = item[i+1:]
					break
				}
			}
			if key == "" {
				continue // 忽略无效格式
			}
			if keyMap[key] == nil {
				keyMap[key] = make(map[string]string)
			}
			keyMap[key][v.IP] = value
			allKeys[key] = true
		}
	}

	var results []resultb

	// 检查每个key在不同IP的值是否一致（包括缺失情况）
	for key := range allKeys {
		ipValues := keyMap[key]

		// 检查该key是否在所有IP中都存在
		missingIPs := make([]string, 0)
		for _, ip := range allIPs {
			if _, exists := ipValues[ip]; !exists {
				missingIPs = append(missingIPs, ip)
			}
		}

		// 情况1：key在某些IP中缺失
		if len(missingIPs) > 0 {
			// 记录缺失的IP
			for _, ip := range missingIPs {
				results = append(results, resultb{
					IP:       ip,
					Variable: fmt.Sprintf("%s=<缺失>", key),
				})
			}
			// 记录存在的IP（值可能也不一致）
			for ip, val := range ipValues {
				results = append(results, resultb{
					IP:       ip,
					Variable: fmt.Sprintf("%s=%s", key, val),
				})
			}
			continue
		}

		// 情况2：key在所有IP中都存在，但值不一致
		uniqueValues := make(map[string]bool)
		for _, val := range ipValues {
			uniqueValues[val] = true
		}
		if len(uniqueValues) > 1 {
			for ip, val := range ipValues {
				results = append(results, resultb{
					IP:       ip,
					Variable: fmt.Sprintf("%s=%s", key, val),
				})
			}
		}
	}

	return results
}

func rmSlicevari(vars []variable, target string) []variable {
	// 创建一个新的切片，避免修改原始数据
	result := make([]variable, len(vars))

	for i, v := range vars {
		// 复制IP字段
		result[i].IP = v.IP

		// 创建一个新的Variable切片，用于存储不包含目标字符串的项
		newVariables := make([]string, 0, len(v.Variable))
		for _, item := range v.Variable {
			if !strings.Contains(item, target) {
				newVariables = append(newVariables, item)
			}
		}
		result[i].Variable = newVariables
	}
	return result
}

func rmSliceResultb(data []resultb) []resultb {
	// 使用map来记录已经出现过的组合
	seen := make(map[string]bool)
	result := make([]resultb, 0)

	for _, item := range data {
		// 创建唯一标识符：IP + Variable
		key := item.IP + "|" + item.Variable

		// 如果这个组合还没有出现过，就添加到结果中
		if !seen[key] {
			seen[key] = true
			result = append(result, item)
		}
	}
	return result
}

// 去重Variables参数，仅保留唯一值
func rmResultVariables(data []resultb) []resultb {
	// 使用map来记录已经出现过的Variable键（等号左边的部分）
	seen := make(map[string]bool)
	result := make([]resultb, 0)

	for _, item := range data {
		// 以等号为分隔符，获取Variable的键（左边部分）
		parts := strings.Split(item.Variable, "=")
		if len(parts) == 0 {
			continue
		}
		key := parts[0] // 等号左边的部分
		// 如果这个键还没有出现过，就添加到结果中
		if !seen[key] {
			seen[key] = true
			result = append(result, item)
		}
	}
	return result
}

// 诊断风险，并返回”正常“或风险，指标
func diagnMark(bbodys [][]string) [][]string {
	if bbodys == nil {
		bbodys = [][]string{{`<span class="status status-ok">正常</span>`}}
	} else {
		bbodys = append(bbodys, []string{`<span class="status status-warning">风险</span>`})
	}
	return bbodys
}

// 匹配[][]string二维数据的不同点
func findDifferences(data [][]string) [][]string {
	// 按 IP 分组，key 是 IP，value 是该 IP 对应的所有配置行（去掉 IP 部分）
	ipConfigs := make(map[string][]string)
	// 记录每个 IP 的完整数据（用于最后输出）
	ipFullData := make(map[string][][]string)

	// 遍历数据，按 IP 分组
	currentIP := ""
	for _, row := range data {
		ip := row[0]
		config := row[1]

		// 如果 IP 不为空，更新当前 IP
		if ip != "" {
			currentIP = ip
		}
		// 保存配置行（去掉 IP 部分）
		ipConfigs[currentIP] = append(ipConfigs[currentIP], config)
		// 保存完整数据
		ipFullData[currentIP] = append(ipFullData[currentIP], []string{ip, config})
	}
	// 统计每个配置的出现次数
	configCount := make(map[string]int)
	for _, configs := range ipConfigs {
		// 将配置切片合并为一个字符串，用于比较
		configKey := strings.Join(configs, "|")
		configCount[configKey]++
	}
	// 找出出现次数最多的配置（作为基准配置）
	var maxConfig string
	maxCount := 0
	for config, count := range configCount {
		if count > maxCount {
			maxCount = count
			maxConfig = config
		}
	}
	// 收集与基准配置不同的 IP 的完整数据
	var result [][]string
	for ip, configs := range ipConfigs {
		configKey := strings.Join(configs, "|")
		if configKey != maxConfig {
			// 将该 IP 的所有数据添加到结果中
			for _, row := range ipFullData[ip] {
				result = append(result, row)
			}
		}
	}
	return result
}

func selectids() []string {
	var sids []string
	for _, m := range util.MetaLink {
		sids = append(sids, fmt.Sprintf(`<option value="/%s">%s</option>`, m["app"].(string), m["app"].(string)))
	}
	return sids
}

// SliceUnique
// 二维数组去重
func SliceUnique(slice [][]string) [][]string {
	seen := make(map[string]bool)
	var result [][]string

	for _, inner := range slice {
		key := strings.Join(inner, "|") // 选择不会出现在元素中的分隔符
		if !seen[key] {
			seen[key] = true
			result = append(result, inner)
		}
	}
	return result
}

// 遍历某个目录下所有的文件
func findDir(dir string) []string {
	var htmlFiles []string
	entries, err := os.ReadDir(dir)
	if err != nil {
		util.Loggrs.Error(err.Error())
		return nil
	}
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(strings.ToLower(entry.Name()), ".html") {
			htmlFiles = append(htmlFiles, entry.Name())
		}
	}
	return htmlFiles
}

// 数据元素中提取日期
func regexDates(files []string) []string {
	re := regexp.MustCompile(`_(\d{8})_`)
	dateMap := make(map[string]bool)
	var uniqueDates []string

	for _, file := range files {
		matches := re.FindStringSubmatch(file)
		if len(matches) > 1 {
			date := matches[1]
			if !dateMap[date] {
				dateMap[date] = true
				uniqueDates = append(uniqueDates, date)
			}
		}
	}
	return uniqueDates
}

// 获取所有的历史接口，并设置对应关系
func getreplacements(appname, date string) []map[string]string {
	var ments []map[string]string
	for _, m := range util.MetaLink {
		app := m["app"].(string)
		ment := map[string]string{
			fmt.Sprintf("/%s", app):         fmt.Sprintf("/%s/%s", app, date),
			fmt.Sprintf("/%s/olap", app):    fmt.Sprintf("/%s/%s/olap", app, date),
			fmt.Sprintf("/%s/avgs", app):    fmt.Sprintf("/%s/%s/avgs", app, date),
			fmt.Sprintf("/%s/conf", app):    fmt.Sprintf("/%s/%s/conf", app, date),
			fmt.Sprintf("/%s/log", app):     fmt.Sprintf("/%s/%s/log", app, date),
			fmt.Sprintf("/%s/monitor", app): fmt.Sprintf("/%s/%s/monitor", app, date),
			fmt.Sprintf("/%s/node", app):    fmt.Sprintf("/%s/%s/node", app, date),
			fmt.Sprintf("<!-- 返回当天标签 -->"): fmt.Sprintf(`<div style="position: relative; margin-top: 20px;">
            <a href="%s" style="position: absolute; left: 0; bottom: 5;color: white;">« 返回当天</a>
        </div>`, fmt.Sprintf("%s:%d/%s", util.H.Ip, util.Read.Server.Port, appname)),
		}
		ments = append(ments, ment)
	}
	return ments
}

// 专门处理HTML属性的精准替换
func preciseHTMLReplace(content, oldStr, newStr string) string {
	// 匹配HTML属性中的路径，如 href="/sr-app"
	patterns := []string{
		// href属性
		fmt.Sprintf(`(href\s*=\s*["'])%s(["'])`, regexp.QuoteMeta(oldStr)),
		// src属性
		fmt.Sprintf(`(src\s*=\s*["'])%s(["'])`, regexp.QuoteMeta(oldStr)),
		// action属性
		fmt.Sprintf(`(action\s*=\s*["'])%s(["'])`, regexp.QuoteMeta(oldStr)),
		// option value属性
		fmt.Sprintf(`(value\s*=\s*["'])%s(["'])`, regexp.QuoteMeta(oldStr)),
		// 匹配 HTML 注释
		`<!-- 返回当天标签 -->`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		content = re.ReplaceAllString(content, "${1}"+newStr+"${2}")
	}
	return content
}

// 使用HTML精准替换的版本
func replacehtml(filePath string, replacementsBody []map[string]string) {
	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		util.Loggrs.Error(err.Error())
		return
	}
	originalContent := string(content)
	newContent := originalContent
	//util.Loggrs.Infof("开始处理文件: %s", filePath)
	totalReplacements := 0
	for _, replacements := range replacementsBody {
		var keys []string
		for k := range replacements {
			keys = append(keys, k)
		}

		sort.Slice(keys, func(i, j int) bool {
			return len(keys[i]) > len(keys[j])
		})

		for _, oldStr := range keys {
			newStr := replacements[oldStr]

			before := newContent
			newContent = preciseHTMLReplace(newContent, oldStr, newStr)

			if before != newContent {
				totalReplacements++
				util.Loggrs.Infof("%s 替换: %s -> %s", filepath.Base(filePath), oldStr, newStr)
			}
		}
	}
	if originalContent != newContent {
		err = ioutil.WriteFile(filePath, []byte(newContent), 0644)
		if err != nil {
			util.Loggrs.Errorf("写入文件失败: %s, error: %v", filePath, err)
		} else {
			util.Loggrs.Infof("文件处理完成: %s, 共执行 %d 次替换", filePath, totalReplacements)
		}
	} else {
		util.Loggrs.Infof("文件无需修改: %s", filePath)
	}
}

// 获取当前服务启动进程的用户
func curritem() string {
	if util.Read.Server.Sshuser != "" {
		return util.Read.Server.Sshuser
	}
	current, err := user.Current()
	if err != nil {
		util.Loggrs.Error(err.Error())
		return "starrocks"
	}
	return current.Username
}
