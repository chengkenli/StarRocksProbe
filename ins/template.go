/*
 *@author  chengkenli
 *@project StarRocksProbe
 *@package module
 *@file    module_template
 *@date    2026/1/7 16:38
 */

package ins

import (
	"fmt"
	"strings"
	"time"
)

type statictemplates struct {
	App           string
	Title         string
	Version       string
	Body          string
	Script        string
	Css           string
	Select        string
	AvaildateJson string
	Watermark     string
}

func bodyTemplates(t statictemplates) string {
	body := fmt.Sprintf(`
<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
	<script src="../istatic/chart.js"></script>
	<script src="../istatic/chartjs-adapter-date-fns.bundle.min.js"></script>
    <title>%s</title>
	<link rel="icon" type="image/x-icon" href="../istatic/favicon.ico" />
	%v
    <link rel="stylesheet" href="../istatic/ui-node.css">
	<link rel="stylesheet" href="../istatic/ui-date.css">
</head>
<body>
	<header>
		<!-- 返回概览标签 -->
        <div style="position: relative; margin-top: 20px;">
            <a href="%s" style="position: absolute; left: 0; bottom: 0;color: white;">« 返回概览</a>
        </div>
		<!-- 返回当天标签 -->
    </header>

	<div class="watermark-container" id="watermark"></div>
    <div class="container">
        <header>
            <h1>%s</h1>
        </header>
        
        <div class="report-info">
            <div><strong>巡检日期：</strong>%s</div>
            <div><strong>集群版本：</strong>%s</div>
        </div>

        <div class="content">
            %v
        </div>
        
        <footer>
            <p>StarRocks%s报告 | 巡检时间: %s</p>
            <p>本报告来自Infra Data - StarRocks</p>
        </footer>
    </div>

	<script>
        // 动态创建重复水印
        function createWatermark() {
            const container = document.getElementById('watermark');
            const text = '%s';
            const spacing = 200; // 水印间距
            
            for (let x = 0; x < window.innerWidth; x += spacing) {
                for (let y = 0; y < window.innerHeight; y += spacing) {
                    const watermark = document.createElement('div');
                    watermark.className = 'watermark';
                    watermark.textContent = text;
                    watermark.style.left = x + 'px';
                    watermark.style.top = y + 'px';
                    container.appendChild(watermark);
                }
            }
        }
        createWatermark();
    </script>
	<script>
        // 添加一些交互效果
        document.addEventListener('DOMContentLoaded', function() {
            const statusElements = document.querySelectorAll('.status');
            
            statusElements.forEach(status => {
                status.addEventListener('click', function() {
                    const statusText = this.textContent;
                });
            });
            
            // 添加按钮点击效果
            const buttons = document.querySelectorAll('.action-btn');
            buttons.forEach(button => {
                button.addEventListener('click', function() {
                    this.style.transform = 'scale(0.95)';
                    setTimeout(() => {
                        this.style.transform = '';
                    }, 150);
                });
            });
        });
    </script>
	%v
	%v
</body>
</html>`, t.Title, t.Css, fmt.Sprintf("%s.html", t.App), t.Title, time.Now().Format("2006-01-02"), t.Version,
		t.Body, t.Title, time.Now().Format("2006-01-02 15:04:05"), t.Watermark, datet(t.AvaildateJson), t.Script)
	return body
}

type conclusion struct {
	Score      int    //总评分
	ScoreLevel string //评分等级，卓越，优秀，良好，一般，差
	ScoreConcl string //结论
}

// Conclusion
// 巡检结论
func bodyConclusion(c conclusion) string {
	return fmt.Sprintf(`<div class="section">
        <h2 class="section-title">巡检总结</h2>
        <div class="overview">
            <div class="score-card">
                <div class="score-circle">
                    <div class="score-value">%d</div>
                </div>
                <div class="score-label">%s</div>
            </div>
            <!-- 巡检总结放在评分后面 -->
            <div class="conclusion">
                <h2><i class="fas fa-clipboard-check"></i>巡检结论</h2>
                <p class="conclusion-text">%s</p>
            </div>
        </div>
    </div>`, c.Score, c.ScoreLevel, c.ScoreConcl)
}

type indicator struct {
	Title3   string      //标题h3
	Value    interface{} //value
	Subvalue string      //sub-value
}

// IndicatorLast
// 后执行
// 显示指标
func bodyIndicatorLast(title2, subindi string) string {
	b2 := fmt.Sprintf(`<div class="section">
        <h2 class="section-title">%s</h2>
        <div class="summary-cards">
			%v
        </div>
    </div>`, title2, subindi)
	return b2
}

// IndicatorFirst
// 先执行
// 显示指标
func bodyIndicatorFirst(s indicator) string {
	b1 := fmt.Sprintf(`<div class="card">
        <h3>%s</h3>
        <div class="value">%v</div>
        <div class="sub-value">%s</div>
    </div>`, s.Title3, s.Value, s.Subvalue)
	return b1
}

type subject struct {
	Title2  string
	Thead   []string
	Tbody   [][]string
	Comment string
}

// Subject
// 明细内容
func bodySubject(s subject) string {
	var title string
	if s.Comment != "" {
		title = s.Comment
	} else {
		title = fmt.Sprintf(`<h2 class="section-title">%s</h2>`, s.Title2)
	}
	return fmt.Sprintf(`<div class="section">
         %s
         <table>
             <thead>
                 %s
             </thead>
             <tbody>
                 %s
             </tbody>
         </table>
     </div>`, title, subject2Sh(s.Thead), subject2td(s.Tbody))
}

// Subject2Sh
// th
func subject2Sh(ths []string) string {
	var th2 []string
	for _, th := range ths {
		th2 = append(th2, fmt.Sprintf("<th>%s</th>", th))
	}
	return fmt.Sprintf("<tr>%s</tr>", strings.Join(th2, "\n"))
}

// Subject2td
// td
func subject2td(tds [][]string) string {
	var trs []string
	for _, td := range tds {
		var td2 []string
		for _, sd := range td {
			td2 = append(td2, fmt.Sprintf("<td>%s</td>", sd))
		}
		trs = append(trs, fmt.Sprintf("<tr>%s</tr>", strings.Join(td2, "\n")))
	}
	return strings.Join(trs, "\n")
}

func datet(availdateJson string) string {
	return fmt.Sprintf("    <script>\n        // 预设的日期列表\n        const availableDates = %v;\n        \n        const buttonContainer = document.getElementById('buttonContainer');\n        const draggableButton = document.getElementById('draggableButton');\n        const datePicker = document.getElementById('datePicker');\n        \n        // 拖动功能\n        let isDragging = false;\n        let offsetX, offsetY;\n        \n        draggableButton.addEventListener('mousedown', function(e) {\n            // 防止拖动时触发点击事件\n            e.stopPropagation();\n            isDragging = true;\n            offsetX = e.clientX - buttonContainer.getBoundingClientRect().left;\n            offsetY = e.clientY - buttonContainer.getBoundingClientRect().top;\n            draggableButton.style.cursor = 'grabbing';\n        });\n        \n        document.addEventListener('mousemove', function(e) {\n            if (!isDragging) return;\n            \n            // 计算新位置\n            let newX = e.clientX - offsetX;\n            let newY = e.clientY - offsetY;\n            \n            // 限制在视口范围内\n            const maxX = window.innerWidth - buttonContainer.offsetWidth;\n            const maxY = window.innerHeight - buttonContainer.offsetHeight;\n            \n            newX = Math.max(0, Math.min(newX, maxX));\n            newY = Math.max(0, Math.min(newY, maxY));\n            \n            // 应用新位置\n            buttonContainer.style.left = newX + 'px';\n            buttonContainer.style.top = newY + 'px';\n        });\n        \n        document.addEventListener('mouseup', function() {\n            isDragging = false;\n            draggableButton.style.cursor = 'move';\n        });\n        \n        // 点击按钮显示日期选择器\n        draggableButton.addEventListener('click', function(e) {\n            // 防止拖动时触发点击事件\n            if (isDragging) return;\n            e.stopPropagation();\n            \n            if (datePicker.style.display === 'block') {\n                datePicker.style.display = 'none';\n            } else {\n                // 清空日期选择器内容\n                datePicker.innerHTML = '';\n                \n                // 创建自定义日期选择器\n                const customDatepicker = document.createElement('div');\n                customDatepicker.className = 'custom-datepicker';\n                \n                // 创建日期输入框\n                const dateInput = document.createElement('input');\n                dateInput.type = 'date';\n                dateInput.className = 'custom-date-input';\n                dateInput.id = 'customDateInput';\n                \n                // 设置最小和最大日期（可选，这里设置为当前年份的前后5年）\n                const currentYear = new Date().getFullYear();\n                dateInput.min = `${currentYear-5}-01-01`;\n                dateInput.max = `${currentYear+5}-12-31`;\n                \n                customDatepicker.appendChild(dateInput);\n                \n                // 创建日期选择器容器\n                const datepickerContainer = document.createElement('div');\n                datepickerContainer.id = 'datepickerContainer';\n                customDatepicker.appendChild(datepickerContainer);\n                \n                // 监听日期输入框变化\n                dateInput.addEventListener('change', function() {\n                    renderCustomDatepicker(dateInput.value, datepickerContainer);\n                });\n                \n                // 初始渲染（使用当前日期）\n                renderCustomDatepicker(new Date().toISOString().split('T')[0], datepickerContainer);\n                \n                datePicker.appendChild(customDatepicker);\n                datePicker.style.display = 'block';\n            }\n        });\n        \n        // 渲染自定义日期选择器\n        function renderCustomDatepicker(selectedDate, container) {\n            container.innerHTML = '';\n            \n            const date = new Date(selectedDate);\n            const year = date.getFullYear();\n            const month = date.getMonth();\n            \n            // 创建月份和年份选择器\n            const monthYearHeader = document.createElement('div');\n            monthYearHeader.style.textAlign = 'center';\n            monthYearHeader.style.marginBottom = '10px';\n            monthYearHeader.style.fontWeight = 'bold';\n            monthYearHeader.textContent = `${year}年${month+1}月`;\n            container.appendChild(monthYearHeader);\n            \n            // 创建星期标题\n            const weekdays = ['日', '一', '二', '三', '四', '五', '六'];\n            const weekdaysRow = document.createElement('div');\n            weekdaysRow.style.display = 'flex';\n            weekdaysRow.style.justifyContent = 'space-around';\n            weekdaysRow.style.marginBottom = '5px';\n            \n            weekdays.forEach(day => {\n                const dayElement = document.createElement('div');\n                dayElement.textContent = day;\n                dayElement.style.width = '30px';\n                dayElement.style.textAlign = 'center';\n                dayElement.style.fontWeight = 'bold';\n                weekdaysRow.appendChild(dayElement);\n            });\n            \n            container.appendChild(weekdaysRow);\n            \n            // 创建日期网格\n            const firstDay = new Date(year, month, 1);\n            const lastDay = new Date(year, month + 1, 0);\n            const daysInMonth = lastDay.getDate();\n            const startingDay = firstDay.getDay();\n            \n            const datesGrid = document.createElement('div');\n            datesGrid.style.display = 'grid';\n            datesGrid.style.gridTemplateColumns = 'repeat(7, 1fr)';\n            datesGrid.style.gap = '5px';\n            \n            // 添加空白单元格\n            for (let i = 0; i < startingDay; i++) {\n                const emptyCell = document.createElement('div');\n                emptyCell.style.height = '30px';\n                datesGrid.appendChild(emptyCell);\n            }\n            \n            // 添加日期单元格\n            for (let day = 1; day <= daysInMonth; day++) {\n                const dayElement = document.createElement('div');\n                dayElement.className = 'datepicker-day';\n                dayElement.textContent = day;\n                dayElement.style.height = '30px';\n                dayElement.style.display = 'flex';\n                dayElement.style.alignItems = 'center';\n                dayElement.style.justifyContent = 'center';\n                dayElement.style.borderRadius = '4px';\n                dayElement.style.cursor = 'pointer';\n                \n                // 格式化日期为YYYYMMDD\n                const formattedDate = `${year}${String(month+1).padStart(2, '0')}${String(day).padStart(2, '0')}`;\n                \n                // 检查日期是否在可用日期列表中\n                const isAvailable = availableDates.some(d => d.date === formattedDate);\n                \n                if (isAvailable) {\n                    dayElement.classList.add('available');\n                    dayElement.addEventListener('click', function() {\n                        const selectedDateObj = availableDates.find(d => d.date === formattedDate);\n                        if (selectedDateObj) {\n                            // 在实际应用中，这里会进行页面跳转\n                            window.location.href = selectedDateObj.url;\n                           // 隐藏日期选择器\n                            datePicker.style.display = 'none';\n                        }\n                    });\n                } else {\n                    dayElement.classList.add('disabled');\n                    dayElement.style.cursor = 'not-allowed';\n                }\n                \n                datesGrid.appendChild(dayElement);\n            }\n            \n            container.appendChild(datesGrid);\n        }\n        \n        // 点击页面其他地方关闭日期选择器\n        document.addEventListener('click', function(event) {\n            if (!buttonContainer.contains(event.target)) {\n                datePicker.style.display = 'none';\n            }\n        });\n        \n        // 初始位置\n        buttonContainer.style.left = '50px';\n        buttonContainer.style.top = '50px';\n    </script>", availdateJson)
}
